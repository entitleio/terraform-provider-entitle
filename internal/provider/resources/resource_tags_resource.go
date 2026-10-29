package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

var _ resource.Resource = &ResourceTagsResource{}
var _ resource.ResourceWithConfigure = &ResourceTagsResource{}

// NewResourceTagsResource creates a new entitle_resource_tags resource.
func NewResourceTagsResource() resource.Resource {
	return &ResourceTagsResource{}
}

// ResourceTagsResource adds a set of tags to many Entitle resources using the bulk API.
type ResourceTagsResource struct {
	client   *client.ClientWithResponses
	executor *bulk.Executor
}

// ResourceTagsResourceModel describes the resource data model.
type ResourceTagsResourceModel struct {
	ResourceIDs types.Set `tfsdk:"resource_ids"`
	Tags        types.Set `tfsdk:"tags"`
}

// Metadata sets the resource type name.
func (r *ResourceTagsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_tags"
}

// Schema defines the resource schema.
func (r *ResourceTagsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ResourceTagsResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"resource_ids": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "IDs (UUID) of the Entitle resources the tags are added to.",
				Description:         "IDs (UUID) of the Entitle resources the tags are added to.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(validators.UUID{}),
				},
			},
			"tags": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Tags added to every resource in `resource_ids`.",
				Description:         "Tags added to every resource in resource_ids.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

