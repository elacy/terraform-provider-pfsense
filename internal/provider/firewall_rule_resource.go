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

// firewallRuleResource implements pfsense_firewall_rule.
//
// pfSense firewall rules have no persistent unique identifier in the REST API
// (object IDs are array indices). This resource therefore requires a unique
// `descr` value, which is used as the natural key to locate the rule.
type firewallRuleResource struct {
	client *client.Client
}

var _ resource.Resource = (*firewallRuleResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallRuleResource)(nil)
var _ resource.ResourceWithImportState = (*firewallRuleResource)(nil)

const (
	firewallRulePlural   = "/api/v2/firewall/rules"
	firewallRuleSingular = "/api/v2/firewall/rule"
)

type firewallRuleModel struct {
	ID              types.String `tfsdk:"id"`
	Type            types.String `tfsdk:"type"`
	Interface       types.List   `tfsdk:"interface"`
	IPProtocol      types.String `tfsdk:"ipprotocol"`
	Protocol        types.String `tfsdk:"protocol"`
	ICMPType        types.List   `tfsdk:"icmptype"`
	Source          types.String `tfsdk:"source"`
	SourcePort      types.String `tfsdk:"source_port"`
	Destination     types.String `tfsdk:"destination"`
	DestinationPort types.String `tfsdk:"destination_port"`
	Descr           types.String `tfsdk:"descr"`
	Disabled        types.Bool   `tfsdk:"disabled"`
	Log             types.Bool   `tfsdk:"log"`
	Dscp            types.String `tfsdk:"dscp"`
	Tag             types.String `tfsdk:"tag"`
	StateType       types.String `tfsdk:"statetype"`
	TCPFlagsAny     types.Bool   `tfsdk:"tcp_flags_any"`
	TCPFlagsOutOf   types.List   `tfsdk:"tcp_flags_out_of"`
	TCPFlagsSet     types.List   `tfsdk:"tcp_flags_set"`
	Gateway         types.String `tfsdk:"gateway"`
	Sched           types.String `tfsdk:"sched"`
	DnPipe          types.String `tfsdk:"dnpipe"`
	PdPipe          types.String `tfsdk:"pdnpipe"`
	DefaultQueue    types.String `tfsdk:"defaultqueue"`
	AckQueue        types.String `tfsdk:"ackqueue"`
	Quick           types.Bool   `tfsdk:"quick"`
	Direction       types.String `tfsdk:"direction"`
}

func NewFirewallRuleResource() resource.Resource {
	return &firewallRuleResource{}
}

func (r *firewallRuleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_rule"
}

func (r *firewallRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}

func (r *firewallRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall rule. Rules are identified by a unique `descr` value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The rule description (rules are identified by description).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The rule action: pass, block or reject.",
				Validators:  []validator.String{stringvalidator.OneOf("pass", "block", "reject")},
			},
			"interface": requiredStringListAttribute("The interface(s) this rule applies to (e.g. `lan`, `wan`)."),
			"ipprotocol": schema.StringAttribute{
				Required:    true,
				Description: "The address family: inet or inet6.",
				Validators:  []validator.String{stringvalidator.OneOf("inet", "inet6")},
			},
			"protocol": schema.StringAttribute{
				Optional:    true,
				Description: "The protocol this rule matches (tcp, udp, icmp, ...).",
				Validators: []validator.String{
					stringvalidator.OneOf("any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "igmp", "pim", "ospf", "ipv6", "carp", "pfsync"),
				},
			},
			"icmptype": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "ICMP types this rule matches (when protocol is icmp).",
			},
			"source": schema.StringAttribute{
				Required:    true,
				Description: "Source address: IP, subnet CIDR, alias name, interface, `any`, `(self)`, or negated with `!`.",
			},
			"source_port": schema.StringAttribute{
				Optional:    true,
				Description: "Source port or range (`port` or `start:end`), or port alias.",
			},
			"destination": schema.StringAttribute{
				Required:    true,
				Description: "Destination address: IP, subnet CIDR, alias name, interface, `any`, `(self)`, or negated with `!`.",
			},
			"destination_port": schema.StringAttribute{
				Optional:    true,
				Description: "Destination port or range (`port` or `start:end`), or port alias.",
			},
			"descr": schema.StringAttribute{
				Required:    true,
				Description: "A unique description. Used to identify the rule; must be unique across all rules.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disabled": schema.BoolAttribute{
				Optional:    true,
				Description: "Create the rule disabled.",
			},
			"log": schema.BoolAttribute{
				Optional:    true,
				Description: "Log packets that match this rule.",
			},
			"dscp": schema.StringAttribute{
				Optional:    true,
				Description: "Differentiated Services Code Point value.",
			},
			"tag": schema.StringAttribute{
				Optional:    true,
				Description: "Packet filter tag for this rule.",
			},
			"statetype": schema.StringAttribute{
				Optional:    true,
				Description: "State tracking behaviour (default `keep state`).",
			},
			"tcp_flags_any": schema.BoolAttribute{
				Optional:    true,
				Description: "Match any of the specified TCP flags.",
			},
			"tcp_flags_out_of": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "TCP flags to match out of.",
			},
			"tcp_flags_set": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "TCP flags that must be set.",
			},
			"gateway": schema.StringAttribute{
				Optional:    true,
				Description: "Gateway name to route matching traffic through.",
			},
			"sched": schema.StringAttribute{
				Optional:    true,
				Description: "Firewall schedule name that gates this rule.",
			},
			"dnpipe": schema.StringAttribute{
				Optional:    true,
				Description: "Download limiter/queue name.",
			},
			"pdnpipe": schema.StringAttribute{
				Optional:    true,
				Description: "Upload limiter/queue name.",
			},
			"defaultqueue": schema.StringAttribute{
				Optional:    true,
				Description: "Default queue name for the limiter.",
			},
			"ackqueue": schema.StringAttribute{
				Optional:    true,
				Description: "ACK queue name for the limiter.",
			},
			"quick": schema.BoolAttribute{
				Optional:    true,
				Description: "Apply the rule immediately and stop processing subsequent rules on match.",
			},
			"direction": schema.StringAttribute{
				Optional:    true,
				Description: "Match direction: any, in or out (default `any`).",
				Validators:  []validator.String{stringvalidator.OneOf("any", "in", "out")},
			},
		},
	}
}

