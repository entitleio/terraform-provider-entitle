//go:build acceptance

package agentTokens_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/provider"
)

const (
	stubAddr       = "entitle_agent_token.stub"
	stubName       = "Stub Agent Token"
	stubTokenID    = "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	stubTokenValue = "eyJzdHViIjp0cnVlfQ=="

	// The body the backend's FeatureFlagGuard returns when the rotation flag
	// is disabled, which is the default state for a tenant.
	flagDisabledBody = `{"errorId":"request.unauthorized",` +
		`"message":"This endpoint is not available because the required feature flag is not enabled."}`

	// A deleted record and an unregistered route both come back as
	// resource.notFound, which is why the provider must not treat either as
	// grounds for dropping the resource from state.
	notFoundBody = `{"errorId":"resource.notFound","message":"Token not found"}`
)

// stubResponses configures the stub's replies. Zero values mean "behave
// normally", so each test only states the one failure it is about.
//
// It is separate from stubAPI so the handler can take a copy under the mutex
// without copying a lock, which go vet's copylocks check would reject.
type stubResponses struct {
	createBody   string
	readBody     string
	putStatus    int
	putBody      string
	rotateStatus int
	rotateBody   string
}

// stubAPI is a minimal stand-in for the Entitle API: enough to create and read
// an agent token, and to fail any one operation on demand.
//
// It exists because the rotate error paths are unreachable against the real
// API. A tenant either has the feature flag or it does not, and a 404 needs the
// record to vanish between the refresh and the apply. The provider's endpoint
// attribute is validated against an allowlist of production URLs, but Configure
// reads ENTITLE_API_ENDPOINT before it looks at the attribute and does not
// validate the environment variable — that is the seam these tests use, so no
// production code has to grow a test hook.
type stubAPI struct {
	mu   sync.Mutex
	name string
	resp stubResponses
}

func newStubAPI(resp stubResponses) *stubAPI {
	return &stubAPI{resp: resp}
}

func (s *stubAPI) responses() stubResponses {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.resp
}

// setReadBody swaps the read response mid-test, from a TestStep PreConfig.
func (s *stubAPI) setReadBody(body string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resp.readBody = body
}

func (s *stubAPI) currentName() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.name
}

func (s *stubAPI) setName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.name = name
}

func (s *stubAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const base = "/public/v1/agentTokens"

	item := base + "/" + stubTokenID
	resp := s.responses()

	switch {
	case r.Method == http.MethodPost && r.URL.Path == base:
		s.setName(decodeName(r))

		if resp.createBody != "" {
			writeJSON(w, http.StatusOK, resp.createBody)
			return
		}

		writeJSON(w, http.StatusOK, fmt.Sprintf(
			`{"result":{"id":%q,"name":%q,"token":%q}}`, stubTokenID, s.currentName(), stubTokenValue))

	case r.Method == http.MethodGet && r.URL.Path == item:
		if resp.readBody != "" {
			writeJSON(w, http.StatusOK, resp.readBody)
			return
		}

		writeJSON(w, http.StatusOK, fmt.Sprintf(
			`{"result":{"id":%q,"name":%q}}`, stubTokenID, s.currentName()))

	case r.Method == http.MethodPut && r.URL.Path == item:
		if resp.putStatus != 0 && resp.putStatus != http.StatusOK {
			// A failed rename must not take effect server-side.
			writeJSON(w, resp.putStatus, orDefault(resp.putBody, notFoundBody))
			return
		}

		s.setName(decodeName(r))

		writeJSON(w, http.StatusOK, fmt.Sprintf(
			`{"result":{"id":%q,"name":%q}}`, stubTokenID, s.currentName()))

	case r.Method == http.MethodPost && r.URL.Path == item+"/rotate":
		status := resp.rotateStatus
		if status == 0 {
			status = http.StatusOK
		}

		body := resp.rotateBody
		if body == "" {
			body = fmt.Sprintf(
				`{"result":{"id":%q,"name":%q,"token":%q}}`, stubTokenID, s.currentName(), stubTokenValue+"rotated")
		}

		writeJSON(w, status, body)

	case r.Method == http.MethodDelete && r.URL.Path == item:
		writeJSON(w, http.StatusOK, `{"ok":true}`)

	default:
		// Mirrors main-api: an unregistered route is rewritten into the same
		// resource.notFound shape as a missing record.
		writeJSON(w, http.StatusNotFound, fmt.Sprintf(
			`{"errorId":"resource.notFound","message":"Cannot %s %s"}`, r.Method, r.URL.Path))
	}
}

func decodeName(r *http.Request) string {
	var body struct {
		Name string `json:"name"`
	}

	_ = json.NewDecoder(r.Body).Decode(&body)

	return body.Name
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = w.Write([]byte(body))
}

// startStubAPI points the provider at a stub server for the duration of the
// test. The provider block in stubConfig deliberately omits "endpoint" so the
// environment variable wins.
func startStubAPI(t *testing.T, api *stubAPI) {
	t.Helper()

	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	t.Setenv("ENTITLE_API_ENDPOINT", srv.URL)
	t.Setenv("ENTITLE_API_KEY", "stub-api-key")
}

func stubProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// wrapped builds an ExpectError pattern that tolerates Terraform's diagnostic
// word wrapping. Terraform rewraps detail text to the terminal width and
// indents continuation lines, so a multi-word phrase is liable to arrive with a
// newline and leading spaces somewhere in the middle of it. Matching each word
// separated by \s+ survives that; a plain literal does not.
func wrapped(phrase string) *regexp.Regexp {
	return regexp.MustCompile(strings.Join(strings.Fields(regexp.QuoteMeta(phrase)), `\s+`))
}

