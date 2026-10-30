// Package bulksettings implements the "*_settings" bulk resources
// (entitle_resource_settings, entitle_integration_settings): the same settings
// applied to many entities with the bulk API.
//
//   - Only attributes set in configuration are sent. New entities receive every
//     configured setting; entities that stay receive only the settings that
//     changed. An attribute removed from configuration becomes unmanaged.
//   - maintainers replaces the whole list; additional_maintainers adds and only
//     removes the maintainers this resource added (ownership in private state).
//   - Refresh reads the entities in batches and re-applies drifted ones.
//   - Removing an ID or destroying leaves settings in place (state-only removal);
//     only additional maintainers added by this resource are removed from an ID
//     taken out of the list.
//
// Each concrete resource only supplies a Kind: its names, how to read its
// entities and how to submit an update.
package bulksettings

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Kind describes one settings resource.
type Kind struct {
	// TypeSuffix is appended to the provider type name, e.g. "_resource_settings".
	TypeSuffix string
	// IDsAttribute is the name of the ID set attribute, e.g. "resource_ids".
	IDsAttribute string
	// Entity and Entities name the managed entities in descriptions and
	// messages, e.g. "resource" / "resources".
	Entity, Entities string
	// MarkdownDescription is the resource documentation.
	MarkdownDescription string
	// Fetch reads the entities with the given IDs. The result is keyed by
	// lower-cased ID; entities that do not exist are absent.
	Fetch func(ctx context.Context, c *client.ClientWithResponses, parallelism int, ids []string) (map[string]*Entity, error)
	// Submit returns the bulk submit function for one update.
	Submit func(c *client.ClientWithResponses, u client.BulkSettingsUpdate) bulk.SubmitFunc
}

// New creates a settings resource for the given kind.
func New(k Kind) resource.Resource {
	return &Resource{kind: k}
}

var _ resource.Resource = &Resource{}
var _ resource.ResourceWithConfigure = &Resource{}
var _ resource.ResourceWithConfigValidators = &Resource{}

// Resource is the shared implementation of the settings resources.
type Resource struct {
	kind     Kind
	client   *client.ClientWithResponses
	executor *bulk.Executor
}

const (
	attrRequestable = "requestable"
	attrWorkflowID  = "workflow_id"
	attrOwner       = "owner"
	attrMaintainers = "maintainers"
	attrAdditional  = "additional_maintainers"
)

// Metadata sets the resource type name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.kind.TypeSuffix
}

func maintainersAttribute(description string) schema.SetNestedAttribute {
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
						stringvalidator.OneOf(MaintainerTypeUser, MaintainerTypeGroup),
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
func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	k := r.kind
	maintainers := maintainersAttribute(fmt.Sprintf("The exact list of maintainers for every %s in `%s`; "+
		"replaces existing maintainers. An empty list removes all maintainers. Conflicts with `additional_maintainers`.", k.Entity, k.IDsAttribute))
	additional := maintainersAttribute(fmt.Sprintf("Maintainers added to every %s in `%s`, keeping existing ones. "+
		"Only maintainers added by this resource are removed when they are taken out of the list or a %s is removed "+
		"from `%s`. Conflicts with `maintainers`.", k.Entity, k.IDsAttribute, k.Entity, k.IDsAttribute))
	additional.Validators = []validator.Set{setvalidator.SizeAtLeast(1)}

	idsDescription := fmt.Sprintf("IDs (UUID) of the Entitle %s the settings are applied to.", k.Entities)

	resp.Schema = schema.Schema{
		MarkdownDescription: k.MarkdownDescription,
		Attributes: map[string]schema.Attribute{
			k.IDsAttribute: schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: idsDescription,
				Description:         idsDescription,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(validators.UUID{}),
				},
			},
			attrRequestable: schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("Whether users can request access to the %s.", k.Entities),
				Description:         fmt.Sprintf("Whether users can request access to the %s.", k.Entities),
			},
			attrWorkflowID: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("ID (UUID) of the default approval workflow for the %s.", k.Entities),
				Description:         fmt.Sprintf("ID (UUID) of the default approval workflow for the %s.", k.Entities),
				Validators:          []validator.String{validators.UUID{}},
			},
			attrOwner: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("Owner of the %s, as a user UUID or email address.", k.Entities),
				Description:         fmt.Sprintf("Owner of the %s, as a user UUID or email address.", k.Entities),
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			attrMaintainers: maintainers,
			attrAdditional:  additional,
		},
	}
}