func (r *firewallRuleResource) payload(plan firewallRuleModel) map[string]any {
	p := map[string]any{}
	setString(p, "type", plan.Type)
	setStringList(p, "interface", plan.Interface)
	setString(p, "ipprotocol", plan.IPProtocol)
	setString(p, "protocol", plan.Protocol)
	setStringList(p, "icmptype", plan.ICMPType)
	setString(p, "source", plan.Source)
	setString(p, "source_port", plan.SourcePort)
	setString(p, "destination", plan.Destination)
	setString(p, "destination_port", plan.DestinationPort)
	setString(p, "descr", plan.Descr)
	setBool(p, "disabled", plan.Disabled)
	setBool(p, "log", plan.Log)
	setString(p, "dscp", plan.Dscp)
	setString(p, "tag", plan.Tag)
	setString(p, "statetype", plan.StateType)
	setBool(p, "tcp_flags_any", plan.TCPFlagsAny)
	setStringList(p, "tcp_flags_out_of", plan.TCPFlagsOutOf)
	setStringList(p, "tcp_flags_set", plan.TCPFlagsSet)
	setString(p, "gateway", plan.Gateway)
	setString(p, "sched", plan.Sched)
	setString(p, "dnpipe", plan.DnPipe)
	setString(p, "pdnpipe", plan.PdPipe)
	setString(p, "defaultqueue", plan.DefaultQueue)
	setString(p, "ackqueue", plan.AckQueue)
	setBool(p, "quick", plan.Quick)
	setString(p, "direction", plan.Direction)
	return p
}

func (r *firewallRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	payload := r.payload(plan)
	applyNow(payload)
	if _, err := r.client.Create(ctx, firewallRuleSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create firewall rule", err.Error())
		return
	}
	plan.ID = plan.Descr
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}

	_, obj, found, err := findByKey(ctx, r.client, firewallRulePlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read firewall rule", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(descr)
	state.Descr = types.StringValue(descr)
	state.Type = strValue(getString(obj, "type"))
	state.Interface = strListValue(ctx, getStringSlice(obj, "interface"))
	state.IPProtocol = strValue(getString(obj, "ipprotocol"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.ICMPType = strListValue(ctx, getStringSlice(obj, "icmptype"))
	state.Source = strValue(getString(obj, "source"))
	state.SourcePort = strValue(getString(obj, "source_port"))
	state.Destination = strValue(getString(obj, "destination"))
	state.DestinationPort = strValue(getString(obj, "destination_port"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.Log = boolValue(getBool(obj, "log"))
	state.Dscp = strValue(getString(obj, "dscp"))
	state.Tag = strValue(getString(obj, "tag"))
	state.StateType = strValue(getString(obj, "statetype"))
	state.TCPFlagsAny = boolValue(getBool(obj, "tcp_flags_any"))
	state.TCPFlagsOutOf = strListValue(ctx, getStringSlice(obj, "tcp_flags_out_of"))
	state.TCPFlagsSet = strListValue(ctx, getStringSlice(obj, "tcp_flags_set"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.Sched = strValue(getString(obj, "sched"))
	state.DnPipe = strValue(getString(obj, "dnpipe"))
	state.PdPipe = strValue(getString(obj, "pdnpipe"))
	state.DefaultQueue = strValue(getString(obj, "defaultqueue"))
	state.AckQueue = strValue(getString(obj, "ackqueue"))
	state.Quick = boolValue(getBool(obj, "quick"))
	state.Direction = strValue(getString(obj, "direction"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}

	id, _, found, err := findByKey(ctx, r.client, firewallRulePlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to update firewall rule", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("firewall rule not found", "rule "+descr+" no longer exists")
		return
	}

	payload := r.payload(plan)
	payload["id"] = id
	applyNow(payload)

	if _, err := r.client.Update(ctx, firewallRuleSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update firewall rule", err.Error())
		return
	}
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}

	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}

	id, _, found, err := findByKey(ctx, r.client, firewallRulePlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete firewall rule", err.Error())
		return
	}
	if !found {
		return
	}

	q := client.Query{}.Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, firewallRuleSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete firewall rule", err.Error())
	}
}

func (r *firewallRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
