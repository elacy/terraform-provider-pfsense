package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// firewallAliasResource implements pfsense_firewall_alias.
type firewallAliasResource struct {
	client *client.Client
}

var _ resource.Resource = (*firewallAliasResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallAliasResource)(nil)
var _ resource.ResourceWithImportState = (*firewallAliasResource)(nil)

const (
	firewallAliasPlural   = "/api/v2/firewall/aliases"
	firewallAliasSingular = "/api/v2/firewall/alias"
)

type firewallAliasModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Descr   types.String `tfsdk:"descr"`
	Address types.List   `tfsdk:"address"`
	Detail  types.List   `tfsdk:"detail"`
}

func NewFirewallAliasResource() resource.Resource {
	return &firewallAliasResource{}
}

func (r *firewallAliasResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_alias"
}

func (r *firewallAliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}

func (r *firewallAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a firewall alias. Aliases group hosts, networks or ports under a single name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The alias name (aliases are identified by name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The alias name. Unique among aliases; immutable after creation.",
				Validators: []validator.String{
					stringvalidator.LengthAtMost(31),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The alias type: host, network or port.",
				Validators: []validator.String{
					stringvalidator.OneOf("host", "network", "port"),
				},
			},
			"descr": schema.StringAttribute{
				Optional:    true,
				Description: "Description for the alias.",
			},
			"address": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Host, network or port entries for the alias (depends on `type`).",
			},
			"detail": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Per-entry descriptions matching `address` order.",
			},
		},
	}
}

func (r *firewallAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setString(payload, "type", plan.Type)
	setString(payload, "descr", plan.Descr)
	setStringList(payload, "address", plan.Address)
	setStringList(payload, "detail", plan.Detail)
	applyNow(payload)

	if _, err := r.client.Create(ctx, firewallAliasSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create firewall alias", err.Error())
		return
	}

	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}

	id, obj, found, err := findByKey(ctx, r.client, firewallAliasPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read firewall alias", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Type = strValue(getString(obj, "type"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Address = strListValue(ctx, getStringSlice(obj, "address"))
	state.Detail = strListValue(ctx, getStringSlice(obj, "detail"))
	_ = id
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallAliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}

	id, _, found, err := findByKey(ctx, r.client, firewallAliasPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to update firewall alias", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("firewall alias not found", "alias "+name+" no longer exists")
		return
	}

	payload := map[string]any{"id": id}
	setString(payload, "type", plan.Type)
	setString(payload, "descr", plan.Descr)
	setStringList(payload, "address", plan.Address)
	setStringList(payload, "detail", plan.Detail)
	applyNow(payload)

	if _, err := r.client.Update(ctx, firewallAliasSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update firewall alias", err.Error())
		return
	}

	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}

	id, _, found, err := findByKey(ctx, r.client, firewallAliasPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete firewall alias", err.Error())
		return
	}
	if !found {
		return // already gone
	}

	q := client.Query{}.Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, firewallAliasSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete firewall alias", err.Error())
		return
	}
}

func (r *firewallAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