// ConfigValidators requires at least one setting and forbids both maintainer modes at once.
func (r *Resource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot(attrMaintainers),
			path.MatchRoot(attrAdditional),
		),
		resourcevalidator.AtLeastOneOf(
			path.MatchRoot(attrRequestable),
			path.MatchRoot(attrWorkflowID),
			path.MatchRoot(attrOwner),
			path.MatchRoot(attrMaintainers),
			path.MatchRoot(attrAdditional),
		),
	}
}

// Configure wires the API client and the bulk executor.
func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create applies every configured setting to every entity.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	desired := r.decode(ctx, req.Plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	a := bulk.Additive{NewIDs: desired.ids, NewValues: desired.additionalKeys()}
	planned, err := bulk.PlanAdditive(ctx, a, r.inspector(nil))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to Read %s Before Applying Settings", title(r.kind.Entities)), err.Error())
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
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to Apply %s Settings", title(r.kind.Entity)), err.Error())
		return
	}

	resp.State.Raw = req.Plan.Raw.Copy()
	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Read verifies every managed entity: configured settings must match, and
// additional maintainers must be present. Entities that drifted are dropped
// from the ID set (so the next plan re-applies the settings) but stay managed
// in private state; entities that no longer exist are forgotten.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	current := r.decode(ctx, req.State, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inSync, next, err := bulk.Refresh(ctx, current.ids, ownership, current.additionalKeys(), r.inspector(&current.settings))
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to Read %s Settings", title(r.kind.Entity)), err.Error())
		return
	}
	if len(next.Drifted) > 0 || len(inSync) != len(current.ids) {
		tflog.Info(ctx, "Entities drifted from settings", map[string]any{
			"resource_type": "entitle" + r.kind.TypeSuffix,
			"drifted":       next.Drifted,
			"in_sync":       len(inSync),
		})
	}

	set, diags := bulk.ToStringSet(ctx, inSync)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.kind.IDsAttribute), set)...)
	resp.Diagnostics.Append(next.Save(ctx, resp.Private)...)
}

// Update sends new entities every configured setting, and entities that stay
// only the settings that changed. Entities removed from the ID set keep their
// settings; only additional maintainers added by this resource are removed.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	desired := r.decode(ctx, req.Plan, &resp.Diagnostics)
	prior := r.decode(ctx, req.State, &resp.Diagnostics)
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
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to Read %s Before Updating Settings", title(r.kind.Entities)), err.Error())
		return
	}

	resp.State.Raw = req.Plan.Raw.Copy()

	if err := r.apply(ctx, a, planned, prior.settings, desired.settings); err != nil {
		// Keep every entity and additional maintainer that might have been
		// touched, plus all ownership learned; the next Read detects what was
		// applied and the next apply finishes the change.
		ids, d := bulk.ToStringSet(ctx, bulk.Unique(append(slices.Clone(prior.ids), desired.ids...)))
		resp.Diagnostics.Append(d...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.kind.IDsAttribute), ids)...)
		if len(prior.settings.additional) > 0 || len(desired.settings.additional) > 0 {
			merged, d := maintainersToSet(ctx, unionMaintainers(prior.settings.additional, desired.settings.additional))
			resp.Diagnostics.Append(d...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attrAdditional), merged)...)
		}

		resp.Diagnostics.Append(planned.OnError.Save(ctx, resp.Private)...)
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to Update %s Settings", title(r.kind.Entity)), err.Error())
		return
	}

	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Delete only removes the resource from Terraform state. Settings and
// maintainers stay as they are in Entitle, like entitle_role_synced.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, fmt.Sprintf("Removing entitle%s from state; %s settings in Entitle are left unchanged", r.kind.TypeSuffix, r.kind.Entity))
}

// apply sends removals of owned additional maintainers first (skipping
// entities that no longer exist), then one bulk operation per distinct update.
func (r *Resource) apply(ctx context.Context, a bulk.Additive, p bulk.AdditivePlan, prior, desired settingsSpec) error {
	ids := a.IDs()
	updates := map[string]client.BulkSettingsUpdate{}

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

	removals := map[string]client.BulkSettingsUpdate{}
	for id, c := range p.RemovalsByID {
		removals[id] = client.BulkSettingsUpdate{Maintainers: maintainersChange(c)}
	}

	ops := append(
		r.operations("remove additional maintainers", removals, true),
		r.operations(fmt.Sprintf("update %s settings", r.kind.Entity), updates, false)...,
	)
	_, err := r.executor.RunAll(ctx, ops...)
	return err
}

