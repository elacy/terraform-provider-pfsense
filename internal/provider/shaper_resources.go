package provider

import (
	"context"
	"strconv"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Shaper-specific conversion helpers. The shared helpers.go covers string,
// bool and int64 fields; the limiter models additionally expose a float64
// `plr` (packet loss ratio) field that has no shared helper.
func getFloat(obj map[string]any, key string) *float64 {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case float64:
		return &t
	case string:
		f, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return nil
		}
		return &f
	default:
		return nil
	}
}

func floatValue(f *float64) types.Float64 {
	if f == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*f)
}

func setFloat(m map[string]any, key string, v types.Float64) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = v.ValueFloat64()
}

// ---------------------------------------------------------------------------
// pfsense_firewall_virtual_ip
// ---------------------------------------------------------------------------

type virtualIPResource struct{ client *client.Client }

var _ resource.Resource = (*virtualIPResource)(nil)
var _ resource.ResourceWithConfigure = (*virtualIPResource)(nil)
var _ resource.ResourceWithImportState = (*virtualIPResource)(nil)

const (
	virtualIPsPlural  = "/api/v2/firewall/virtual_ips"
	virtualIPSingular = "/api/v2/firewall/virtual_ip"
)

type virtualIPModel struct {
	ID         types.String `tfsdk:"id"`
	Uniqid     types.String `tfsdk:"uniqid"`
	Mode       types.String `tfsdk:"mode"`
	Interface  types.String `tfsdk:"interface"`
	Type       types.String `tfsdk:"type"`
	Subnet     types.String `tfsdk:"subnet"`
	SubnetBits types.Int64  `tfsdk:"subnet_bits"`
	Descr      types.String `tfsdk:"descr"`
	Noexpand   types.Bool   `tfsdk:"noexpand"`
	Vhid       types.Int64  `tfsdk:"vhid"`
	Advbase    types.Int64  `tfsdk:"advbase"`
	Advskew    types.Int64  `tfsdk:"advskew"`
	Password   types.String `tfsdk:"password"`
	CARPStatus types.String `tfsdk:"carp_status"`
	CARPMode   types.String `tfsdk:"carp_mode"`
	CARPPeer   types.String `tfsdk:"carp_peer"`
}

func NewVirtualIPResource() resource.Resource { return &virtualIPResource{} }

func (r *virtualIPResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_virtual_ip"
}
func (r *virtualIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *virtualIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall virtual IP. Identified by its `descr` description.",
		Attributes: map[string]schema.Attribute{
			"id":     computedIDAttribute(),
			"uniqid": schema.StringAttribute{Computed: true, Description: "The unique ID for this virtual IP, assigned by the system."},
			"mode": schema.StringAttribute{
				Required:    true,
				Description: "The virtual IP mode: ipalias, proxyarp, carp or other.",
				Validators:  []validator.String{stringvalidator.OneOf("ipalias", "proxyarp", "carp", "other")},
			},
			"interface":   requiredStringAttribute("The interface this virtual IP applies to."),
			"type":        enumAttribute("The virtual IP scope type: single or network.", "single", "network"),
			"subnet":      requiredStringAttribute("The address for this virtual IP."),
			"subnet_bits": schema.Int64Attribute{Required: true, Description: "The subnet bits for this virtual IP."},
			"descr": schema.StringAttribute{
				Required:    true,
				Description: "A description for administrative reference. Unique; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"noexpand": optionalBoolAttribute("Disable expansion of this entry into IPs on NAT lists."),
			"vhid":     schema.Int64Attribute{Required: true, Description: "The VHID group shared by all CARP members (CARP mode only)."},
			"advbase":  optionalIntAttribute("The base frequency that this machine advertises (CARP mode only)."),
			"advskew":  optionalIntAttribute("The frequency skew that this machine advertises (CARP mode only)."),
			"password": sensitiveStringAttribute("The VHID group password shared by all CARP members (CARP mode only). Write-only: the API never returns it, so the configured value is retained in state."),
			"carp_status": schema.StringAttribute{
				Computed:    true,
				Description: "The current CARP status of this virtual IP (CARP mode only).",
			},
			"carp_mode": enumAttribute("The CARP mode: mcast or ucast (CARP mode only).", "mcast", "ucast"),
			"carp_peer": requiredStringAttribute("The IP address of the CARP peer (ucast CARP mode only)."),
		},
	}
}

func (r *virtualIPResource) payload(m virtualIPModel) map[string]any {
	p := map[string]any{}
	setString(p, "mode", m.Mode)
	setString(p, "interface", m.Interface)
	setString(p, "type", m.Type)
	setString(p, "subnet", m.Subnet)
	setInt(p, "subnet_bits", m.SubnetBits)
	setString(p, "descr", m.Descr)
	setBool(p, "noexpand", m.Noexpand)
	setInt(p, "vhid", m.Vhid)
	setInt(p, "advbase", m.Advbase)
	setInt(p, "advskew", m.Advskew)
	setString(p, "password", m.Password)
	setString(p, "carp_mode", m.CARPMode)
	setString(p, "carp_peer", m.CARPPeer)
	return p
}

func (r *virtualIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan virtualIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, virtualIPSingular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create virtual IP", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create virtual IP", err.Error())
		return
	}
	plan.Uniqid = strValue(getString(obj, "uniqid"))
	plan.CARPStatus = strValue(getString(obj, "carp_status"))
	plan.ID = plan.Descr
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state virtualIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, virtualIPsPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read virtual IP", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.Uniqid = strValue(getString(obj, "uniqid"))
	state.Mode = strValue(getString(obj, "mode"))
	state.Interface = strValue(getString(obj, "interface"))
	state.Type = strValue(getString(obj, "type"))
	state.Subnet = strValue(getString(obj, "subnet"))
	state.SubnetBits = intValue(getInt(obj, "subnet_bits"))
	state.Descr = types.StringValue(descr)
	state.Noexpand = boolValue(getBool(obj, "noexpand"))
	state.Vhid = intValue(getInt(obj, "vhid"))
	state.Advbase = intValue(getInt(obj, "advbase"))
	state.Advskew = intValue(getInt(obj, "advskew"))
	// The CARP password is write-only: overwriting state with the (absent)
	// API value would drop it and leave a permanent diff.
	if password := getString(obj, "password"); password != nil {
		state.Password = types.StringValue(*password)
	}
	state.CARPStatus = strValue(getString(obj, "carp_status"))
	state.CARPMode = strValue(getString(obj, "carp_mode"))
	state.CARPPeer = strValue(getString(obj, "carp_peer"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *virtualIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan virtualIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, virtualIPsPlural, "descr", descr)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update virtual IP", err.Error())
		} else {
			resp.Diagnostics.AddError("virtual IP not found", "virtual IP "+descr+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	raw, err := r.client.Update(ctx, virtualIPSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update virtual IP", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update virtual IP", err.Error())
		return
	}
	plan.Uniqid = strValue(getString(obj, "uniqid"))
	plan.CARPStatus = strValue(getString(obj, "carp_status"))
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *virtualIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state virtualIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, virtualIPsPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete virtual IP", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, virtualIPSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete virtual IP", err.Error())
	}
}

