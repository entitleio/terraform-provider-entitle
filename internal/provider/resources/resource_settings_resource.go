package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

var _ resource.Resource = &ResourceSettingsResource{}
var _ resource.ResourceWithConfigure = &ResourceSettingsResource{}
var _ resource.ResourceWithConfigValidators = &ResourceSettingsResource{}

// NewResourceSettingsResource creates a new entitle_resource_settings resource.
func NewResourceSettingsResource() resource.Resource {
	return &ResourceSettingsResource{}
}

// ResourceSettingsResource applies the same settings to many Entitle resources
// using the bulk API. Only attributes set in configuration are managed.
type ResourceSettingsResource struct {
	client   *client.ClientWithResponses
	executor *bulk.Executor
}

// ResourceSettingsResourceModel describes the resource data model.
type ResourceSettingsResourceModel struct {
	ResourceIDs           types.Set    `tfsdk:"resource_ids"`
	Requestable           types.Bool   `tfsdk:"requestable"`
	WorkflowID            types.String `tfsdk:"workflow_id"`
	Owner                 types.String `tfsdk:"owner"`
	Maintainers           types.Set    `tfsdk:"maintainers"`
	AdditionalMaintainers types.Set    `tfsdk:"additional_maintainers"`
}

// bulkMaintainerModel is one element of maintainers / additional_maintainers.
type bulkMaintainerModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

var bulkMaintainerAttrTypes = map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}

// Metadata sets the resource type name.
func (r *ResourceSettingsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_settings"
}

func bulkMaintainersAttribute(description string) schema.SetNestedAttribute {
	return schema.SetNestedAttribute{
		Optional:            true,
		MarkdownDescription: description,
		Description:         description,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "`user` or `group`.",
					Description:         "user or group.",
					Validators: []validator.String{
						stringvalidator.OneOf(utils.MaintainerTypeUser, utils.MaintainerTypeGroup),
					},
				},
				"id": schema.StringAttribute{
					Required:            true,
					MarkdownDescription: "The user or group, as a UUID or an email address.",
					Description:         "The user or group, as a UUID or an email address.",
					Validators: []validator.String{
						stringvalidator.LengthAtLeast(1),
					},
				},
			},
		},
	}
}

// Schema defines the resource schema.
func (r *ResourceSettingsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	maintainers := bulkMaintainersAttribute("The exact list of maintainers for every resource in `resource_ids`; " +
		"replaces existing maintainers. An empty list removes all maintainers. Conflicts with `additional_maintainers`.")
	additional := bulkMaintainersAttribute("Maintainers added to every resource in `resource_ids`, keeping existing ones. " +
		"Only maintainers added by this resource are removed when they are taken out of the list or a resource is removed " +
		"from `resource_ids`. Conflicts with `maintainers`.")
	additional.Validators = []validator.Set{setvalidator.SizeAtLeast(1)}

	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ResourceSettingsResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"resource_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "IDs (UUID) of the Entitle resources the settings are applied to.",
				Description:         "IDs (UUID) of the Entitle resources the settings are applied to.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(validators.UUID{}),
				},
			},
			"requestable": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether users can request access to the resources.",
				Description:         "Whether users can request access to the resources.",
			},
			"workflow_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ID (UUID) of the default approval workflow for the resources.",
				Description:         "ID (UUID) of the default approval workflow for the resources.",
				Validators:          []validator.String{validators.UUID{}},
			},
			"owner": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Owner of the resources, as a user UUID or email address.",
				Description:         "Owner of the resources, as a user UUID or email address.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"maintainers":            maintainers,
			"additional_maintainers": additional,
		},
	}
}

// ConfigValidators requires at least one setting and forbids both maintainer modes at once.
func (r *ResourceSettingsResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("maintainers"),
			path.MatchRoot("additional_maintainers"),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot("requestable"),
			path.MatchRoot("workflow_id"),
			path.MatchRoot("owner"),
			path.MatchRoot("maintainers"),
			path.MatchRoot("additional_maintainers"),
		),
	}
}

// Configure wires the API client and the bulk executor.
func (r *ResourceSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	executor, c, err := bulk.NewExecutorFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", err.Error())
		return
	}
	if executor == nil {
		return
	}

	r.client = c
	r.executor = executor
}