// rollbackAdditional removes the additional maintainers this resource owns
// (everything not recorded as pre-existing), skipping entities that are gone.
func (r *Resource) rollbackAdditional(ctx context.Context, d decodedSettings, owned bulk.Ownership) error {
	if len(d.settings.additional) == 0 {
		return nil
	}
	removals := map[string]client.BulkSettingsUpdate{}
	for _, g := range owned.RemoveAll(d.ids, d.additionalKeys()) {
		for _, id := range g.IDs {
			removals[id] = client.BulkSettingsUpdate{Maintainers: maintainersChange(g.Change)}
		}
	}
	_, err := r.executor.RunAll(ctx, r.operations("roll back additional maintainers", removals, true)...)
	return err
}

// operations groups entities with identical updates into bulk operations.
func (r *Resource) operations(name string, updates map[string]client.BulkSettingsUpdate, skipMissing bool) []bulk.Operation {
	groups := bulk.GroupBy(updates, updateKey)
	ops := make([]bulk.Operation, 0, len(groups))
	for _, g := range groups {
		ops = append(ops, bulk.Operation{
			Name:           name,
			IDs:            g.IDs,
			SkipMissingIDs: skipMissing,
			Submit:         r.kind.Submit(r.client, g.Value),
		})
	}
	return ops
}

// updateKey identifies an update for grouping; "" means nothing to send.
func updateKey(u client.BulkSettingsUpdate) string {
	if u.IsEmpty() {
		return ""
	}
	b, err := json.Marshal(u)
	if err != nil {
		// Cannot happen for these plain structs; never merge on failure.
		return fmt.Sprintf("%p", &u)
	}
	return string(b)
}

func (r *Resource) parallelism() int {
	return int(r.executor.Config().Parallelism)
}

// inspector returns the InspectFunc for Read (spec != nil: compare settings)
// and for read-before-write planning (spec == nil: additional maintainers only).
func (r *Resource) inspector(spec *settingsSpec) bulk.InspectFunc {
	return func(ctx context.Context, want map[string][]string) (map[string]bulk.Presence, error) {
		found, err := r.kind.Fetch(ctx, r.client, r.parallelism(), bulk.SortedKeys(want))
		if err != nil {
			return nil, err
		}

		out := make(map[string]bulk.Presence, len(want))
		for id, keys := range want {
			entity, ok := found[strings.ToLower(id)]
			if !ok || entity == nil {
				continue // does not exist
			}

			presence := bulk.Presence{Exists: true}
			for _, key := range keys {
				if m, ok := parseMaintainerKey(key); ok && slices.ContainsFunc(entity.Maintainers, m.matches) {
					presence.Values = append(presence.Values, key)
				}
			}
			if spec != nil {
				presence.Mismatch = !spec.matches(entity)
			}
			out[id] = presence
		}
		return out, nil
	}
}

// decode reads the configured settings from a plan or state.
func (r *Resource) decode(ctx context.Context, g attributeGetter, diags *diag.Diagnostics) decodedSettings {
	var (
		out                    decodedSettings
		ids, maint, additional types.Set
		requestable            types.Bool
		workflowID, owner      types.String
	)

	diags.Append(g.GetAttribute(ctx, path.Root(r.kind.IDsAttribute), &ids)...)
	diags.Append(g.GetAttribute(ctx, path.Root(attrRequestable), &requestable)...)
	diags.Append(g.GetAttribute(ctx, path.Root(attrWorkflowID), &workflowID)...)
	diags.Append(g.GetAttribute(ctx, path.Root(attrOwner), &owner)...)
	diags.Append(g.GetAttribute(ctx, path.Root(attrMaintainers), &maint)...)
	diags.Append(g.GetAttribute(ctx, path.Root(attrAdditional), &additional)...)
	if diags.HasError() {
		return out
	}

	list, d := bulk.StringSet(ctx, ids)
	diags.Append(d...)
	out.ids = list

	if !requestable.IsNull() && !requestable.IsUnknown() {
		v := requestable.ValueBool()
		out.settings.requestable = &v
	}
	if !workflowID.IsNull() && !workflowID.IsUnknown() {
		v := workflowID.ValueString()
		out.settings.workflowID = &v
	}
	if !owner.IsNull() && !owner.IsUnknown() {
		v := owner.ValueString()
		out.settings.owner = &v
	}
	if !maint.IsNull() && !maint.IsUnknown() {
		refs := decodeMaintainers(ctx, maint, diags)
		out.settings.maintainers = &refs
	}
	if !additional.IsNull() && !additional.IsUnknown() {
		out.settings.additional = decodeMaintainers(ctx, additional, diags)
	}
	return out
}

// attributeGetter is satisfied by tfsdk.Plan and tfsdk.State.
type attributeGetter interface {
	GetAttribute(ctx context.Context, p path.Path, target interface{}) diag.Diagnostics
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
