package bulksettings

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
)

const (
	MaintainerTypeUser  = "user"
	MaintainerTypeGroup = "group"
)

// ---------------------------------------------------------------------------
// Entities as observed in the API
// ---------------------------------------------------------------------------

// Entity is the common view of a resource or integration that the settings
// resources compare against configuration.
type Entity struct {
	Requestable bool
	WorkflowID  string    // "" when none
	Owner       *Identity // nil when none
	Maintainers []Maintainer
}

// Identity is a user or group as returned by the API.
type Identity struct {
	ID    string
	Email string
}

// Maintainer is a maintainer as returned by the API.
type Maintainer struct {
	Type string // lower-case "user" or "group"
	Identity
}

// IdentityOf converts an API entity into an Identity (nil stays nil).
func IdentityOf(e *client.EntityResponseSchema) *Identity {
	if e == nil {
		return nil
	}
	id := &Identity{ID: e.Id.String()}
	if e.Email != nil {
		id.Email = string(*e.Email)
	}
	return id
}

// maintainerBody is the common JSON shape of the maintainer union types.
type maintainerBody struct {
	Type  string                      `json:"type"`
	User  client.EntityResponseSchema `json:"user"`
	Group client.EntityResponseSchema `json:"group"`
}

// ParseMaintainers converts the API's maintainer union items into Maintainers.
func ParseMaintainers[T interface{ MarshalJSON() ([]byte, error) }](items []T) ([]Maintainer, error) {
	out := make([]Maintainer, 0, len(items))
	for _, item := range items {
		raw, err := item.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var body maintainerBody
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}

		entity := body.User
		if strings.EqualFold(body.Type, MaintainerTypeGroup) {
			entity = body.Group
		}
		out = append(out, Maintainer{Type: strings.ToLower(body.Type), Identity: *IdentityOf(&entity)})
	}
	return out, nil
}

// identityMatches reports whether ref (UUID or email) identifies the identity, ignoring case.
func identityMatches(i Identity, ref string) bool {
	return strings.EqualFold(i.ID, ref) || (i.Email != "" && strings.EqualFold(i.Email, ref))
}

// ---------------------------------------------------------------------------
// Configured settings
// ---------------------------------------------------------------------------

// settingsSpec holds the settings set in configuration; nil fields are unmanaged.
type settingsSpec struct {
	requestable *bool
	workflowID  *string
	owner       *string
	maintainers *[]maintainerRef // replace mode
	additional  []maintainerRef  // add mode
}

type decodedSettings struct {
	ids      []string
	settings settingsSpec
}

func (d decodedSettings) additionalKeys() []string {
	keys := make([]string, 0, len(d.settings.additional))
	for _, m := range d.settings.additional {
		keys = append(keys, m.key())
	}
	return keys
}

// changedSince returns only the settings that differ from prior. Settings
// removed from configuration are not sent (they become unmanaged).
func (s settingsSpec) changedSince(prior settingsSpec) settingsSpec {
	var out settingsSpec
	if s.requestable != nil && (prior.requestable == nil || *prior.requestable != *s.requestable) {
		out.requestable = s.requestable
	}
	if s.workflowID != nil && (prior.workflowID == nil || !strings.EqualFold(*prior.workflowID, *s.workflowID)) {
		out.workflowID = s.workflowID
	}
	if s.owner != nil && (prior.owner == nil || !strings.EqualFold(*prior.owner, *s.owner)) {
		out.owner = s.owner
	}
	if s.maintainers != nil && (prior.maintainers == nil || !sameRefs(*prior.maintainers, *s.maintainers)) {
		out.maintainers = s.maintainers
	}
	return out
}

// update converts the replace-mode settings into a bulk update body.
// Additional maintainers are handled by the additive planner.
func (s settingsSpec) update() client.BulkSettingsUpdate {
	var u client.BulkSettingsUpdate
	u.Requestable = s.requestable
	if s.workflowID != nil {
		u.Workflow = &client.BulkEntityRef{Id: *s.workflowID}
	}
	if s.owner != nil {
		u.Owner = &client.BulkEntityRef{Id: *s.owner}
	}
	if s.maintainers != nil {
		list := toBulkMaintainers(*s.maintainers)
		u.Maintainers = &client.BulkMaintainersUpdate{Replace: &list}
	}
	return u
}

