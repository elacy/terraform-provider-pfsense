package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_routing_gateway
// ---------------------------------------------------------------------------

type routingGatewayResource struct{ client *client.Client }

var _ resource.Resource = (*routingGatewayResource)(nil)
var _ resource.ResourceWithConfigure = (*routingGatewayResource)(nil)
var _ resource.ResourceWithImportState = (*routingGatewayResource)(nil)

const (
	routingGatewayPlural   = "/api/v2/routing/gateways"
	routingGatewaySingular = "/api/v2/routing/gateway"
)

type routingGatewayModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	Descr                     types.String `tfsdk:"descr"`
	Disabled                  types.Bool   `tfsdk:"disabled"`
	IPProtocol                types.String `tfsdk:"ipprotocol"`
	Interface                 types.String `tfsdk:"interface"`
	Gateway                   types.String `tfsdk:"gateway"`
	MonitorDisable            types.Bool   `tfsdk:"monitor_disable"`
	Monitor                   types.String `tfsdk:"monitor"`
	ActionDisable             types.Bool   `tfsdk:"action_disable"`
	ForceDown                 types.Bool   `tfsdk:"force_down"`
	DpingerDontAddStaticRoute types.Bool   `tfsdk:"dpinger_dont_add_static_route"`
	GwDownKillStates          types.String `tfsdk:"gw_down_kill_states"`
	NonlocalGateway           types.Bool   `tfsdk:"nonlocalgateway"`
	Weight                    types.Int64  `tfsdk:"weight"`
	DataPayload               types.Int64  `tfsdk:"data_payload"`
	LatencyLow                types.Int64  `tfsdk:"latencylow"`
	LatencyHigh               types.Int64  `tfsdk:"latencyhigh"`
	LossLow                   types.Int64  `tfsdk:"losslow"`
	LossHigh                  types.Int64  `tfsdk:"losshigh"`
	Interval                  types.Int64  `tfsdk:"interval"`
	LossInterval              types.Int64  `tfsdk:"loss_interval"`
	TimePeriod                types.Int64  `tfsdk:"time_period"`
	AlertInterval             types.Int64  `tfsdk:"alert_interval"`
}

func NewRoutingGatewayResource() resource.Resource { return &routingGatewayResource{} }

func (r *routingGatewayResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_routing_gateway"
}
func (r *routingGatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *routingGatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a routing gateway.",
		Attributes: map[string]schema.Attribute{
			"id":                            computedIDAttribute(),
			"name":                          requiredNameAttribute(),
			"descr":                         optionalStringAttribute("Description for this gateway."),
			"disabled":                      optionalBoolAttribute("Disable this gateway."),
			"ipprotocol":                    enumAttribute("Address family: inet or inet6.", "inet", "inet6"),
			"interface":                     optionalStringAttribute("Interface this gateway is reachable through."),
			"gateway":                       optionalStringAttribute("The gateway IP address.", isIPAddressOrDynamic()),
			"monitor_disable":               optionalBoolAttribute("Disable gateway monitoring."),
			"monitor":                       optionalStringAttribute("Alternate IP address to monitor.", isIPAddress()),
			"action_disable":                optionalBoolAttribute("Do not take action when the gateway is down."),
			"force_down":                    optionalBoolAttribute("Force the gateway as down."),
			"dpinger_dont_add_static_route": optionalBoolAttribute("Do not add a static route for this gateway."),
			"gw_down_kill_states":           optionalStringAttribute("State kill behaviour when the gateway is down."),
			"nonlocalgateway":               optionalBoolAttribute("Allow a non-local gateway address."),
			"weight":                        optionalIntAttribute("Weight for gateway groups (1-30)."),
			"data_payload":                  optionalIntAttribute("Data payload for monitoring probes."),
			"latencylow":                    optionalIntAttribute("Low latency threshold (ms)."),
			"latencyhigh":                   optionalIntAttribute("High latency threshold (ms)."),
			"losslow":                       optionalIntAttribute("Low packet-loss threshold."),
			"losshigh":                      optionalIntAttribute("High packet-loss threshold."),
			"interval":                      optionalIntAttribute("Probe interval (ms)."),
			"loss_interval":                 optionalIntAttribute("Loss probe interval (ms)."),
			"time_period":                   optionalIntAttribute("Time period for monitoring (ms)."),
			"alert_interval":                optionalIntAttribute("Alert interval (ms)."),
		},
	}
}

