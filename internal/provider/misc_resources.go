package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_firewall_schedule
// ---------------------------------------------------------------------------

type firewallScheduleResource struct{ client *client.Client }

var _ resource.Resource = (*firewallScheduleResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallScheduleResource)(nil)
var _ resource.ResourceWithImportState = (*firewallScheduleResource)(nil)

const (
	firewallSchedulePlural   = "/api/v2/firewall/schedules"
	firewallScheduleSingular = "/api/v2/firewall/schedule"
)

// The API models `position`, `month` and `day` as arrays of integers (a time
// range covers one or more weekday positions, or one or more month/day pairs),
// and rejects scalars with FIELD_INVALID_MANY_VALUE.
var timeRangeAttrTypes = map[string]attr.Type{
	"position":   types.ListType{ElemType: types.Int64Type},
	"month":      types.ListType{ElemType: types.Int64Type},
	"day":        types.ListType{ElemType: types.Int64Type},
	"hour":       types.StringType,
	"rangedescr": types.StringType,
}

type firewallScheduleModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Descr     types.String `tfsdk:"descr"`
	Active    types.Bool   `tfsdk:"active"`
	TimeRange types.List   `tfsdk:"timerange"`
}

func NewFirewallScheduleResource() resource.Resource { return &firewallScheduleResource{} }

func (r *firewallScheduleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_schedule"
}
func (r *firewallScheduleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *firewallScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall schedule.",
		Attributes: map[string]schema.Attribute{
			"id":     computedIDAttribute(),
			"name":   requiredNameAttribute(),
			"descr":  optionalStringAttribute("Description for the schedule."),
			"active": schema.BoolAttribute{Computed: true, Description: "Whether the schedule is currently active."},
			"timerange": schema.ListNestedAttribute{
				Required:    true,
				Description: "Time ranges that make up the schedule.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"position":   optionalIntListAttribute("Weekday position(s) this range applies to (1 = Sunday through 7 = Saturday)."),
						"month":      optionalIntListAttribute("Month(s) this range applies to, one entry per matching `day` entry."),
						"day":        optionalIntListAttribute("Day(s) of the month this range applies to, one entry per matching `month` entry."),
						"hour":       optionalStringAttribute("Hour range (e.g. `9:00-17:00`)."),
						"rangedescr": optionalStringAttribute("Description for this time range."),
					},
				},
			},
		},
	}
}

func (r *firewallScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setString(payload, "descr", plan.Descr)
	payload["timerange"] = listObjects(plan.TimeRange)
	raw, err := r.client.Create(ctx, firewallScheduleSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to create schedule", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create schedule", err.Error())
		return
	}
	plan.Active = boolValue(getBool(obj, "active"))
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, firewallSchedulePlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read schedule", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Descr = strValue(getString(obj, "descr"))
	state.Active = boolValue(getBool(obj, "active"))
	state.TimeRange = objectsToListValue(ctx, getSliceMap(obj, "timerange"), types.ObjectType{AttrTypes: timeRangeAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallScheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, firewallSchedulePlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update schedule", err.Error())
		} else {
			resp.Diagnostics.AddError("schedule not found", name+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "descr", plan.Descr)
	payload["timerange"] = listObjects(plan.TimeRange)
	raw, err := r.client.Update(ctx, firewallScheduleSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update schedule", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update schedule", err.Error())
		return
	}
	plan.Active = boolValue(getBool(obj, "active"))
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallScheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, firewallSchedulePlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete schedule", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, firewallScheduleSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete schedule", err.Error())
	}
}