// matches reports whether an entity has every configured replace-mode setting.
func (s settingsSpec) matches(e *Entity) bool {
	if s.requestable != nil && e.Requestable != *s.requestable {
		return false
	}
	if s.workflowID != nil && !strings.EqualFold(e.WorkflowID, *s.workflowID) {
		return false
	}
	if s.owner != nil && (e.Owner == nil || !identityMatches(*e.Owner, *s.owner)) {
		return false
	}
	if s.maintainers != nil && !sameMaintainers(*s.maintainers, e.Maintainers) {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Configured maintainers
// ---------------------------------------------------------------------------

// maintainerRef is a configured maintainer: a type and a UUID or email.
type maintainerRef struct {
	Type string
	ID   string
}

const maintainerKeySep = "\x1f"

// key is the stable identity used for ownership tracking.
func (m maintainerRef) key() string {
	return strings.ToLower(m.Type) + maintainerKeySep + m.ID
}

func parseMaintainerKey(key string) (maintainerRef, bool) {
	t, id, ok := strings.Cut(key, maintainerKeySep)
	return maintainerRef{Type: t, ID: id}, ok
}

func (m maintainerRef) matches(a Maintainer) bool {
	return strings.EqualFold(m.Type, a.Type) && identityMatches(a.Identity, m.ID)
}

// sameMaintainers reports whether actual is exactly the configured list.
func sameMaintainers(want []maintainerRef, actual []Maintainer) bool {
	if len(want) != len(actual) {
		return false
	}
	used := make([]bool, len(actual))
	for _, w := range want {
		found := false
		for i, a := range actual {
			if !used[i] && w.matches(a) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func sameRefs(a, b []maintainerRef) bool {
	keys := func(refs []maintainerRef) []string {
		out := make([]string, len(refs))
		for i, m := range refs {
			out[i] = strings.ToLower(m.key())
		}
		sort.Strings(out)
		return out
	}
	return slices.Equal(keys(a), keys(b))
}

func unionMaintainers(a, b []maintainerRef) []maintainerRef {
	seen := map[string]bool{}
	var out []maintainerRef
	for _, m := range append(slices.Clone(a), b...) {
		if !seen[m.key()] {
			seen[m.key()] = true
			out = append(out, m)
		}
	}
	return out
}

func toBulkMaintainers(refs []maintainerRef) []client.BulkMaintainer {
	sorted := slices.Clone(refs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].key() < sorted[j].key() })

	out := make([]client.BulkMaintainer, 0, len(sorted))
	for _, m := range sorted {
		ref := &client.BulkEntityRef{Id: m.ID}
		bm := client.BulkMaintainer{Type: strings.ToLower(m.Type)}
		if bm.Type == MaintainerTypeGroup {
			bm.Group = ref
		} else {
			bm.User = ref
		}
		out = append(out, bm)
	}
	return out
}

// maintainersChange converts an additive change of maintainer keys into a bulk body.
func maintainersChange(c bulk.Change) *client.BulkMaintainersUpdate {
	if c.IsEmpty() {
		return nil
	}
	toRefs := func(keys []string) []maintainerRef {
		refs := make([]maintainerRef, 0, len(keys))
		for _, k := range keys {
			if m, ok := parseMaintainerKey(k); ok {
				refs = append(refs, m)
			}
		}
		return refs
	}

	u := &client.BulkMaintainersUpdate{}
	if len(c.Add) > 0 {
		add := toBulkMaintainers(toRefs(c.Add))
		u.Add = &add
	}
	if len(c.Remove) > 0 {
		remove := toBulkMaintainers(toRefs(c.Remove))
		u.Remove = &remove
	}
	return u
}

// ---------------------------------------------------------------------------
// Terraform values
// ---------------------------------------------------------------------------

// maintainerModel is one element of maintainers / additional_maintainers.
type maintainerModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

var maintainerAttrTypes = map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}

func decodeMaintainers(ctx context.Context, set types.Set, diags *diag.Diagnostics) []maintainerRef {
	var models []maintainerModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)

	out := make([]maintainerRef, 0, len(models))
	for _, m := range models {
		out = append(out, maintainerRef{Type: m.Type.ValueString(), ID: m.ID.ValueString()})
	}
	return out
}

func maintainersToSet(ctx context.Context, refs []maintainerRef) (types.Set, diag.Diagnostics) {
	models := make([]maintainerModel, 0, len(refs))
	for _, m := range refs {
		models = append(models, maintainerModel{Type: types.StringValue(m.Type), ID: types.StringValue(m.ID)})
	}
	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: maintainerAttrTypes}, models)
}