func (r *routingGatewayResource) payload(m routingGatewayModel) map[string]any {
	p := map[string]any{}
	setString(p, "name", m.Name)
	setString(p, "descr", m.Descr)
	setBool(p, "disabled", m.Disabled)
	setString(p, "ipprotocol", m.IPProtocol)
	setString(p, "interface", m.Interface)
	setString(p, "gateway", m.Gateway)
	setBool(p, "monitor_disable", m.MonitorDisable)
	setString(p, "monitor", m.Monitor)
	setBool(p, "action_disable", m.ActionDisable)
	setBool(p, "force_down", m.ForceDown)
	setBool(p, "dpinger_dont_add_static_route", m.DpingerDontAddStaticRoute)
	setString(p, "gw_down_kill_states", m.GwDownKillStates)
	setBool(p, "nonlocalgateway", m.NonlocalGateway)
	setInt(p, "weight", m.Weight)
	setInt(p, "data_payload", m.DataPayload)
	setInt(p, "latencylow", m.LatencyLow)
	setInt(p, "latencyhigh", m.LatencyHigh)
	setInt(p, "losslow", m.LossLow)
	setInt(p, "losshigh", m.LossHigh)
	setInt(p, "interval", m.Interval)
	setInt(p, "loss_interval", m.LossInterval)
	setInt(p, "time_period", m.TimePeriod)
	setInt(p, "alert_interval", m.AlertInterval)
	return p
}