// Create applies every configured setting to every resource.
func (r *ResourceSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired := r.decode(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	a := bulk.Additive{NewIDs: desired.ids, NewValues: desired.additionalKeys()}
	planned, err := bulk.PlanAdditive(ctx, a, r.inspector(nil))
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resources Before Applying Settings", err.Error())
		return
	}

	if err := r.apply(ctx, a, planned, settingsSpec{}, desired.settings); err != nil {
		// No state is saved, so the next apply creates the resource again. Roll
		// back the additional maintainers added so far; otherwise that create
		// would see them as pre-existing and never remove them. Settings are not
		// owned, so a partially applied setting is simply applied again.
		if rbErr := r.rollbackAdditional(ctx, desired, planned.Next); rbErr != nil {
			resp.Diagnostics.AddWarning("Failed to Roll Back Additional Maintainers",
				"Some additional maintainers added before the failure could not be removed and will be treated as "+
					"pre-existing on the next apply: "+rbErr.Error())
		}
		resp.Diagnostics.AddError("Failed to Apply Resource Settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Read verifies every managed resource: configured settings must match, and
// additional maintainers must be present. Resources that drifted are dropped
// from resource_ids (so the next plan re-applies the settings) but stay managed
// in private state; resources that no longer exist are forgotten.
func (r *ResourceSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceSettingsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current := r.decode(ctx, state, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inSync, next, err := bulk.Refresh(ctx, current.ids, ownership, current.additionalKeys(), r.inspector(&current.settings))
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resource Settings", err.Error())
		return
	}
	if len(next.Drifted) > 0 || len(inSync) != len(current.ids) {
		tflog.Info(ctx, "Resources drifted from entitle_resource_settings", map[string]any{
			"drifted": next.Drifted,
			"in_sync": len(inSync),
		})
	}

	set, diags := bulk.ToStringSet(ctx, inSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.ResourceIDs = set

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(next.Save(ctx, resp.Private)...)
}

// Update sends new resources every configured setting, and resources that stay
// only the settings that changed. Resources removed from resource_ids keep
// their settings; only additional maintainers added by this resource are removed.
func (r *ResourceSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceSettingsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired := r.decode(ctx, plan, &resp.Diagnostics)
	prior := r.decode(ctx, state, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	a := bulk.Additive{
		OldIDs:    prior.ids,
		NewIDs:    desired.ids,
		OldValues: prior.additionalKeys(),
		NewValues: desired.additionalKeys(),
		Prior:     ownership,
	}
	planned, err := bulk.PlanAdditive(ctx, a, r.inspector(nil))
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resources Before Updating Settings", err.Error())
		return
	}

	if err := r.apply(ctx, a, planned, prior.settings, desired.settings); err != nil {
		// Keep every resource and additional maintainer that might have been
		// touched, plus all ownership learned; the next Read detects what was
		// applied and the next apply finishes the change.
		failed := plan
		ids, d1 := bulk.ToStringSet(ctx, bulk.Unique(append(slices.Clone(prior.ids), desired.ids...)))
		resp.Diagnostics.Append(d1...)
		failed.ResourceIDs = ids
		if !plan.AdditionalMaintainers.IsNull() || !state.AdditionalMaintainers.IsNull() {
			merged, d2 := maintainersToSet(ctx, unionMaintainers(prior.settings.additional, desired.settings.additional))
			resp.Diagnostics.Append(d2...)
			failed.AdditionalMaintainers = merged
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &failed)...)
		resp.Diagnostics.Append(planned.OnError.Save(ctx, resp.Private)...)
		resp.Diagnostics.AddError("Failed to Update Resource Settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Delete only removes the resource from Terraform state. Settings and
// maintainers stay as they are in Entitle, like entitle_role_synced.
func (r *ResourceSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Removing entitle_resource_settings from state; resource settings in Entitle are left unchanged")
}

// apply sends removals of owned additional maintainers first (skipping
// resources that no longer exist), then one bulk operation per distinct update.
func (r *ResourceSettingsResource) apply(ctx context.Context, a bulk.Additive, p bulk.AdditivePlan, prior, desired settingsSpec) error {
	ids := a.IDs()
	updates := map[string]client.BulkResourceUpdate{}

	full := desired.update()
	changed := desired.changedSince(prior).update()
	for _, id := range ids.Added {
		updates[id] = full
	}
	for _, id := range ids.Kept {
		if a.IsDrifted(id) {
			updates[id] = full
		} else {
			updates[id] = changed
		}
	}
	for id, c := range p.ChangesByID {
		u := updates[id]
		if u.Maintainers != nil {
			// maintainers (replace) is being set, which defines the exact list;
			// any add/remove of additional maintainers is superseded.
			continue
		}
		u.Maintainers = maintainersChange(c)
		updates[id] = u
	}

	removals := map[string]client.BulkResourceUpdate{}
	for id, c := range p.RemovalsByID {
		removals[id] = client.BulkResourceUpdate{Maintainers: maintainersChange(c)}
	}

	ops := append(
		r.operations("remove additional maintainers", removals, true),
		r.operations("update resource settings", updates, false)...,
	)
	_, err := r.executor.RunAll(ctx, ops...)
	return err
}

// rollbackAdditional removes the additional maintainers this resource owns
// (everything not recorded as pre-existing), skipping resources that are gone.
func (r *ResourceSettingsResource) rollbackAdditional(ctx context.Context, d decodedSettings, owned bulk.Ownership) error {
	if len(d.settings.additional) == 0 {
		return nil
	}
	removals := map[string]client.BulkResourceUpdate{}
	for _, g := range owned.RemoveAll(d.ids, d.additionalKeys()) {
		for _, id := range g.IDs {
			removals[id] = client.BulkResourceUpdate{Maintainers: maintainersChange(g.Change)}
		}
	}
	_, err := r.executor.RunAll(ctx, r.operations("roll back additional maintainers", removals, true)...)
	return err
}

// operations groups resources with identical updates into bulk operations.
func (r *ResourceSettingsResource) operations(name string, updates map[string]client.BulkResourceUpdate, skipMissing bool) []bulk.Operation {
	groups := bulk.GroupBy(updates, updateKey)
	ops := make([]bulk.Operation, 0, len(groups))
	for _, g := range groups {
		ops = append(ops, bulk.Operation{
			Name:           name,
			IDs:            g.IDs,
			SkipMissingIDs: skipMissing,
			Submit:         bulk.ResourcesUpdate(r.client, g.Value),
		})
	}
	return ops
}

// updateKey identifies an update for grouping; "" means nothing to send.
func updateKey(u client.BulkResourceUpdate) string {
	if u.Requestable == nil && u.Workflow == nil && u.Owner == nil && u.Maintainers == nil && u.Tags == nil {
		return ""
	}
	b, err := json.Marshal(u)
	if err != nil {
		// Cannot happen for these plain structs; never merge on failure.
		return fmt.Sprintf("%p", &u)
	}
	return string(b)
}

func (r *ResourceSettingsResource) parallelism() int {
	return int(r.executor.Config().Parallelism)
}

// inspector returns the InspectFunc for Read (spec != nil: compare settings)
// and for read-before-write planning (spec == nil: additional maintainers only).
func (r *ResourceSettingsResource) inspector(spec *settingsSpec) bulk.InspectFunc {
	return func(ctx context.Context, want map[string][]string) (map[string]bulk.Presence, error) {
		return inspectResources(ctx, r.client, r.parallelism(), want, func(result *client.IntegrationResourceResultSchema, keys []string) (bulk.Presence, error) {
			actual, err := actualMaintainers(result.Maintainers)
			if err != nil {
				return bulk.Presence{}, fmt.Errorf("reading maintainers: %w", err)
			}

			var presence bulk.Presence
			for _, key := range keys {
				if m, ok := parseMaintainerKey(key); ok && slices.ContainsFunc(actual, m.matches) {
					presence.Values = append(presence.Values, key)
				}
			}
			if spec != nil {
				presence.Mismatch = !spec.matches(result, actual)
			}
			return presence, nil
		})
	}
}

// ---------------------------------------------------------------------------
// Settings model helpers
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

func (r *ResourceSettingsResource) decode(ctx context.Context, m ResourceSettingsResourceModel, diags *diag.Diagnostics) decodedSettings {
	var out decodedSettings

	ids, d := bulk.StringSet(ctx, m.ResourceIDs)
	diags.Append(d...)
	out.ids = ids

	if !m.Requestable.IsNull() && !m.Requestable.IsUnknown() {
		v := m.Requestable.ValueBool()
		out.settings.requestable = &v
	}
	if !m.WorkflowID.IsNull() && !m.WorkflowID.IsUnknown() {
		v := m.WorkflowID.ValueString()
		out.settings.workflowID = &v
	}
	if !m.Owner.IsNull() && !m.Owner.IsUnknown() {
		v := m.Owner.ValueString()
		out.settings.owner = &v
	}
	if !m.Maintainers.IsNull() && !m.Maintainers.IsUnknown() {
		list := decodeMaintainers(ctx, m.Maintainers, diags)
		out.settings.maintainers = &list
	}
	if !m.AdditionalMaintainers.IsNull() && !m.AdditionalMaintainers.IsUnknown() {
		out.settings.additional = decodeMaintainers(ctx, m.AdditionalMaintainers, diags)
	}

	return out
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
func (s settingsSpec) update() client.BulkResourceUpdate {
	var u client.BulkResourceUpdate
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

// matches reports whether a resource has every configured replace-mode setting.
func (s settingsSpec) matches(res *client.IntegrationResourceResultSchema, actual []actualMaintainer) bool {
	if s.requestable != nil && res.Requestable != *s.requestable {
		return false
	}
	if s.workflowID != nil && (res.Workflow == nil || !strings.EqualFold(res.Workflow.Id.String(), *s.workflowID)) {
		return false
	}
	if s.owner != nil && (res.Owner == nil || !entityMatches(*res.Owner, *s.owner)) {
		return false
	}
	if s.maintainers != nil && !sameMaintainers(*s.maintainers, actual) {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// Maintainers
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

// actualMaintainer is a maintainer as returned by the API.
type actualMaintainer struct {
	Type  string
	ID    string
	Email string
}

func (m maintainerRef) matches(a actualMaintainer) bool {
	if !strings.EqualFold(m.Type, a.Type) {
		return false
	}
	return strings.EqualFold(m.ID, a.ID) || (a.Email != "" && strings.EqualFold(m.ID, a.Email))
}

func entityMatches(e client.EntityResponseSchema, ref string) bool {
	if strings.EqualFold(e.Id.String(), ref) {
		return true
	}
	return e.Email != nil && strings.EqualFold(string(*e.Email), ref)
}

func actualMaintainers(items []client.IntegrationResourceResultSchema_Maintainers_Item) ([]actualMaintainer, error) {
	out := make([]actualMaintainer, 0, len(items))
	for _, item := range items {
		raw, err := item.MarshalJSON()
		if err != nil {
			return nil, err
		}
		var body utils.MaintainerCommonResponseSchema
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, err
		}

		entity := body.User
		if strings.EqualFold(body.Type, utils.MaintainerTypeGroup) {
			entity = body.Group
		}
		a := actualMaintainer{Type: strings.ToLower(body.Type), ID: entity.Id.String()}
		if entity.Email != nil {
			a.Email = string(*entity.Email)
		}
		out = append(out, a)
	}
	return out, nil
}

// sameMaintainers reports whether actual is exactly the configured list.
func sameMaintainers(want []maintainerRef, actual []actualMaintainer) bool {
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
		if bm.Type == utils.MaintainerTypeGroup {
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

func decodeMaintainers(ctx context.Context, set types.Set, diags *diag.Diagnostics) []maintainerRef {
	var models []bulkMaintainerModel
	diags.Append(set.ElementsAs(ctx, &models, false)...)

	out := make([]maintainerRef, 0, len(models))
	for _, m := range models {
		out = append(out, maintainerRef{Type: m.Type.ValueString(), ID: m.ID.ValueString()})
	}
	return out
}

func maintainersToSet(ctx context.Context, refs []maintainerRef) (types.Set, diag.Diagnostics) {
	models := make([]bulkMaintainerModel, 0, len(refs))
	for _, m := range refs {
		models = append(models, bulkMaintainerModel{Type: types.StringValue(m.Type), ID: types.StringValue(m.ID)})
	}
	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: bulkMaintainerAttrTypes}, models)
}