// Configure wires the API client and the bulk executor.
func (r *ResourceTagsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create adds the tags to every resource. Tags a resource already has are
// recorded as pre-existing and will never be removed by this resource.
func (r *ResourceTagsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceTagsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, tags := r.values(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	planned, err := r.plan(ctx, bulk.Additive{NewIDs: ids, NewValues: tags})
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resources Before Adding Tags", err.Error())
		return
	}

	if err := r.apply(ctx, planned); err != nil {
		// Persist state so Terraform marks the resource as tainted; the next
		// apply removes the partially added (owned) tags and re-creates it.
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.Append(planned.OnError.Save(ctx, resp.Private)...)
		resp.Diagnostics.AddError("Failed to Add Resource Tags", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Read verifies that every managed resource still exists and carries all tags.
// Resources that no longer exist are forgotten. Resources missing a tag are
// dropped from resource_ids (so the next plan re-applies the tags) but stay
// managed in private state.
func (r *ResourceTagsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceTagsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, tags := r.values(ctx, state, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	managed := ownership.Managed(ids)
	observed, err := bulk.Inspect(ctx, r.parallelism(), managed, func(ctx context.Context, id string) (bulk.Presence, error) {
		return r.inspect(ctx, id, tags)
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resource Tags", err.Error())
		return
	}

	inSync, next := ownership.Reconcile(managed, observed, tags)
	if len(next.Drifted) > 0 || len(inSync) != len(ids) {
		tflog.Info(ctx, "Resources drifted from entitle_resource_tags", map[string]any{
			"drifted": next.Drifted,
			"missing": bulk.Difference(managed, append(slices.Clone(inSync), next.Drifted...)),
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

// Update applies only the differences between prior state and plan, never
// removing pre-existing tags.
func (r *ResourceTagsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceTagsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newIDs, newTags := r.values(ctx, plan, &resp.Diagnostics)
	oldIDs, oldTags := r.values(ctx, state, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned, err := r.plan(ctx, bulk.Additive{
		OldIDs:    oldIDs,
		NewIDs:    newIDs,
		OldValues: oldTags,
		NewValues: newTags,
		Prior:     ownership,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to Read Resources Before Updating Tags", err.Error())
		return
	}

	if err := r.apply(ctx, planned); err != nil {
		// Keep every resource and tag that might have been touched, plus all
		// ownership learned, so Read detects what was partially applied and
		// the next apply finishes it without losing track of pre-existing tags.
		failed := state
		ids, d1 := bulk.ToStringSet(ctx, bulk.Unique(append(slices.Clone(oldIDs), newIDs...)))
		tags, d2 := bulk.ToStringSet(ctx, bulk.Unique(append(slices.Clone(oldTags), newTags...)))
		resp.Diagnostics.Append(d1...)
		resp.Diagnostics.Append(d2...)
		failed.ResourceIDs, failed.Tags = ids, tags

		resp.Diagnostics.Append(resp.State.Set(ctx, &failed)...)
		resp.Diagnostics.Append(planned.OnError.Save(ctx, resp.Private)...)
		resp.Diagnostics.AddError("Failed to Update Resource Tags", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(planned.Next.Save(ctx, resp.Private)...)
}

// Delete removes the tags this resource added. Pre-existing tags stay, and
// resources that no longer exist are skipped.
func (r *ResourceTagsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceTagsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ids, tags := r.values(ctx, state, &resp.Diagnostics)
	ownership, diags := bulk.LoadOwnership(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ops := r.operations("remove resource tags", ownership.RemoveAll(ids, tags), true)
	if _, err := r.executor.RunAll(ctx, ops...); err != nil {
		resp.Diagnostics.AddError("Failed to Remove Resource Tags", err.Error())
	}
}

// plan reads the resources that need it and computes the bulk operations.
func (r *ResourceTagsResource) plan(ctx context.Context, a bulk.Additive) (bulk.AdditivePlan, error) {
	toInspect := a.ToInspect()
	ids := make([]string, 0, len(toInspect))
	for id := range toInspect {
		ids = append(ids, id)
	}

	observed, err := bulk.Inspect(ctx, r.parallelism(), ids, func(ctx context.Context, id string) (bulk.Presence, error) {
		return r.inspect(ctx, id, toInspect[id])
	})
	if err != nil {
		return bulk.AdditivePlan{}, err
	}

	present := make(map[string][]string, len(observed))
	for id, p := range observed {
		present[id] = p.Values
	}

	return a.Plan(present), nil
}

// apply runs removals first (skipping resources that no longer exist), then the remaining changes.
func (r *ResourceTagsResource) apply(ctx context.Context, p bulk.AdditivePlan) error {
	ops := append(
		r.operations("remove resource tags", p.Removals, true),
		r.operations("update resource tags", p.Changes, false)...,
	)
	_, err := r.executor.RunAll(ctx, ops...)
	return err
}

func (r *ResourceTagsResource) parallelism() int {
	return int(r.executor.Config().Parallelism)
}

// values extracts resource IDs and tags from a model.
func (r *ResourceTagsResource) values(ctx context.Context, m ResourceTagsResourceModel, diags *diag.Diagnostics) ([]string, []string) {
	ids, d := bulk.StringSet(ctx, m.ResourceIDs)
	diags.Append(d...)
	tags, d := bulk.StringSet(ctx, m.Tags)
	diags.Append(d...)
	return ids, tags
}

// operations converts change groups into bulk operations on resource tags.
func (r *ResourceTagsResource) operations(name string, groups []bulk.ChangeGroup, skipMissing bool) []bulk.Operation {
	ops := make([]bulk.Operation, 0, len(groups))
	for _, g := range groups {
		tagsUpdate := &client.BulkTagsUpdate{}
		if len(g.Add) > 0 {
			add := g.Add
			tagsUpdate.Add = &add
		}
		if len(g.Remove) > 0 {
			remove := g.Remove
			tagsUpdate.Remove = &remove
		}

		ops = append(ops, bulk.Operation{
			Name:           name,
			IDs:            g.IDs,
			SkipMissingIDs: skipMissing,
			Submit:         bulk.ResourcesUpdate(r.client, client.BulkResourceUpdate{Tags: tagsUpdate}),
		})
	}
	return ops
}

// inspect reads a resource and reports which of the given tags it has
// (case-insensitive, returned in the given spelling).
func (r *ResourceTagsResource) inspect(ctx context.Context, id string, tags []string) (bulk.Presence, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		// Unreachable for configured IDs (validated as UUIDs by the schema);
		// fail loudly rather than treating a corrupt ID as a deleted resource.
		return bulk.Presence{}, fmt.Errorf("invalid resource id %q: %w", id, err)
	}

	resp, err := r.client.ResourcesShowWithResponse(ctx, uid)
	if err != nil {
		return bulk.Presence{}, err
	}
	if resp.HTTPResponse.StatusCode == http.StatusNotFound {
		return bulk.Presence{}, nil
	}
	if err := utils.HTTPResponseToError(resp.HTTPResponse.StatusCode, resp.Body); err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			return bulk.Presence{}, nil
		}
		return bulk.Presence{}, err
	}
	if resp.JSON200 == nil {
		return bulk.Presence{}, errors.New("empty response when reading resource " + id)
	}

	result := resp.JSON200.Result
	current := map[string]struct{}{}
	for _, t := range append(utils.StringSliceValue(result.UserDefinedTags), utils.StringSliceValue(result.Tags)...) {
		current[normalizeTag(t)] = struct{}{}
	}

	presence := bulk.Presence{Exists: true}
	for _, t := range tags {
		if _, ok := current[normalizeTag(t)]; ok {
			presence.Values = append(presence.Values, t)
		}
	}
	return presence, nil
}

func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