func (r *routingGatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routingGatewayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, routingGatewaySingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create gateway", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingGatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routingGatewayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, routingGatewayPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read gateway", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Descr = strValue(getString(obj, "descr"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.IPProtocol = strValue(getString(obj, "ipprotocol"))
	state.Interface = strValue(getString(obj, "interface"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.MonitorDisable = boolValue(getBool(obj, "monitor_disable"))
	state.Monitor = strValue(getString(obj, "monitor"))
	state.ActionDisable = boolValue(getBool(obj, "action_disable"))
	state.ForceDown = boolValue(getBool(obj, "force_down"))
	state.DpingerDontAddStaticRoute = boolValue(getBool(obj, "dpinger_dont_add_static_route"))
	state.GwDownKillStates = strValue(getString(obj, "gw_down_kill_states"))
	state.NonlocalGateway = boolValue(getBool(obj, "nonlocalgateway"))
	state.Weight = intValue(getInt(obj, "weight"))
	state.DataPayload = intValue(getInt(obj, "data_payload"))
	state.LatencyLow = intValue(getInt(obj, "latencylow"))
	state.LatencyHigh = intValue(getInt(obj, "latencyhigh"))
	state.LossLow = intValue(getInt(obj, "losslow"))
	state.LossHigh = intValue(getInt(obj, "losshigh"))
	state.Interval = intValue(getInt(obj, "interval"))
	state.LossInterval = intValue(getInt(obj, "loss_interval"))
	state.TimePeriod = intValue(getInt(obj, "time_period"))
	state.AlertInterval = intValue(getInt(obj, "alert_interval"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *routingGatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routingGatewayModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, routingGatewayPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update gateway", err.Error())
		} else {
			resp.Diagnostics.AddError("gateway not found", "gateway "+name+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, routingGatewaySingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update gateway", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingGatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routingGatewayModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, routingGatewayPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete gateway", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, routingGatewaySingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete gateway", err.Error())
	}
}

func (r *routingGatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_routing_gateway_group
// ---------------------------------------------------------------------------

type routingGatewayGroupResource struct{ client *client.Client }

var _ resource.Resource = (*routingGatewayGroupResource)(nil)
var _ resource.ResourceWithConfigure = (*routingGatewayGroupResource)(nil)
var _ resource.ResourceWithImportState = (*routingGatewayGroupResource)(nil)

const (
	routingGatewayGroupPlural   = "/api/v2/routing/gateway/groups"
	routingGatewayGroupSingular = "/api/v2/routing/gateway/group"
)

type gatewayGroupPriorityModel struct {
	Gateway   types.String `tfsdk:"gateway"`
	Tier      types.Int64  `tfsdk:"tier"`
	VirtualIP types.String `tfsdk:"virtual_ip"`
}

type routingGatewayGroupModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Trigger    types.String `tfsdk:"trigger"`
	Descr      types.String `tfsdk:"descr"`
	Priorities types.List   `tfsdk:"priorities"`
}

var gatewayGroupPriorityAttrTypes = map[string]attr.Type{
	"gateway":    types.StringType,
	"tier":       types.Int64Type,
	"virtual_ip": types.StringType,
}

func NewRoutingGatewayGroupResource() resource.Resource { return &routingGatewayGroupResource{} }

func (r *routingGatewayGroupResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_routing_gateway_group"
}
func (r *routingGatewayGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *routingGatewayGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a gateway group (a tiered set of gateways for failover/load balancing).",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"name":    requiredNameAttribute(),
			"trigger": enumAttribute("Group trigger level: down, downloss, downlatency or downlosslatency.", "down", "downloss", "downlatency", "downlosslatency"),
			"descr":   optionalStringAttribute("Description for this gateway group."),
			"priorities": schema.ListNestedAttribute{
				Required:    true,
				Description: "Ordered gateway priorities (tiers).",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"gateway":    optionalStringAttribute("Gateway name."),
						"tier":       optionalIntAttribute("Tier for this gateway (1-255)."),
						"virtual_ip": optionalStringAttribute("Virtual IP to use as the group address."),
					},
				},
			},
		},
	}
}

func (r *routingGatewayGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routingGatewayGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setString(payload, "trigger", plan.Trigger)
	setString(payload, "descr", plan.Descr)
	payload["priorities"] = listObjects(plan.Priorities)
	if _, err := r.client.Create(ctx, routingGatewayGroupSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create gateway group", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingGatewayGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routingGatewayGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, routingGatewayGroupPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read gateway group", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Trigger = strValue(getString(obj, "trigger"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Priorities = objectsToListValue(ctx, getSliceMap(obj, "priorities"), types.ObjectType{AttrTypes: gatewayGroupPriorityAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *routingGatewayGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routingGatewayGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, routingGatewayGroupPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update gateway group", err.Error())
		} else {
			resp.Diagnostics.AddError("gateway group not found", "group "+name+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "trigger", plan.Trigger)
	setString(payload, "descr", plan.Descr)
	payload["priorities"] = listObjects(plan.Priorities)
	if _, err := r.client.Update(ctx, routingGatewayGroupSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update gateway group", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingGatewayGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routingGatewayGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, routingGatewayGroupPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete gateway group", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, routingGatewayGroupSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete gateway group", err.Error())
	}
}

func (r *routingGatewayGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_routing_static_route
// ---------------------------------------------------------------------------

type routingStaticRouteResource struct{ client *client.Client }

var _ resource.Resource = (*routingStaticRouteResource)(nil)
var _ resource.ResourceWithConfigure = (*routingStaticRouteResource)(nil)
var _ resource.ResourceWithImportState = (*routingStaticRouteResource)(nil)

const (
	routingStaticRoutePlural   = "/api/v2/routing/static_routes"
	routingStaticRouteSingular = "/api/v2/routing/static_route"
)

type routingStaticRouteModel struct {
	ID       types.String `tfsdk:"id"`
	Network  types.String `tfsdk:"network"`
	Gateway  types.String `tfsdk:"gateway"`
	Descr    types.String `tfsdk:"descr"`
	Disabled types.Bool   `tfsdk:"disabled"`
}

func NewRoutingStaticRouteResource() resource.Resource { return &routingStaticRouteResource{} }

func (r *routingStaticRouteResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_routing_static_route"
}
func (r *routingStaticRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *routingStaticRouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a static route. Identified by the combination of `network` and `gateway`.",
		Attributes: map[string]schema.Attribute{
			"id":       computedIDAttribute(),
			"network":  requiredStringAttribute("Destination network CIDR."),
			"gateway":  requiredStringAttribute("Gateway name for this route."),
			"descr":    optionalStringAttribute("Description for this route."),
			"disabled": optionalBoolAttribute("Create the route disabled."),
		},
	}
}

func (r *routingStaticRouteResource) routeKey(network, gateway string) string {
	return network + "|" + gateway
}

func (r *routingStaticRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routingStaticRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "network", plan.Network)
	setString(payload, "gateway", plan.Gateway)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "disabled", plan.Disabled)
	if _, err := r.client.Create(ctx, routingStaticRouteSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create static route", err.Error())
		return
	}
	plan.ID = types.StringValue(r.routeKey(plan.Network.ValueString(), plan.Gateway.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingStaticRouteResource) find(ctx context.Context, network, gateway string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, routingStaticRoutePlural, map[string]string{"network": network, "gateway": gateway})
}

func (r *routingStaticRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routingStaticRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	network := state.Network.ValueString()
	gateway := state.Gateway.ValueString()
	if network == "" {
		network, gateway = splitRouteKey(state.ID.ValueString())
	}
	_, obj, found, err := r.find(ctx, network, gateway)
	if err != nil {
		resp.Diagnostics.AddError("failed to read static route", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.routeKey(network, gateway))
	state.Network = strValue(getString(obj, "network"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *routingStaticRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routingStaticRouteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	network := plan.Network.ValueString()
	gateway := plan.Gateway.ValueString()
	if network == "" {
		network, gateway = splitRouteKey(plan.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, network, gateway)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update static route", err.Error())
		} else {
			resp.Diagnostics.AddError("static route not found", network+" via "+gateway+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "network", plan.Network)
	setString(payload, "gateway", plan.Gateway)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "disabled", plan.Disabled)
	if _, err := r.client.Update(ctx, routingStaticRouteSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update static route", err.Error())
		return
	}
	plan.ID = types.StringValue(r.routeKey(network, gateway))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routingStaticRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routingStaticRouteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	network := state.Network.ValueString()
	gateway := state.Gateway.ValueString()
	if network == "" {
		network, gateway = splitRouteKey(state.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, network, gateway)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete static route", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, routingStaticRouteSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete static route", err.Error())
	}
}

func (r *routingStaticRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func splitRouteKey(id string) (network, gateway string) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '|' {
			return id[:i], id[i+1:]
		}
	}
	return id, ""
}