func (r *virtualIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper
// ---------------------------------------------------------------------------

type trafficShaperResource struct{ client *client.Client }

var _ resource.Resource = (*trafficShaperResource)(nil)
var _ resource.ResourceWithConfigure = (*trafficShaperResource)(nil)
var _ resource.ResourceWithImportState = (*trafficShaperResource)(nil)

const (
	trafficShapersPlural  = "/api/v2/firewall/traffic_shapers"
	trafficShaperSingular = "/api/v2/firewall/traffic_shaper"
)

// shaperQueueAttrTypes is the shape of the queue entries embedded in a traffic
// shaper's `queue` list.
var shaperQueueAttrTypes = map[string]attr.Type{
	"name":          types.StringType,
	"enabled":       types.BoolType,
	"priority":      types.Int64Type,
	"qlimit":        types.Int64Type,
	"description":   types.StringType,
	"default":       types.BoolType,
	"red":           types.BoolType,
	"rio":           types.BoolType,
	"ecn":           types.BoolType,
	"codel":         types.BoolType,
	"bandwidthtype": types.StringType,
	"bandwidth":     types.Int64Type,
	"buckets":       types.Int64Type,
	"hogs":          types.Int64Type,
	"borrow":        types.BoolType,
	"upperlimit":    types.BoolType,
	"upperlimit_m1": types.StringType,
	"upperlimit_d":  types.Int64Type,
	"upperlimit_m2": types.StringType,
	"realtime":      types.BoolType,
	"realtime_m1":   types.StringType,
	"realtime_d":    types.Int64Type,
	"realtime_m2":   types.StringType,
	"linkshare":     types.BoolType,
	"linkshare_m1":  types.StringType,
	"linkshare_d":   types.Int64Type,
	"linkshare_m2":  types.StringType,
}

type trafficShaperModel struct {
	ID            types.String `tfsdk:"id"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Interface     types.String `tfsdk:"interface"`
	Name          types.String `tfsdk:"name"`
	Scheduler     types.String `tfsdk:"scheduler"`
	Bandwidthtype types.String `tfsdk:"bandwidthtype"`
	Bandwidth     types.Int64  `tfsdk:"bandwidth"`
	Qlimit        types.Int64  `tfsdk:"qlimit"`
	Tbrconfig     types.Int64  `tfsdk:"tbrconfig"`
	Queue         types.List   `tfsdk:"queue"`
}

func NewTrafficShaperResource() resource.Resource { return &trafficShaperResource{} }

func (r *trafficShaperResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_traffic_shaper"
}
func (r *trafficShaperResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *trafficShaperResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a traffic shaper. Identified by the `interface` it is applied to.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"enabled": optionalBoolAttribute("Enable this traffic shaper."),
			"interface": schema.StringAttribute{
				Required:    true,
				Description: "The interface this traffic shaper applies to. Unique; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The system-generated name of this traffic shaper; cannot be changed.",
			},
			"scheduler": schema.StringAttribute{
				Required:    true,
				Description: "The scheduler type: HFSC, CBQ, FAIRQ, CODELQ or PRIQ.",
				Validators:  []validator.String{stringvalidator.OneOf("HFSC", "CBQ", "FAIRQ", "CODELQ", "PRIQ")},
			},
			"bandwidthtype": schema.StringAttribute{
				Required:    true,
				Description: "The scale type of the `bandwidth` value: %, b, Kb, Mb or Gb.",
				Validators:  []validator.String{stringvalidator.OneOf("%", "b", "Kb", "Mb", "Gb")},
			},
			"bandwidth": schema.Int64Attribute{Required: true, Description: "The total bandwidth amount allowed by this traffic shaper."},
			"qlimit":    optionalIntAttribute("The number of packets that can be held in a queue (default 50)."),
			"tbrconfig": optionalIntAttribute("The size, in bytes, of the token bucket regulator."),
			"queue": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The child queues assigned to this traffic shaper.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":          optionalStringAttribute("Queue name."),
						"enabled":       optionalBoolAttribute("Enable this queue."),
						"priority":      optionalIntAttribute("Priority level for this queue."),
						"qlimit":        optionalIntAttribute("Number of packets that can be held in this queue."),
						"description":   optionalStringAttribute("Queue description."),
						"default":       optionalBoolAttribute("Mark this queue as the default queue."),
						"red":           optionalBoolAttribute("Use the Random Early Detection scheduler option."),
						"rio":           optionalBoolAttribute("Use the Random Early Detection In and Out scheduler option."),
						"ecn":           optionalBoolAttribute("Use the Explicit Congestion Notification scheduler option."),
						"codel":         optionalBoolAttribute("Use the CoDel Active Queue scheduler option."),
						"bandwidthtype": enumAttribute("The scale type of the `bandwidth` value.", "%", "b", "Kb", "Mb", "Gb"),
						"bandwidth":     optionalIntAttribute("The total bandwidth amount allowed by this queue."),
						"buckets":       optionalIntAttribute("The queue's bucket size (FAIRQ only)."),
						"hogs":          optionalIntAttribute("The bandwidth limit per host (FAIRQ only)."),
						"borrow":        optionalBoolAttribute("Allow this queue to borrow from other queues (CBQ only)."),
						"upperlimit":    optionalBoolAttribute("Allow setting the maximum bandwidth for the queue (HFSC only)."),
						"upperlimit_m1": optionalStringAttribute("The burst-able bandwidth limit."),
						"upperlimit_d":  optionalIntAttribute("The duration (ms) the burst-able limit is in effect."),
						"upperlimit_m2": optionalStringAttribute("The normal bandwidth limit."),
						"realtime":      optionalBoolAttribute("Allow setting the guaranteed minimum bandwidth (HFSC only)."),
						"realtime_m1":   optionalStringAttribute("The guaranteed minimum bandwidth during real time."),
						"realtime_d":    optionalIntAttribute("The duration (ms) the guaranteed minimum is in effect."),
						"realtime_m2":   optionalStringAttribute("The maximum bandwidth this queue may use."),
						"linkshare":     optionalBoolAttribute("Allow sharing bandwidth from this queue (HFSC only)."),
						"linkshare_m1":  optionalStringAttribute("The initial bandwidth limit when link sharing."),
						"linkshare_d":   optionalIntAttribute("The duration (ms) the initial limit is in effect."),
						"linkshare_m2":  optionalStringAttribute("The maximum bandwidth this queue may use when link sharing."),
					},
				},
			},
		},
	}
}

func (r *trafficShaperResource) payload(m trafficShaperModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enabled", m.Enabled)
	setString(p, "interface", m.Interface)
	setString(p, "scheduler", m.Scheduler)
	setString(p, "bandwidthtype", m.Bandwidthtype)
	setInt(p, "bandwidth", m.Bandwidth)
	setInt(p, "qlimit", m.Qlimit)
	setInt(p, "tbrconfig", m.Tbrconfig)
	p["queue"] = listObjects(m.Queue)
	return p
}

func (r *trafficShaperResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trafficShaperModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, trafficShaperSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper", err.Error())
		return
	}
	// The create response omits the system-generated `name`, so it is read
	// back from the shaper itself.
	iface := plan.Interface.ValueString()
	_, obj, found, err := findByKey(ctx, r.client, trafficShapersPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("failed to create traffic shaper", "no traffic shaper on "+iface+" after creation")
		return
	}
	plan.Name = strValue(getString(obj, "name"))
	plan.ID = plan.Interface
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trafficShaperModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, trafficShapersPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to read traffic shaper", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(iface)
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Interface = types.StringValue(iface)
	state.Name = strValue(getString(obj, "name"))
	state.Scheduler = strValue(getString(obj, "scheduler"))
	state.Bandwidthtype = strValue(getString(obj, "bandwidthtype"))
	state.Bandwidth = intValue(getInt(obj, "bandwidth"))
	state.Qlimit = intValue(getInt(obj, "qlimit"))
	state.Tbrconfig = intValue(getInt(obj, "tbrconfig"))
	state.Queue = objectsToListValue(ctx, getSliceMap(obj, "queue"), types.ObjectType{AttrTypes: shaperQueueAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficShaperResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trafficShaperModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.Interface.ValueString()
	if iface == "" {
		iface = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, trafficShapersPlural, "interface", iface)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update traffic shaper", err.Error())
		} else {
			resp.Diagnostics.AddError("traffic shaper not found", "no traffic shaper on "+iface)
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	raw, err := r.client.Update(ctx, trafficShaperSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper", err.Error())
		return
	}
	plan.Name = strValue(getString(obj, "name"))
	plan.ID = types.StringValue(iface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trafficShaperModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, trafficShapersPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, trafficShaperSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper", err.Error())
	}
}

func (r *trafficShaperResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_limiter
// ---------------------------------------------------------------------------

type trafficShaperLimiterResource struct{ client *client.Client }

var _ resource.Resource = (*trafficShaperLimiterResource)(nil)
var _ resource.ResourceWithConfigure = (*trafficShaperLimiterResource)(nil)
var _ resource.ResourceWithImportState = (*trafficShaperLimiterResource)(nil)

const (
	trafficShaperLimitersPlural  = "/api/v2/firewall/traffic_shaper/limiters"
	trafficShaperLimiterSingular = "/api/v2/firewall/traffic_shaper/limiter"
)

// limiterBandwidthAttrTypes is the shape of the bandwidth profile entries
// embedded in a limiter's `bandwidth` list.
var limiterBandwidthAttrTypes = map[string]attr.Type{
	"bw":      types.Int64Type,
	"bwscale": types.StringType,
	"bwsched": types.StringType,
}

// limiterQueueAttrTypes is the shape of the queue entries embedded in a
// limiter's `queue` list.
var limiterQueueAttrTypes = map[string]attr.Type{
	"name":                 types.StringType,
	"number":               types.Int64Type,
	"enabled":              types.BoolType,
	"mask":                 types.StringType,
	"maskbits":             types.Int64Type,
	"maskbitsv6":           types.Int64Type,
	"qlimit":               types.Int64Type,
	"ecn":                  types.BoolType,
	"description":          types.StringType,
	"aqm":                  types.StringType,
	"param_codel_target":   types.Int64Type,
	"param_codel_interval": types.Int64Type,
	"param_pie_target":     types.Int64Type,
	"param_pie_tupdate":    types.Int64Type,
	"param_pie_alpha":      types.Int64Type,
	"param_pie_beta":       types.Int64Type,
	"param_pie_max_burst":  types.Int64Type,
	"param_pie_max_ecnth":  types.Int64Type,
	"pie_onoff":            types.BoolType,
	"pie_capdrop":          types.BoolType,
	"pie_qdelay":           types.BoolType,
	"pie_pderand":          types.BoolType,
	"param_red_w_q":        types.Int64Type,
	"param_red_min_th":     types.Int64Type,
	"param_red_max_th":     types.Int64Type,
	"param_red_max_p":      types.Int64Type,
	"param_gred_w_q":       types.Int64Type,
	"param_gred_min_th":    types.Int64Type,
	"param_gred_max_th":    types.Int64Type,
	"param_gred_max_p":     types.Int64Type,
	"weight":               types.Int64Type,
	"plr":                  types.Float64Type,
	"buckets":              types.Int64Type,
}

type trafficShaperLimiterModel struct {
	ID                   types.String  `tfsdk:"id"`
	Name                 types.String  `tfsdk:"name"`
	Number               types.Int64   `tfsdk:"number"`
	Enabled              types.Bool    `tfsdk:"enabled"`
	Mask                 types.String  `tfsdk:"mask"`
	Maskbits             types.Int64   `tfsdk:"maskbits"`
	Maskbitsv6           types.Int64   `tfsdk:"maskbitsv6"`
	Qlimit               types.Int64   `tfsdk:"qlimit"`
	Ecn                  types.Bool    `tfsdk:"ecn"`
	Description          types.String  `tfsdk:"description"`
	Aqm                  types.String  `tfsdk:"aqm"`
	Sched                types.String  `tfsdk:"sched"`
	ParamCodelTarget     types.Int64   `tfsdk:"param_codel_target"`
	ParamCodelInterval   types.Int64   `tfsdk:"param_codel_interval"`
	ParamPieTarget       types.Int64   `tfsdk:"param_pie_target"`
	ParamPieTupdate      types.Int64   `tfsdk:"param_pie_tupdate"`
	ParamPieAlpha        types.Int64   `tfsdk:"param_pie_alpha"`
	ParamPieBeta         types.Int64   `tfsdk:"param_pie_beta"`
	ParamPieMaxBurst     types.Int64   `tfsdk:"param_pie_max_burst"`
	ParamPieMaxEcnth     types.Int64   `tfsdk:"param_pie_max_ecnth"`
	PieOnoff             types.Bool    `tfsdk:"pie_onoff"`
	PieCapdrop           types.Bool    `tfsdk:"pie_capdrop"`
	PieQdelay            types.Bool    `tfsdk:"pie_qdelay"`
	PiePderand           types.Bool    `tfsdk:"pie_pderand"`
	ParamRedWQ           types.Int64   `tfsdk:"param_red_w_q"`
	ParamRedMinTh        types.Int64   `tfsdk:"param_red_min_th"`
	ParamRedMaxTh        types.Int64   `tfsdk:"param_red_max_th"`
	ParamRedMaxP         types.Int64   `tfsdk:"param_red_max_p"`
	ParamGredWQ          types.Int64   `tfsdk:"param_gred_w_q"`
	ParamGredMinTh       types.Int64   `tfsdk:"param_gred_min_th"`
	ParamGredMaxTh       types.Int64   `tfsdk:"param_gred_max_th"`
	ParamGredMaxP        types.Int64   `tfsdk:"param_gred_max_p"`
	ParamFqCodelTarget   types.Int64   `tfsdk:"param_fq_codel_target"`
	ParamFqCodelInterval types.Int64   `tfsdk:"param_fq_codel_interval"`
	ParamFqCodelQuantum  types.Int64   `tfsdk:"param_fq_codel_quantum"`
	ParamFqCodelLimit    types.Int64   `tfsdk:"param_fq_codel_limit"`
	ParamFqCodelFlows    types.Int64   `tfsdk:"param_fq_codel_flows"`
	ParamFqPieTarget     types.Int64   `tfsdk:"param_fq_pie_target"`
	ParamFqPieTupdate    types.Int64   `tfsdk:"param_fq_pie_tupdate"`
	ParamFqPieAlpha      types.Int64   `tfsdk:"param_fq_pie_alpha"`
	ParamFqPieBeta       types.Int64   `tfsdk:"param_fq_pie_beta"`
	ParamFqPieMaxBurst   types.Int64   `tfsdk:"param_fq_pie_max_burst"`
	ParamFqPieMaxEcnth   types.Int64   `tfsdk:"param_fq_pie_max_ecnth"`
	ParamFqPieQuantum    types.Int64   `tfsdk:"param_fq_pie_quantum"`
	ParamFqPieLimit      types.Int64   `tfsdk:"param_fq_pie_limit"`
	ParamFqPieFlows      types.Int64   `tfsdk:"param_fq_pie_flows"`
	Delay                types.Int64   `tfsdk:"delay"`
	Plr                  types.Float64 `tfsdk:"plr"`
	Buckets              types.Int64   `tfsdk:"buckets"`
	Bandwidth            types.List    `tfsdk:"bandwidth"`
	Queue                types.List    `tfsdk:"queue"`
}

func NewTrafficShaperLimiterResource() resource.Resource { return &trafficShaperLimiterResource{} }

func (r *trafficShaperLimiterResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_traffic_shaper_limiter"
}
func (r *trafficShaperLimiterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *trafficShaperLimiterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a traffic shaper limiter. Identified by its `name`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The unique name for this limiter. Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"number":      schema.Int64Attribute{Computed: true, Description: "A unique number auto-assigned to this limiter."},
			"enabled":     optionalBoolAttribute("Enable this limiter and its child queues."),
			"mask":        enumAttribute("Dynamic pipe scope: none, srcaddress or dstaddress.", "none", "srcaddress", "dstaddress"),
			"maskbits":    optionalIntAttribute("IPv4 mask bits for the dynamic pipe scope (default 32)."),
			"maskbitsv6":  optionalIntAttribute("IPv6 mask bits for the dynamic pipe scope (default 128)."),
			"qlimit":      optionalIntAttribute("The length of the limiter's queue."),
			"ecn":         optionalBoolAttribute("Enable or disable ECN."),
			"description": optionalStringAttribute("A verbose description for this limiter."),
			"aqm": schema.StringAttribute{
				Required:    true,
				Description: "The Active Queue Management algorithm: droptail, codel, pie, red or gred.",
				Validators:  []validator.String{stringvalidator.OneOf("droptail", "codel", "pie", "red", "gred")},
			},
			"sched": schema.StringAttribute{
				Required:    true,
				Description: "The scheduler: wf2q+, fifo, qfq, rr, prio, fq_codel or fq_pie.",
				Validators:  []validator.String{stringvalidator.OneOf("wf2q+", "fifo", "qfq", "rr", "prio", "fq_codel", "fq_pie")},
			},
			"param_codel_target":      optionalIntAttribute("CoDel target parameter."),
			"param_codel_interval":    optionalIntAttribute("CoDel interval parameter."),
			"param_pie_target":        optionalIntAttribute("PIE target parameter."),
			"param_pie_tupdate":       optionalIntAttribute("PIE tupdate parameter."),
			"param_pie_alpha":         optionalIntAttribute("PIE alpha parameter."),
			"param_pie_beta":          optionalIntAttribute("PIE beta parameter."),
			"param_pie_max_burst":     optionalIntAttribute("PIE max_burst parameter."),
			"param_pie_max_ecnth":     optionalIntAttribute("PIE ecnth parameter."),
			"pie_onoff":               optionalBoolAttribute("Turn PIE on and off depending on queue load."),
			"pie_capdrop":             optionalBoolAttribute("Enable cap drop adjustment."),
			"pie_qdelay":              optionalBoolAttribute("Use timestamps (true) or departure rate estimation (false) for queue delay."),
			"pie_pderand":             optionalBoolAttribute("Enable drop probability de-randomisation."),
			"param_red_w_q":           optionalIntAttribute("RED w_q parameter."),
			"param_red_min_th":        optionalIntAttribute("RED min_th parameter."),
			"param_red_max_th":        optionalIntAttribute("RED max_th parameter."),
			"param_red_max_p":         optionalIntAttribute("RED max_p parameter."),
			"param_gred_w_q":          optionalIntAttribute("GRED w_q parameter."),
			"param_gred_min_th":       optionalIntAttribute("GRED min_th parameter."),
			"param_gred_max_th":       optionalIntAttribute("GRED max_th parameter."),
			"param_gred_max_p":        optionalIntAttribute("GRED max_p parameter."),
			"param_fq_codel_target":   optionalIntAttribute("FQ CoDel target parameter."),
			"param_fq_codel_interval": optionalIntAttribute("FQ CoDel interval parameter."),
			"param_fq_codel_quantum":  optionalIntAttribute("FQ CoDel quantum parameter."),
			"param_fq_codel_limit":    optionalIntAttribute("FQ CoDel limit parameter."),
			"param_fq_codel_flows":    optionalIntAttribute("FQ CoDel flows parameter."),
			"param_fq_pie_target":     optionalIntAttribute("FQ PIE target parameter."),
			"param_fq_pie_tupdate":    optionalIntAttribute("FQ PIE tupdate parameter."),
			"param_fq_pie_alpha":      optionalIntAttribute("FQ PIE alpha parameter."),
			"param_fq_pie_beta":       optionalIntAttribute("FQ PIE beta parameter."),
			"param_fq_pie_max_burst":  optionalIntAttribute("FQ PIE max_burst parameter."),
			"param_fq_pie_max_ecnth":  optionalIntAttribute("FQ PIE ecnth parameter."),
			"param_fq_pie_quantum":    optionalIntAttribute("FQ PIE quantum parameter."),
			"param_fq_pie_limit":      optionalIntAttribute("FQ PIE limit parameter."),
			"param_fq_pie_flows":      optionalIntAttribute("FQ PIE flows parameter."),
			"delay":                   optionalIntAttribute("The amount of delay (ms) added to traffic."),
			"plr":                     schema.Float64Attribute{Optional: true, Description: "The amount of packet loss (percentage) added to traffic."},
			"buckets":                 optionalIntAttribute("The limiter's bucket size (slots)."),
			"bandwidth": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The bandwidth profiles for this limiter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bw":      optionalIntAttribute("The amount of bandwidth this profile allows."),
						"bwscale": enumAttribute("The scale factor of the `bw` value.", "b", "Kb", "Mb"),
						"bwsched": optionalStringAttribute("The schedule this bandwidth profile is assigned to."),
					},
				},
			},
			"queue": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The child queues for this limiter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":                 optionalStringAttribute("The unique name for this limiter queue."),
						"number":               schema.Int64Attribute{Computed: true, Description: "A unique number auto-assigned to this queue."},
						"enabled":              optionalBoolAttribute("Enable this limiter queue."),
						"mask":                 optionalStringAttribute("Dynamic pipe scope: none, srcaddress or dstaddress."),
						"maskbits":             optionalIntAttribute("IPv4 mask bits for the dynamic pipe scope (default 32)."),
						"maskbitsv6":           optionalIntAttribute("IPv6 mask bits for the dynamic pipe scope (default 128)."),
						"qlimit":               optionalIntAttribute("The length of the queue."),
						"ecn":                  optionalBoolAttribute("Enable or disable ECN."),
						"description":          optionalStringAttribute("A verbose description for this queue."),
						"aqm":                  optionalStringAttribute("The Active Queue Management algorithm."),
						"param_codel_target":   optionalIntAttribute("CoDel target parameter."),
						"param_codel_interval": optionalIntAttribute("CoDel interval parameter."),
						"param_pie_target":     optionalIntAttribute("PIE target parameter."),
						"param_pie_tupdate":    optionalIntAttribute("PIE tupdate parameter."),
						"param_pie_alpha":      optionalIntAttribute("PIE alpha parameter."),
						"param_pie_beta":       optionalIntAttribute("PIE beta parameter."),
						"param_pie_max_burst":  optionalIntAttribute("PIE max_burst parameter."),
						"param_pie_max_ecnth":  optionalIntAttribute("PIE ecnth parameter."),
						"pie_onoff":            optionalBoolAttribute("Turn PIE on and off depending on queue load."),
						"pie_capdrop":          optionalBoolAttribute("Enable cap drop adjustment."),
						"pie_qdelay":           optionalBoolAttribute("Use timestamps (true) or departure rate estimation (false)."),
						"pie_pderand":          optionalBoolAttribute("Enable drop probability de-randomisation."),
						"param_red_w_q":        optionalIntAttribute("RED w_q parameter."),
						"param_red_min_th":     optionalIntAttribute("RED min_th parameter."),
						"param_red_max_th":     optionalIntAttribute("RED max_th parameter."),
						"param_red_max_p":      optionalIntAttribute("RED max_p parameter."),
						"param_gred_w_q":       optionalIntAttribute("GRED w_q parameter."),
						"param_gred_min_th":    optionalIntAttribute("GRED min_th parameter."),
						"param_gred_max_th":    optionalIntAttribute("GRED max_th parameter."),
						"param_gred_max_p":     optionalIntAttribute("GRED max_p parameter."),
						"weight":               optionalIntAttribute("The share of the parent limiter this queue gets."),
						"plr":                  schema.Float64Attribute{Optional: true, Description: "The amount of packet loss (percentage) added to traffic."},
						"buckets":              optionalIntAttribute("The queue's bucket size (slots)."),
					},
				},
			},
		},
	}
}

func (r *trafficShaperLimiterResource) payload(m trafficShaperLimiterModel) map[string]any {
	p := map[string]any{}
	setString(p, "name", m.Name)
	setBool(p, "enabled", m.Enabled)
	setString(p, "mask", m.Mask)
	setInt(p, "maskbits", m.Maskbits)
	setInt(p, "maskbitsv6", m.Maskbitsv6)
	setInt(p, "qlimit", m.Qlimit)
	setBool(p, "ecn", m.Ecn)
	setString(p, "description", m.Description)
	setString(p, "aqm", m.Aqm)
	setString(p, "sched", m.Sched)
	setInt(p, "param_codel_target", m.ParamCodelTarget)
	setInt(p, "param_codel_interval", m.ParamCodelInterval)
	setInt(p, "param_pie_target", m.ParamPieTarget)
	setInt(p, "param_pie_tupdate", m.ParamPieTupdate)
	setInt(p, "param_pie_alpha", m.ParamPieAlpha)
	setInt(p, "param_pie_beta", m.ParamPieBeta)
	setInt(p, "param_pie_max_burst", m.ParamPieMaxBurst)
	setInt(p, "param_pie_max_ecnth", m.ParamPieMaxEcnth)
	setBool(p, "pie_onoff", m.PieOnoff)
	setBool(p, "pie_capdrop", m.PieCapdrop)
	setBool(p, "pie_qdelay", m.PieQdelay)
	setBool(p, "pie_pderand", m.PiePderand)
	setInt(p, "param_red_w_q", m.ParamRedWQ)
	setInt(p, "param_red_min_th", m.ParamRedMinTh)
	setInt(p, "param_red_max_th", m.ParamRedMaxTh)
	setInt(p, "param_red_max_p", m.ParamRedMaxP)
	setInt(p, "param_gred_w_q", m.ParamGredWQ)
	setInt(p, "param_gred_min_th", m.ParamGredMinTh)
	setInt(p, "param_gred_max_th", m.ParamGredMaxTh)
	setInt(p, "param_gred_max_p", m.ParamGredMaxP)
	setInt(p, "param_fq_codel_target", m.ParamFqCodelTarget)
	setInt(p, "param_fq_codel_interval", m.ParamFqCodelInterval)
	setInt(p, "param_fq_codel_quantum", m.ParamFqCodelQuantum)
	setInt(p, "param_fq_codel_limit", m.ParamFqCodelLimit)
	setInt(p, "param_fq_codel_flows", m.ParamFqCodelFlows)
	setInt(p, "param_fq_pie_target", m.ParamFqPieTarget)
	setInt(p, "param_fq_pie_tupdate", m.ParamFqPieTupdate)
	setInt(p, "param_fq_pie_alpha", m.ParamFqPieAlpha)
	setInt(p, "param_fq_pie_beta", m.ParamFqPieBeta)
	setInt(p, "param_fq_pie_max_burst", m.ParamFqPieMaxBurst)
	setInt(p, "param_fq_pie_max_ecnth", m.ParamFqPieMaxEcnth)
	setInt(p, "param_fq_pie_quantum", m.ParamFqPieQuantum)
	setInt(p, "param_fq_pie_limit", m.ParamFqPieLimit)
	setInt(p, "param_fq_pie_flows", m.ParamFqPieFlows)
	setInt(p, "delay", m.Delay)
	setFloat(p, "plr", m.Plr)
	setInt(p, "buckets", m.Buckets)
	p["bandwidth"] = listObjects(m.Bandwidth)
	p["queue"] = listObjects(m.Queue)
	return p
}

func (r *trafficShaperLimiterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trafficShaperLimiterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, trafficShaperLimiterSingular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper limiter", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper limiter", err.Error())
		return
	}
	plan.Number = intValue(getInt(obj, "number"))
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperLimiterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trafficShaperLimiterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, trafficShaperLimitersPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read traffic shaper limiter", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Number = intValue(getInt(obj, "number"))
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Mask = strValue(getString(obj, "mask"))
	state.Maskbits = intValue(getInt(obj, "maskbits"))
	state.Maskbitsv6 = intValue(getInt(obj, "maskbitsv6"))
	state.Qlimit = intValue(getInt(obj, "qlimit"))
	state.Ecn = boolValue(getBool(obj, "ecn"))
	state.Description = strValue(getString(obj, "description"))
	state.Aqm = strValue(getString(obj, "aqm"))
	state.Sched = strValue(getString(obj, "sched"))
	state.ParamCodelTarget = intValue(getInt(obj, "param_codel_target"))
	state.ParamCodelInterval = intValue(getInt(obj, "param_codel_interval"))
	state.ParamPieTarget = intValue(getInt(obj, "param_pie_target"))
	state.ParamPieTupdate = intValue(getInt(obj, "param_pie_tupdate"))
	state.ParamPieAlpha = intValue(getInt(obj, "param_pie_alpha"))
	state.ParamPieBeta = intValue(getInt(obj, "param_pie_beta"))
	state.ParamPieMaxBurst = intValue(getInt(obj, "param_pie_max_burst"))
	state.ParamPieMaxEcnth = intValue(getInt(obj, "param_pie_max_ecnth"))
	state.PieOnoff = boolValue(getBool(obj, "pie_onoff"))
	state.PieCapdrop = boolValue(getBool(obj, "pie_capdrop"))
	state.PieQdelay = boolValue(getBool(obj, "pie_qdelay"))
	state.PiePderand = boolValue(getBool(obj, "pie_pderand"))
	state.ParamRedWQ = intValue(getInt(obj, "param_red_w_q"))
	state.ParamRedMinTh = intValue(getInt(obj, "param_red_min_th"))
	state.ParamRedMaxTh = intValue(getInt(obj, "param_red_max_th"))
	state.ParamRedMaxP = intValue(getInt(obj, "param_red_max_p"))
	state.ParamGredWQ = intValue(getInt(obj, "param_gred_w_q"))
	state.ParamGredMinTh = intValue(getInt(obj, "param_gred_min_th"))
	state.ParamGredMaxTh = intValue(getInt(obj, "param_gred_max_th"))
	state.ParamGredMaxP = intValue(getInt(obj, "param_gred_max_p"))
	state.ParamFqCodelTarget = intValue(getInt(obj, "param_fq_codel_target"))
	state.ParamFqCodelInterval = intValue(getInt(obj, "param_fq_codel_interval"))
	state.ParamFqCodelQuantum = intValue(getInt(obj, "param_fq_codel_quantum"))
	state.ParamFqCodelLimit = intValue(getInt(obj, "param_fq_codel_limit"))
	state.ParamFqCodelFlows = intValue(getInt(obj, "param_fq_codel_flows"))
	state.ParamFqPieTarget = intValue(getInt(obj, "param_fq_pie_target"))
	state.ParamFqPieTupdate = intValue(getInt(obj, "param_fq_pie_tupdate"))
	state.ParamFqPieAlpha = intValue(getInt(obj, "param_fq_pie_alpha"))
	state.ParamFqPieBeta = intValue(getInt(obj, "param_fq_pie_beta"))
	state.ParamFqPieMaxBurst = intValue(getInt(obj, "param_fq_pie_max_burst"))
	state.ParamFqPieMaxEcnth = intValue(getInt(obj, "param_fq_pie_max_ecnth"))
	state.ParamFqPieQuantum = intValue(getInt(obj, "param_fq_pie_quantum"))
	state.ParamFqPieLimit = intValue(getInt(obj, "param_fq_pie_limit"))
	state.ParamFqPieFlows = intValue(getInt(obj, "param_fq_pie_flows"))
	state.Delay = intValue(getInt(obj, "delay"))
	state.Plr = floatValue(getFloat(obj, "plr"))
	state.Buckets = intValue(getInt(obj, "buckets"))
	state.Bandwidth = objectsToListValue(ctx, getSliceMap(obj, "bandwidth"), types.ObjectType{AttrTypes: limiterBandwidthAttrTypes})
	state.Queue = objectsToListValue(ctx, getSliceMap(obj, "queue"), types.ObjectType{AttrTypes: limiterQueueAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficShaperLimiterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trafficShaperLimiterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, trafficShaperLimitersPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update traffic shaper limiter", err.Error())
		} else {
			resp.Diagnostics.AddError("traffic shaper limiter not found", "limiter "+name+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	raw, err := r.client.Update(ctx, trafficShaperLimiterSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper limiter", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper limiter", err.Error())
		return
	}
	plan.Number = intValue(getInt(obj, "number"))
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperLimiterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trafficShaperLimiterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, trafficShaperLimitersPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper limiter", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, trafficShaperLimiterSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper limiter", err.Error())
	}
}

func (r *trafficShaperLimiterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_queue
// (parent: pfsense_firewall_traffic_shaper, keyed by interface)
// ---------------------------------------------------------------------------

type trafficShaperQueueResource struct{ client *client.Client }

var _ resource.Resource = (*trafficShaperQueueResource)(nil)
var _ resource.ResourceWithConfigure = (*trafficShaperQueueResource)(nil)
var _ resource.ResourceWithImportState = (*trafficShaperQueueResource)(nil)

const (
	trafficShaperQueuesPlural  = "/api/v2/firewall/traffic_shaper/queues"
	trafficShaperQueueSingular = "/api/v2/firewall/traffic_shaper/queue"
)

type trafficShaperQueueModel struct {
	ID            types.String `tfsdk:"id"`
	ParentID      types.String `tfsdk:"parent_id"`
	Interface     types.String `tfsdk:"interface"`
	Enabled       types.Bool   `tfsdk:"enabled"`
	Name          types.String `tfsdk:"name"`
	Priority      types.Int64  `tfsdk:"priority"`
	Qlimit        types.Int64  `tfsdk:"qlimit"`
	Description   types.String `tfsdk:"description"`
	Default       types.Bool   `tfsdk:"default"`
	Red           types.Bool   `tfsdk:"red"`
	Rio           types.Bool   `tfsdk:"rio"`
	Ecn           types.Bool   `tfsdk:"ecn"`
	Codel         types.Bool   `tfsdk:"codel"`
	Bandwidthtype types.String `tfsdk:"bandwidthtype"`
	Bandwidth     types.Int64  `tfsdk:"bandwidth"`
	Buckets       types.Int64  `tfsdk:"buckets"`
	Hogs          types.Int64  `tfsdk:"hogs"`
	Borrow        types.Bool   `tfsdk:"borrow"`
	Upperlimit    types.Bool   `tfsdk:"upperlimit"`
	UpperlimitM1  types.String `tfsdk:"upperlimit_m1"`
	UpperlimitD   types.Int64  `tfsdk:"upperlimit_d"`
	UpperlimitM2  types.String `tfsdk:"upperlimit_m2"`
	Realtime      types.Bool   `tfsdk:"realtime"`
	RealtimeM1    types.String `tfsdk:"realtime_m1"`
	RealtimeD     types.Int64  `tfsdk:"realtime_d"`
	RealtimeM2    types.String `tfsdk:"realtime_m2"`
	Linkshare     types.Bool   `tfsdk:"linkshare"`
	LinkshareM1   types.String `tfsdk:"linkshare_m1"`
	LinkshareD    types.Int64  `tfsdk:"linkshare_d"`
	LinkshareM2   types.String `tfsdk:"linkshare_m2"`
}

func NewTrafficShaperQueueResource() resource.Resource { return &trafficShaperQueueResource{} }

func (r *trafficShaperQueueResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_traffic_shaper_queue"
}
func (r *trafficShaperQueueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *trafficShaperQueueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a traffic shaper queue. Identified by `parent_id` (the parent shaper's interface) and `name`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent traffic shaper's `interface` value (natural key). Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"interface": schema.StringAttribute{
				Computed:    true,
				Description: "The parent interface this queue is a child of, determined by the parent shaper.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name to assign this traffic shaper queue. Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled":       optionalBoolAttribute("Enable this traffic shaper queue."),
			"priority":      optionalIntAttribute("The priority level for this queue (default 1)."),
			"qlimit":        schema.Int64Attribute{Required: true, Description: "The number of packets that can be held in this queue."},
			"description":   optionalStringAttribute("A description for this queue."),
			"default":       optionalBoolAttribute("Mark this queue as the default queue."),
			"red":           optionalBoolAttribute("Use the Random Early Detection scheduler option."),
			"rio":           optionalBoolAttribute("Use the Random Early Detection In and Out scheduler option."),
			"ecn":           optionalBoolAttribute("Use the Explicit Congestion Notification scheduler option."),
			"codel":         optionalBoolAttribute("Use the CoDel Active Queue scheduler option."),
			"bandwidthtype": enumAttribute("The scale type of the `bandwidth` value.", "%", "b", "Kb", "Mb", "Gb"),
			"bandwidth":     schema.Int64Attribute{Required: true, Description: "The total bandwidth amount allowed by this queue."},
			"buckets":       optionalIntAttribute("The queue's bucket size (FAIRQ only)."),
			"hogs":          optionalIntAttribute("The bandwidth limit per host (FAIRQ only)."),
			"borrow":        optionalBoolAttribute("Allow this queue to borrow from other queues (CBQ only)."),
			"upperlimit":    optionalBoolAttribute("Allow setting the maximum bandwidth for the queue (HFSC only)."),
			"upperlimit_m1": optionalStringAttribute("The burst-able bandwidth limit."),
			"upperlimit_d":  optionalIntAttribute("The duration (ms) the burst-able limit is in effect."),
			"upperlimit_m2": requiredStringAttribute("The normal bandwidth limit."),
			"realtime":      optionalBoolAttribute("Allow setting the guaranteed minimum bandwidth (HFSC only)."),
			"realtime_m1":   optionalStringAttribute("The guaranteed minimum bandwidth during real time."),
			"realtime_d":    optionalIntAttribute("The duration (ms) the guaranteed minimum is in effect."),
			"realtime_m2":   requiredStringAttribute("The maximum bandwidth this queue may use."),
			"linkshare":     optionalBoolAttribute("Allow sharing bandwidth from this queue (HFSC only)."),
			"linkshare_m1":  optionalStringAttribute("The initial bandwidth limit when link sharing."),
			"linkshare_d":   optionalIntAttribute("The duration (ms) the initial limit is in effect."),
			"linkshare_m2":  requiredStringAttribute("The maximum bandwidth this queue may use when link sharing."),
		},
	}
}

func (r *trafficShaperQueueResource) payload(m trafficShaperQueueModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enabled", m.Enabled)
	setString(p, "name", m.Name)
	setInt(p, "priority", m.Priority)
	setInt(p, "qlimit", m.Qlimit)
	setString(p, "description", m.Description)
	setBool(p, "default", m.Default)
	setBool(p, "red", m.Red)
	setBool(p, "rio", m.Rio)
	setBool(p, "ecn", m.Ecn)
	setBool(p, "codel", m.Codel)
	setString(p, "bandwidthtype", m.Bandwidthtype)
	setInt(p, "bandwidth", m.Bandwidth)
	setInt(p, "buckets", m.Buckets)
	setInt(p, "hogs", m.Hogs)
	setBool(p, "borrow", m.Borrow)
	setBool(p, "upperlimit", m.Upperlimit)
	setString(p, "upperlimit_m1", m.UpperlimitM1)
	setInt(p, "upperlimit_d", m.UpperlimitD)
	setString(p, "upperlimit_m2", m.UpperlimitM2)
	setBool(p, "realtime", m.Realtime)
	setString(p, "realtime_m1", m.RealtimeM1)
	setInt(p, "realtime_d", m.RealtimeD)
	setString(p, "realtime_m2", m.RealtimeM2)
	setBool(p, "linkshare", m.Linkshare)
	setString(p, "linkshare_m1", m.LinkshareM1)
	setInt(p, "linkshare_d", m.LinkshareD)
	setString(p, "linkshare_m2", m.LinkshareM2)
	return p
}

func (r *trafficShaperQueueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trafficShaperQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	pid, err := resolveParentID(ctx, r.client, trafficShapersPlural, "interface", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent traffic shaper", err.Error())
		return
	}
	payload := r.payload(plan)
	payload["parent_id"] = pid
	raw, err := r.client.Create(ctx, trafficShaperQueueSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper queue", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create traffic shaper queue", err.Error())
		return
	}
	plan.Interface = strValue(getString(obj, "interface"))
	plan.ID = types.StringValue(parentKey + "|" + plan.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperQueueResource) find(ctx context.Context, parentKey, name string) (any, map[string]any, bool, error) {
	pid, err := resolveParentID(ctx, r.client, trafficShapersPlural, "interface", parentKey)
	if err != nil {
		return nil, nil, false, err
	}
	return findByKeyInParent(ctx, r.client, trafficShaperQueuesPlural, formatID(pid), "name", name)
}

func (r *trafficShaperQueueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trafficShaperQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	name := state.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(state.ID.ValueString())
	}
	_, obj, found, err := r.find(ctx, parentKey, name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read traffic shaper queue", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(parentKey + "|" + name)
	state.ParentID = types.StringValue(parentKey)
	state.Interface = strValue(getString(obj, "interface"))
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Name = types.StringValue(name)
	state.Priority = intValue(getInt(obj, "priority"))
	state.Qlimit = intValue(getInt(obj, "qlimit"))
	state.Description = strValue(getString(obj, "description"))
	state.Default = boolValue(getBool(obj, "default"))
	state.Red = boolValue(getBool(obj, "red"))
	state.Rio = boolValue(getBool(obj, "rio"))
	state.Ecn = boolValue(getBool(obj, "ecn"))
	state.Codel = boolValue(getBool(obj, "codel"))
	state.Bandwidthtype = strValue(getString(obj, "bandwidthtype"))
	state.Bandwidth = intValue(getInt(obj, "bandwidth"))
	state.Buckets = intValue(getInt(obj, "buckets"))
	state.Hogs = intValue(getInt(obj, "hogs"))
	state.Borrow = boolValue(getBool(obj, "borrow"))
	state.Upperlimit = boolValue(getBool(obj, "upperlimit"))
	state.UpperlimitM1 = strValue(getString(obj, "upperlimit_m1"))
	state.UpperlimitD = intValue(getInt(obj, "upperlimit_d"))
	state.UpperlimitM2 = strValue(getString(obj, "upperlimit_m2"))
	state.Realtime = boolValue(getBool(obj, "realtime"))
	state.RealtimeM1 = strValue(getString(obj, "realtime_m1"))
	state.RealtimeD = intValue(getInt(obj, "realtime_d"))
	state.RealtimeM2 = strValue(getString(obj, "realtime_m2"))
	state.Linkshare = boolValue(getBool(obj, "linkshare"))
	state.LinkshareM1 = strValue(getString(obj, "linkshare_m1"))
	state.LinkshareD = intValue(getInt(obj, "linkshare_d"))
	state.LinkshareM2 = strValue(getString(obj, "linkshare_m2"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficShaperQueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trafficShaperQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	name := plan.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, trafficShapersPlural, "interface", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent traffic shaper", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, trafficShaperQueuesPlural, formatID(pid), "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update traffic shaper queue", err.Error())
		} else {
			resp.Diagnostics.AddError("traffic shaper queue not found", "queue "+name+" on shaper "+parentKey+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["parent_id"] = pid
	payload["id"] = id
	raw, err := r.client.Update(ctx, trafficShaperQueueSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper queue", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update traffic shaper queue", err.Error())
		return
	}
	plan.Interface = strValue(getString(obj, "interface"))
	plan.ID = types.StringValue(parentKey + "|" + name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperQueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trafficShaperQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	name := state.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, trafficShapersPlural, "interface", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent traffic shaper", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, trafficShaperQueuesPlural, formatID(pid), "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper queue", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, trafficShaperQueueSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete traffic shaper queue", err.Error())
	}
}

func (r *trafficShaperQueueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_limiter_queue
// (parent: pfsense_firewall_traffic_shaper_limiter, keyed by name)
// ---------------------------------------------------------------------------

type trafficShaperLimiterQueueResource struct{ client *client.Client }

var _ resource.Resource = (*trafficShaperLimiterQueueResource)(nil)
var _ resource.ResourceWithConfigure = (*trafficShaperLimiterQueueResource)(nil)
var _ resource.ResourceWithImportState = (*trafficShaperLimiterQueueResource)(nil)

const (
	trafficShaperLimiterQueuesPlural  = "/api/v2/firewall/traffic_shaper/limiter/queues"
	trafficShaperLimiterQueueSingular = "/api/v2/firewall/traffic_shaper/limiter/queue"
)

type trafficShaperLimiterQueueModel struct {
	ID                 types.String  `tfsdk:"id"`
	ParentID           types.String  `tfsdk:"parent_id"`
	Name               types.String  `tfsdk:"name"`
	Number             types.Int64   `tfsdk:"number"`
	Enabled            types.Bool    `tfsdk:"enabled"`
	Mask               types.String  `tfsdk:"mask"`
	Maskbits           types.Int64   `tfsdk:"maskbits"`
	Maskbitsv6         types.Int64   `tfsdk:"maskbitsv6"`
	Qlimit             types.Int64   `tfsdk:"qlimit"`
	Ecn                types.Bool    `tfsdk:"ecn"`
	Description        types.String  `tfsdk:"description"`
	Aqm                types.String  `tfsdk:"aqm"`
	ParamCodelTarget   types.Int64   `tfsdk:"param_codel_target"`
	ParamCodelInterval types.Int64   `tfsdk:"param_codel_interval"`
	ParamPieTarget     types.Int64   `tfsdk:"param_pie_target"`
	ParamPieTupdate    types.Int64   `tfsdk:"param_pie_tupdate"`
	ParamPieAlpha      types.Int64   `tfsdk:"param_pie_alpha"`
	ParamPieBeta       types.Int64   `tfsdk:"param_pie_beta"`
	ParamPieMaxBurst   types.Int64   `tfsdk:"param_pie_max_burst"`
	ParamPieMaxEcnth   types.Int64   `tfsdk:"param_pie_max_ecnth"`
	PieOnoff           types.Bool    `tfsdk:"pie_onoff"`
	PieCapdrop         types.Bool    `tfsdk:"pie_capdrop"`
	PieQdelay          types.Bool    `tfsdk:"pie_qdelay"`
	PiePderand         types.Bool    `tfsdk:"pie_pderand"`
	ParamRedWQ         types.Int64   `tfsdk:"param_red_w_q"`
	ParamRedMinTh      types.Int64   `tfsdk:"param_red_min_th"`
	ParamRedMaxTh      types.Int64   `tfsdk:"param_red_max_th"`
	ParamRedMaxP       types.Int64   `tfsdk:"param_red_max_p"`
	ParamGredWQ        types.Int64   `tfsdk:"param_gred_w_q"`
	ParamGredMinTh     types.Int64   `tfsdk:"param_gred_min_th"`
	ParamGredMaxTh     types.Int64   `tfsdk:"param_gred_max_th"`
	ParamGredMaxP      types.Int64   `tfsdk:"param_gred_max_p"`
	Weight             types.Int64   `tfsdk:"weight"`
	Plr                types.Float64 `tfsdk:"plr"`
	Buckets            types.Int64   `tfsdk:"buckets"`
}

func NewTrafficShaperLimiterQueueResource() resource.Resource {
	return &trafficShaperLimiterQueueResource{}
}

func (r *trafficShaperLimiterQueueResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_traffic_shaper_limiter_queue"
}
func (r *trafficShaperLimiterQueueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *trafficShaperLimiterQueueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a traffic shaper limiter queue. Identified by `parent_id` (the parent limiter's name) and `name`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent limiter's `name` value (natural key). Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The unique name for this limiter queue. Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"number":      schema.Int64Attribute{Computed: true, Description: "A unique number auto-assigned to this queue."},
			"enabled":     optionalBoolAttribute("Enable this limiter queue."),
			"mask":        enumAttribute("Dynamic pipe scope: none, srcaddress or dstaddress.", "none", "srcaddress", "dstaddress"),
			"maskbits":    optionalIntAttribute("IPv4 mask bits for the dynamic pipe scope (default 32)."),
			"maskbitsv6":  optionalIntAttribute("IPv6 mask bits for the dynamic pipe scope (default 128)."),
			"qlimit":      optionalIntAttribute("The length of this queue."),
			"ecn":         optionalBoolAttribute("Enable or disable ECN."),
			"description": optionalStringAttribute("A verbose description for this queue."),
			"aqm": schema.StringAttribute{
				Required:    true,
				Description: "The Active Queue Management algorithm: droptail, codel, pie, red or gred.",
				Validators:  []validator.String{stringvalidator.OneOf("droptail", "codel", "pie", "red", "gred")},
			},
			"param_codel_target":   optionalIntAttribute("CoDel target parameter."),
			"param_codel_interval": optionalIntAttribute("CoDel interval parameter."),
			"param_pie_target":     optionalIntAttribute("PIE target parameter."),
			"param_pie_tupdate":    optionalIntAttribute("PIE tupdate parameter."),
			"param_pie_alpha":      optionalIntAttribute("PIE alpha parameter."),
			"param_pie_beta":       optionalIntAttribute("PIE beta parameter."),
			"param_pie_max_burst":  optionalIntAttribute("PIE max_burst parameter."),
			"param_pie_max_ecnth":  optionalIntAttribute("PIE ecnth parameter."),
			"pie_onoff":            optionalBoolAttribute("Turn PIE on and off depending on queue load."),
			"pie_capdrop":          optionalBoolAttribute("Enable cap drop adjustment."),
			"pie_qdelay":           optionalBoolAttribute("Use timestamps (true) or departure rate estimation (false)."),
			"pie_pderand":          optionalBoolAttribute("Enable drop probability de-randomisation."),
			"param_red_w_q":        optionalIntAttribute("RED w_q parameter."),
			"param_red_min_th":     optionalIntAttribute("RED min_th parameter."),
			"param_red_max_th":     optionalIntAttribute("RED max_th parameter."),
			"param_red_max_p":      optionalIntAttribute("RED max_p parameter."),
			"param_gred_w_q":       optionalIntAttribute("GRED w_q parameter."),
			"param_gred_min_th":    optionalIntAttribute("GRED min_th parameter."),
			"param_gred_max_th":    optionalIntAttribute("GRED max_th parameter."),
			"param_gred_max_p":     optionalIntAttribute("GRED max_p parameter."),
			"weight":               optionalIntAttribute("The share of the parent limiter this queue gets."),
			"plr":                  schema.Float64Attribute{Optional: true, Description: "The amount of packet loss (percentage) added to traffic."},
			"buckets":              optionalIntAttribute("The queue's bucket size (slots)."),
		},
	}
}

func (r *trafficShaperLimiterQueueResource) payload(m trafficShaperLimiterQueueModel) map[string]any {
	p := map[string]any{}
	setString(p, "name", m.Name)
	setBool(p, "enabled", m.Enabled)
	setString(p, "mask", m.Mask)
	setInt(p, "maskbits", m.Maskbits)
	setInt(p, "maskbitsv6", m.Maskbitsv6)
	setInt(p, "qlimit", m.Qlimit)
	setBool(p, "ecn", m.Ecn)
	setString(p, "description", m.Description)
	setString(p, "aqm", m.Aqm)
	setInt(p, "param_codel_target", m.ParamCodelTarget)
	setInt(p, "param_codel_interval", m.ParamCodelInterval)
	setInt(p, "param_pie_target", m.ParamPieTarget)
	setInt(p, "param_pie_tupdate", m.ParamPieTupdate)
	setInt(p, "param_pie_alpha", m.ParamPieAlpha)
	setInt(p, "param_pie_beta", m.ParamPieBeta)
	setInt(p, "param_pie_max_burst", m.ParamPieMaxBurst)
	setInt(p, "param_pie_max_ecnth", m.ParamPieMaxEcnth)
	setBool(p, "pie_onoff", m.PieOnoff)
	setBool(p, "pie_capdrop", m.PieCapdrop)
	setBool(p, "pie_qdelay", m.PieQdelay)
	setBool(p, "pie_pderand", m.PiePderand)
	setInt(p, "param_red_w_q", m.ParamRedWQ)
	setInt(p, "param_red_min_th", m.ParamRedMinTh)
	setInt(p, "param_red_max_th", m.ParamRedMaxTh)
	setInt(p, "param_red_max_p", m.ParamRedMaxP)
	setInt(p, "param_gred_w_q", m.ParamGredWQ)
	setInt(p, "param_gred_min_th", m.ParamGredMinTh)
	setInt(p, "param_gred_max_th", m.ParamGredMaxTh)
	setInt(p, "param_gred_max_p", m.ParamGredMaxP)
	setInt(p, "weight", m.Weight)
	setFloat(p, "plr", m.Plr)
	setInt(p, "buckets", m.Buckets)
	return p
}

func (r *trafficShaperLimiterQueueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan trafficShaperLimiterQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	pid, err := resolveParentID(ctx, r.client, trafficShaperLimitersPlural, "name", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent limiter", err.Error())
		return
	}
	payload := r.payload(plan)
	payload["parent_id"] = pid
	raw, err := r.client.Create(ctx, trafficShaperLimiterQueueSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to create limiter queue", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create limiter queue", err.Error())
		return
	}
	plan.Number = intValue(getInt(obj, "number"))
	plan.ID = types.StringValue(parentKey + "|" + plan.Name.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperLimiterQueueResource) find(ctx context.Context, parentKey, name string) (any, map[string]any, bool, error) {
	pid, err := resolveParentID(ctx, r.client, trafficShaperLimitersPlural, "name", parentKey)
	if err != nil {
		return nil, nil, false, err
	}
	return findByKeyInParent(ctx, r.client, trafficShaperLimiterQueuesPlural, formatID(pid), "name", name)
}

func (r *trafficShaperLimiterQueueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state trafficShaperLimiterQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	name := state.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(state.ID.ValueString())
	}
	_, obj, found, err := r.find(ctx, parentKey, name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read limiter queue", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(parentKey + "|" + name)
	state.ParentID = types.StringValue(parentKey)
	state.Name = types.StringValue(name)
	state.Number = intValue(getInt(obj, "number"))
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Mask = strValue(getString(obj, "mask"))
	state.Maskbits = intValue(getInt(obj, "maskbits"))
	state.Maskbitsv6 = intValue(getInt(obj, "maskbitsv6"))
	state.Qlimit = intValue(getInt(obj, "qlimit"))
	state.Ecn = boolValue(getBool(obj, "ecn"))
	state.Description = strValue(getString(obj, "description"))
	state.Aqm = strValue(getString(obj, "aqm"))
	state.ParamCodelTarget = intValue(getInt(obj, "param_codel_target"))
	state.ParamCodelInterval = intValue(getInt(obj, "param_codel_interval"))
	state.ParamPieTarget = intValue(getInt(obj, "param_pie_target"))
	state.ParamPieTupdate = intValue(getInt(obj, "param_pie_tupdate"))
	state.ParamPieAlpha = intValue(getInt(obj, "param_pie_alpha"))
	state.ParamPieBeta = intValue(getInt(obj, "param_pie_beta"))
	state.ParamPieMaxBurst = intValue(getInt(obj, "param_pie_max_burst"))
	state.ParamPieMaxEcnth = intValue(getInt(obj, "param_pie_max_ecnth"))
	state.PieOnoff = boolValue(getBool(obj, "pie_onoff"))
	state.PieCapdrop = boolValue(getBool(obj, "pie_capdrop"))
	state.PieQdelay = boolValue(getBool(obj, "pie_qdelay"))
	state.PiePderand = boolValue(getBool(obj, "pie_pderand"))
	state.ParamRedWQ = intValue(getInt(obj, "param_red_w_q"))
	state.ParamRedMinTh = intValue(getInt(obj, "param_red_min_th"))
	state.ParamRedMaxTh = intValue(getInt(obj, "param_red_max_th"))
	state.ParamRedMaxP = intValue(getInt(obj, "param_red_max_p"))
	state.ParamGredWQ = intValue(getInt(obj, "param_gred_w_q"))
	state.ParamGredMinTh = intValue(getInt(obj, "param_gred_min_th"))
	state.ParamGredMaxTh = intValue(getInt(obj, "param_gred_max_th"))
	state.ParamGredMaxP = intValue(getInt(obj, "param_gred_max_p"))
	state.Weight = intValue(getInt(obj, "weight"))
	state.Plr = floatValue(getFloat(obj, "plr"))
	state.Buckets = intValue(getInt(obj, "buckets"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *trafficShaperLimiterQueueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan trafficShaperLimiterQueueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	name := plan.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, trafficShaperLimitersPlural, "name", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent limiter", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, trafficShaperLimiterQueuesPlural, formatID(pid), "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update limiter queue", err.Error())
		} else {
			resp.Diagnostics.AddError("limiter queue not found", "queue "+name+" on limiter "+parentKey+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["parent_id"] = pid
	payload["id"] = id
	raw, err := r.client.Update(ctx, trafficShaperLimiterQueueSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update limiter queue", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update limiter queue", err.Error())
		return
	}
	plan.Number = intValue(getInt(obj, "number"))
	plan.ID = types.StringValue(parentKey + "|" + name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *trafficShaperLimiterQueueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state trafficShaperLimiterQueueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	name := state.Name.ValueString()
	if parentKey == "" || name == "" {
		parentKey, name = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, trafficShaperLimitersPlural, "name", parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent limiter", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, trafficShaperLimiterQueuesPlural, formatID(pid), "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete limiter queue", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, trafficShaperLimiterQueueSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete limiter queue", err.Error())
	}
}

func (r *trafficShaperLimiterQueueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