func stubConfig(rotation string) string {
	return stubConfigNamed(stubName, rotation)
}

func stubConfigNamed(name, rotation string) string {
	return fmt.Sprintf(`
provider "entitle" {}

resource "entitle_agent_token" "stub" {
	name     = %q
	rotation = %q
}
`, name, rotation)
}

// TestAgentTokenRotationFeatureFlagDisabled covers the default state of a
// tenant: the rotate endpoint is gated by "enableAgentTokenRotation" and its
// guard answers 401, which HTTPResponseToError would otherwise report as
// "unauthorized token: update the entitle token and retry please" — sending the
// operator to replace credentials that are working fine.
func TestAgentTokenRotationFeatureFlagDisabled(t *testing.T) {
	api := newStubAPI(stubResponses{
		rotateStatus: http.StatusUnauthorized,
		rotateBody:   flagDisabledBody,
	})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: stubConfig("1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(stubAddr, "id", stubTokenID),
					resource.TestCheckResourceAttr(stubAddr, "token", stubTokenValue),
				),
			},
			{
				Config:      stubConfig("2"),
				ExpectError: wrapped("enableAgentTokenRotation"),
			},
			// Proves the API's own text is carried through, which is what lets
			// an operator tell a disabled flag from bad credentials.
			{
				Config:      stubConfig("3"),
				ExpectError: wrapped("required feature flag is not enabled"),
			},
		},
	})
}

// TestAgentTokenRotationNotFoundKeepsState covers the 404 path on rotate. A 404
// is ambiguous — deleted token, unregistered route, or a provider released
// ahead of the API — so the resource must stay in state rather than be silently
// dropped and recreated, which would orphan a live token.
func TestAgentTokenRotationNotFoundKeepsState(t *testing.T) {
	api := newStubAPI(stubResponses{
		rotateStatus: http.StatusNotFound,
		rotateBody:   notFoundBody,
	})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: stubConfig("1"),
				Check:  resource.TestCheckResourceAttr(stubAddr, "id", stubTokenID),
			},
			{
				Config:      stubConfig("2"),
				ExpectError: wrapped("The existing token is unchanged"),
			},
			// The failed rotate did not persist rotation, so state still holds
			// "1" and this config matches it exactly. An empty plan is only
			// possible if the resource survived in state: had the 404 dropped
			// it, this would plan a create and PlanOnly would fail on the
			// non-empty plan. That is the assertion.
			{
				Config:   stubConfig("1"),
				PlanOnly: true,
			},
		},
	})
}

// TestAgentTokenRenameNotFoundKeepsState covers the same 404 rule on the rename
// path, which previously removed the resource from state inside Update.
func TestAgentTokenRenameNotFoundKeepsState(t *testing.T) {
	api := newStubAPI(stubResponses{
		putStatus: http.StatusNotFound,
		putBody:   notFoundBody,
	})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: stubConfigNamed(stubName, "1"),
				Check:  resource.TestCheckResourceAttr(stubAddr, "id", stubTokenID),
			},
			{
				Config:      stubConfigNamed("Renamed Agent Token", "1"),
				ExpectError: wrapped("Failed to update the Agent Token"),
			},
			// The rename failed server-side too, so the original name still
			// matches. An empty plan proves the resource is still in state.
			{
				Config:   stubConfigNamed(stubName, "1"),
				PlanOnly: true,
			},
		},
	})
}

// TestAgentTokenRotationWithoutNewTokenIsFatal covers a rotate that reports
// success but returns no secret. Accepting it would leave the dead token in
// state with no pending change, so the agent would be locked out and the next
// plan would look clean.
func TestAgentTokenRotationWithoutNewTokenIsFatal(t *testing.T) {
	api := newStubAPI(stubResponses{
		rotateStatus: http.StatusOK,
		rotateBody: fmt.Sprintf(
			`{"result":{"id":%q,"name":%q,"token":""}}`, stubTokenID, stubName),
	})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: stubConfig("1"),
				Check:  resource.TestCheckResourceAttr(stubAddr, "token", stubTokenValue),
			},
			{
				Config:      stubConfig("2"),
				ExpectError: wrapped("returned no new token value"),
			},
		},
	})
}

// TestAgentTokenCreateWithoutResultIsFatal covers a 200 with no result object,
// which would otherwise be a nil dereference and crash the provider.
func TestAgentTokenCreateWithoutResultIsFatal(t *testing.T) {
	api := newStubAPI(stubResponses{createBody: `{}`})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      stubConfig("1"),
				ExpectError: wrapped("create response contained no result"),
			},
		},
	})
}

// TestAgentTokenReadWithoutResultIsFatal covers the same guard on read.
func TestAgentTokenReadWithoutResultIsFatal(t *testing.T) {
	api := newStubAPI(stubResponses{})

	startStubAPI(t, api)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: stubProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: stubConfig("1"),
				Check:  resource.TestCheckResourceAttr(stubAddr, "id", stubTokenID),
			},
			{
				PreConfig:   func() { api.setReadBody(`{}`) },
				Config:      stubConfig("1"),
				ExpectError: wrapped("read response for the id"),
			},
			// Restore the stub so the framework's teardown refresh and destroy
			// do not trip over the broken read.
			{
				PreConfig: func() { api.setReadBody("") },
				Config:    stubConfig("1"),
				PlanOnly:  true,
			},
		},
	})
}