func (r *firewallScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_interface_vlan
// ---------------------------------------------------------------------------

type interfaceVLANResource struct{ client *client.Client }

var _ resource.Resource = (*interfaceVLANResource)(nil)
var _ resource.ResourceWithConfigure = (*interfaceVLANResource)(nil)
var _ resource.ResourceWithImportState = (*interfaceVLANResource)(nil)

const (
	interfaceVLANPlural   = "/api/v2/interface/vlans"
	interfaceVLANSingular = "/api/v2/interface/vlan"
)

type interfaceVLANModel struct {
	ID     types.String `tfsdk:"id"`
	If     types.String `tfsdk:"if"`
	Tag    types.Int64  `tfsdk:"tag"`
	PCP    types.Int64  `tfsdk:"pcp"`
	Descr  types.String `tfsdk:"descr"`
	VLANIf types.String `tfsdk:"vlanif"`
}

func NewInterfaceVLANResource() resource.Resource { return &interfaceVLANResource{} }

func (r *interfaceVLANResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_interface_vlan"
}
func (r *interfaceVLANResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *interfaceVLANResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a VLAN interface. Identified by parent interface and VLAN tag.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			// `if` and `tag` form the natural key and are the only fields the
			// API refuses to change in place, so both replace the resource.
			"if": schema.StringAttribute{
				Required:    true,
				Description: "Parent interface for the VLAN.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag": schema.Int64Attribute{
				Required:    true,
				Description: "VLAN tag (1-4094).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"pcp":    optionalIntAttribute("802.1p priority code point (0-7)."),
			"descr":  optionalStringAttribute("Description for the VLAN."),
			"vlanif": schema.StringAttribute{Computed: true, Description: "The generated VLAN interface name (e.g. `vlan0.10`)."},
		},
	}
}

func (r *interfaceVLANResource) key(iface string, tag int64) string {
	return iface + "|" + formatID(tag)
}

func (r *interfaceVLANResource) find(ctx context.Context, iface string, tag int64) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, interfaceVLANPlural, map[string]string{"if": iface, "tag": formatID(tag)})
}

func (r *interfaceVLANResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceVLANModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "if", plan.If)
	setInt(payload, "tag", plan.Tag)
	setInt(payload, "pcp", plan.PCP)
	setString(payload, "descr", plan.Descr)
	raw, err := r.client.Create(ctx, interfaceVLANSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to create VLAN", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create VLAN", err.Error())
		return
	}
	plan.VLANIf = strValue(getString(obj, "vlanif"))
	plan.ID = types.StringValue(r.key(plan.If.ValueString(), plan.Tag.ValueInt64()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceVLANResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceVLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	tag := state.Tag.ValueInt64()
	if iface == "" {
		var tagStr string
		iface, tagStr = splitRouteKey(state.ID.ValueString())
		tag = atoi64(tagStr)
	}
	_, obj, found, err := r.find(ctx, iface, tag)
	if err != nil {
		resp.Diagnostics.AddError("failed to read VLAN", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, tag))
	state.If = types.StringValue(iface)
	state.Tag = types.Int64Value(tag)
	state.PCP = intValue(getInt(obj, "pcp"))
	state.Descr = strValue(getString(obj, "descr"))
	state.VLANIf = strValue(getString(obj, "vlanif"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interfaceVLANResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceVLANModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.If.ValueString()
	tag := plan.Tag.ValueInt64()
	if iface == "" {
		var tagStr string
		iface, tagStr = splitRouteKey(plan.ID.ValueString())
		tag = atoi64(tagStr)
	}
	id, _, found, err := r.find(ctx, iface, tag)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update VLAN", err.Error())
		} else {
			resp.Diagnostics.AddError("VLAN not found", iface+" tag "+formatID(tag)+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setInt(payload, "pcp", plan.PCP)
	setString(payload, "descr", plan.Descr)
	raw, err := r.client.Update(ctx, interfaceVLANSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update VLAN", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update VLAN", err.Error())
		return
	}
	plan.VLANIf = strValue(getString(obj, "vlanif"))
	plan.ID = types.StringValue(r.key(iface, tag))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceVLANResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interfaceVLANModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	tag := state.Tag.ValueInt64()
	if iface == "" {
		var tagStr string
		iface, tagStr = splitRouteKey(state.ID.ValueString())
		tag = atoi64(tagStr)
	}
	id, _, found, err := r.find(ctx, iface, tag)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete VLAN", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, interfaceVLANSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete VLAN", err.Error())
	}
}

func (r *interfaceVLANResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func atoi64(s string) int64 {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int64(c-'0')
	}
	return n
}
