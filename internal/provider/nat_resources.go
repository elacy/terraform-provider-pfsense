package provider

import (
	"context"
	"strings"

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

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_port_forward
// ---------------------------------------------------------------------------

type firewallNATPortForwardResource struct{ client *client.Client }

var _ resource.Resource = (*firewallNATPortForwardResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallNATPortForwardResource)(nil)
var _ resource.ResourceWithImportState = (*firewallNATPortForwardResource)(nil)

const (
	firewallNATPortForwardPlural   = "/api/v2/firewall/nat/port_forwards"
	firewallNATPortForwardSingular = "/api/v2/firewall/nat/port_forward"
)

type firewallNATPortForwardModel struct {
	ID               types.String `tfsdk:"id"`
	Interface        types.String `tfsdk:"interface"`
	IPProtocol       types.String `tfsdk:"ipprotocol"`
	Protocol         types.String `tfsdk:"protocol"`
	Source           types.String `tfsdk:"source"`
	SourcePort       types.String `tfsdk:"source_port"`
	Destination      types.String `tfsdk:"destination"`
	DestinationPort  types.String `tfsdk:"destination_port"`
	Target           types.String `tfsdk:"target"`
	LocalPort        types.String `tfsdk:"local_port"`
	Disabled         types.Bool   `tfsdk:"disabled"`
	NoRDR            types.Bool   `tfsdk:"nordr"`
	NoSync           types.Bool   `tfsdk:"nosync"`
	Descr            types.String `tfsdk:"descr"`
	NATReflection    types.String `tfsdk:"natreflection"`
	AssociatedRuleID types.String `tfsdk:"associated_rule_id"`
	CreatedTime      types.Int64  `tfsdk:"created_time"`
	CreatedBy        types.String `tfsdk:"created_by"`
	UpdatedTime      types.Int64  `tfsdk:"updated_time"`
	UpdatedBy        types.String `tfsdk:"updated_by"`
}

func NewFirewallNATPortForwardResource() resource.Resource { return &firewallNATPortForwardResource{} }

func (r *firewallNATPortForwardResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_nat_port_forward"
}
func (r *firewallNATPortForwardResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *firewallNATPortForwardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall NAT port forward rule. Identified by the combination of `interface`, `protocol`, `destination` and `target`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"interface": schema.StringAttribute{
				Required:    true,
				Description: "The interface this port forward rule applies to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ipprotocol": enumAttribute("The IP protocol this port forward rule should match: inet, inet6 or inet46.", "inet", "inet6", "inet46"),
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "The IP/transport protocol this port forward rule should match.",
				Validators: []validator.String{
					stringvalidator.OneOf("any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "ipv6", "igmp", "pim", "ospf"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source":      requiredStringAttribute("The source address this port forward rule applies to."),
			"source_port": optionalStringAttribute("The source port this port forward rule applies to."),
			"destination": schema.StringAttribute{
				Required:    true,
				Description: "The destination address this port forward rule applies to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination_port": optionalStringAttribute("The destination port this port forward rule applies to."),
			"target": schema.StringAttribute{
				Required:    true,
				Description: "The IP address or alias of the internal host to forward matching traffic to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"local_port":         requiredStringAttribute("The port on the internal host to forward matching traffic to."),
			"disabled":           optionalBoolAttribute("Disable this port forward rule."),
			"nordr":              optionalBoolAttribute("Disable redirection for traffic matching this rule."),
			"nosync":             optionalBoolAttribute("Prevent this rule from being synced to non-primary CARP members."),
			"descr":              optionalStringAttribute("A description for this port forward rule."),
			"natreflection":      enumAttribute("NAT reflection mode for this rule: enable, disable or purenat.", "enable", "disable", "purenat"),
			"associated_rule_id": optionalStringAttribute("Associated firewall rule mode or id (empty string, `new`, `pass`, or an existing rule id)."),
			"created_time":       schema.Int64Attribute{Computed: true, Description: "Unix timestamp of when this rule was originally created."},
			"created_by":         schema.StringAttribute{Computed: true, Description: "Username and IP of the user who originally created this rule."},
			"updated_time":       schema.Int64Attribute{Computed: true, Description: "Unix timestamp of when this rule was last updated."},
			"updated_by":         schema.StringAttribute{Computed: true, Description: "Username and IP of the user who last updated this rule."},
		},
	}
}

func (r *firewallNATPortForwardResource) key(iface, protocol, destination, target string) string {
	return iface + "|" + protocol + "|" + destination + "|" + target
}

func (r *firewallNATPortForwardResource) find(ctx context.Context, iface, protocol, destination, target string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, firewallNATPortForwardPlural, map[string]string{
		"interface":   iface,
		"protocol":    protocol,
		"destination": destination,
		"target":      target,
	})
}

func (r *firewallNATPortForwardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallNATPortForwardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "interface", plan.Interface)
	setString(payload, "ipprotocol", plan.IPProtocol)
	setString(payload, "protocol", plan.Protocol)
	setString(payload, "source", plan.Source)
	setString(payload, "source_port", plan.SourcePort)
	setString(payload, "destination", plan.Destination)
	setString(payload, "destination_port", plan.DestinationPort)
	setString(payload, "target", plan.Target)
	setString(payload, "local_port", plan.LocalPort)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nordr", plan.NoRDR)
	setBool(payload, "nosync", plan.NoSync)
	setString(payload, "descr", plan.Descr)
	setString(payload, "natreflection", plan.NATReflection)
	setString(payload, "associated_rule_id", plan.AssociatedRuleID)
	if _, err := r.client.Create(ctx, firewallNATPortForwardSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create port forward rule", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan.Interface.ValueString(), plan.Protocol.ValueString(), plan.Destination.ValueString(), plan.Target.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATPortForwardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallNATPortForwardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	protocol := state.Protocol.ValueString()
	destination := state.Destination.ValueString()
	target := state.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(state.ID.ValueString(), 4)
		iface, protocol, destination, target = parts[0], parts[1], parts[2], parts[3]
	}
	_, obj, found, err := r.find(ctx, iface, protocol, destination, target)
	if err != nil {
		resp.Diagnostics.AddError("failed to read port forward rule", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, protocol, destination, target))
	state.Interface = strValue(getString(obj, "interface"))
	state.IPProtocol = strValue(getString(obj, "ipprotocol"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.Source = strValue(getString(obj, "source"))
	state.SourcePort = strValue(getString(obj, "source_port"))
	state.Destination = strValue(getString(obj, "destination"))
	state.DestinationPort = strValue(getString(obj, "destination_port"))
	state.Target = strValue(getString(obj, "target"))
	state.LocalPort = strValue(getString(obj, "local_port"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.NoRDR = boolValue(getBool(obj, "nordr"))
	state.NoSync = boolValue(getBool(obj, "nosync"))
	state.Descr = strValue(getString(obj, "descr"))
	state.NATReflection = strValue(getString(obj, "natreflection"))
	state.AssociatedRuleID = strValue(getString(obj, "associated_rule_id"))
	state.CreatedTime = intValue(getInt(obj, "created_time"))
	state.CreatedBy = strValue(getString(obj, "created_by"))
	state.UpdatedTime = intValue(getInt(obj, "updated_time"))
	state.UpdatedBy = strValue(getString(obj, "updated_by"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallNATPortForwardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallNATPortForwardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.Interface.ValueString()
	protocol := plan.Protocol.ValueString()
	destination := plan.Destination.ValueString()
	target := plan.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(plan.ID.ValueString(), 4)
		iface, protocol, destination, target = parts[0], parts[1], parts[2], parts[3]
	}
	id, _, found, err := r.find(ctx, iface, protocol, destination, target)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update port forward rule", err.Error())
		} else {
			resp.Diagnostics.AddError("port forward rule not found", "rule "+iface+"|"+protocol+"|"+destination+"|"+target+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "interface", plan.Interface)
	setString(payload, "ipprotocol", plan.IPProtocol)
	setString(payload, "protocol", plan.Protocol)
	setString(payload, "source", plan.Source)
	setString(payload, "source_port", plan.SourcePort)
	setString(payload, "destination", plan.Destination)
	setString(payload, "destination_port", plan.DestinationPort)
	setString(payload, "target", plan.Target)
	setString(payload, "local_port", plan.LocalPort)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nordr", plan.NoRDR)
	setBool(payload, "nosync", plan.NoSync)
	setString(payload, "descr", plan.Descr)
	setString(payload, "natreflection", plan.NATReflection)
	setString(payload, "associated_rule_id", plan.AssociatedRuleID)
	if _, err := r.client.Update(ctx, firewallNATPortForwardSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update port forward rule", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, protocol, destination, target))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATPortForwardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallNATPortForwardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	protocol := state.Protocol.ValueString()
	destination := state.Destination.ValueString()
	target := state.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(state.ID.ValueString(), 4)
		iface, protocol, destination, target = parts[0], parts[1], parts[2], parts[3]
	}
	id, _, found, err := r.find(ctx, iface, protocol, destination, target)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete port forward rule", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, firewallNATPortForwardSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete port forward rule", err.Error())
	}
}

func (r *firewallNATPortForwardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_one_to_one
// ---------------------------------------------------------------------------

type firewallNATOneToOneResource struct{ client *client.Client }

var _ resource.Resource = (*firewallNATOneToOneResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallNATOneToOneResource)(nil)
var _ resource.ResourceWithImportState = (*firewallNATOneToOneResource)(nil)

const (
	firewallNATOneToOnePlural   = "/api/v2/firewall/nat/one_to_one/mappings"
	firewallNATOneToOneSingular = "/api/v2/firewall/nat/one_to_one/mapping"
)

type firewallNATOneToOneModel struct {
	ID            types.String `tfsdk:"id"`
	Interface     types.String `tfsdk:"interface"`
	Disabled      types.Bool   `tfsdk:"disabled"`
	NoBINAT       types.Bool   `tfsdk:"nobinat"`
	NATReflection types.String `tfsdk:"natreflection"`
	IPProtocol    types.String `tfsdk:"ipprotocol"`
	External      types.String `tfsdk:"external"`
	Source        types.String `tfsdk:"source"`
	Destination   types.String `tfsdk:"destination"`
	Descr         types.String `tfsdk:"descr"`
}

func NewFirewallNATOneToOneResource() resource.Resource { return &firewallNATOneToOneResource{} }

func (r *firewallNATOneToOneResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_nat_one_to_one"
}
func (r *firewallNATOneToOneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *firewallNATOneToOneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall NAT 1:1 mapping. Identified by the combination of `interface` and `external`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"interface": schema.StringAttribute{
				Required:    true,
				Description: "The interface this 1:1 NAT mapping applies to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disabled":      optionalBoolAttribute("Disable this 1:1 NAT mapping."),
			"nobinat":       optionalBoolAttribute("Exclude traffic matching this mapping from a later, more general, mapping."),
			"natreflection": enumAttribute("NAT reflection mode for this mapping: enable or disable.", "enable", "disable"),
			"ipprotocol":    enumAttribute("The IP version this mapping applies to: inet or inet6.", "inet", "inet6"),
			"external": schema.StringAttribute{
				Required:    true,
				Description: "The external IP address or interface for the 1:1 mapping.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source":      requiredStringAttribute("The source IP address or subnet that traffic must match to apply this mapping."),
			"destination": requiredStringAttribute("The destination IP address or subnet that traffic must match to apply this mapping."),
			"descr":       optionalStringAttribute("A description for this 1:1 NAT mapping."),
		},
	}
}

func (r *firewallNATOneToOneResource) key(iface, external string) string {
	return iface + "|" + external
}

func (r *firewallNATOneToOneResource) find(ctx context.Context, iface, external string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, firewallNATOneToOnePlural, map[string]string{
		"interface": iface,
		"external":  external,
	})
}

func (r *firewallNATOneToOneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallNATOneToOneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "interface", plan.Interface)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nobinat", plan.NoBINAT)
	setString(payload, "natreflection", plan.NATReflection)
	setString(payload, "ipprotocol", plan.IPProtocol)
	setString(payload, "external", plan.External)
	setString(payload, "source", plan.Source)
	setString(payload, "destination", plan.Destination)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, firewallNATOneToOneSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create 1:1 NAT mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan.Interface.ValueString(), plan.External.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATOneToOneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallNATOneToOneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	external := state.External.ValueString()
	if iface == "" {
		iface, external = splitRouteKey(state.ID.ValueString())
	}
	_, obj, found, err := r.find(ctx, iface, external)
	if err != nil {
		resp.Diagnostics.AddError("failed to read 1:1 NAT mapping", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, external))
	state.Interface = strValue(getString(obj, "interface"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.NoBINAT = boolValue(getBool(obj, "nobinat"))
	state.NATReflection = strValue(getString(obj, "natreflection"))
	state.IPProtocol = strValue(getString(obj, "ipprotocol"))
	state.External = strValue(getString(obj, "external"))
	state.Source = strValue(getString(obj, "source"))
	state.Destination = strValue(getString(obj, "destination"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallNATOneToOneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallNATOneToOneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.Interface.ValueString()
	external := plan.External.ValueString()
	if iface == "" {
		iface, external = splitRouteKey(plan.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, iface, external)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update 1:1 NAT mapping", err.Error())
		} else {
			resp.Diagnostics.AddError("1:1 NAT mapping not found", "mapping "+iface+"|"+external+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "interface", plan.Interface)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nobinat", plan.NoBINAT)
	setString(payload, "natreflection", plan.NATReflection)
	setString(payload, "ipprotocol", plan.IPProtocol)
	setString(payload, "external", plan.External)
	setString(payload, "source", plan.Source)
	setString(payload, "destination", plan.Destination)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, firewallNATOneToOneSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update 1:1 NAT mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, external))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATOneToOneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallNATOneToOneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	external := state.External.ValueString()
	if iface == "" {
		iface, external = splitRouteKey(state.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, iface, external)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete 1:1 NAT mapping", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, firewallNATOneToOneSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete 1:1 NAT mapping", err.Error())
	}
}

func (r *firewallNATOneToOneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_outbound
// ---------------------------------------------------------------------------

type firewallNATOutboundResource struct{ client *client.Client }

var _ resource.Resource = (*firewallNATOutboundResource)(nil)
var _ resource.ResourceWithConfigure = (*firewallNATOutboundResource)(nil)
var _ resource.ResourceWithImportState = (*firewallNATOutboundResource)(nil)

const (
	firewallNATOutboundPlural   = "/api/v2/firewall/nat/outbound/mappings"
	firewallNATOutboundSingular = "/api/v2/firewall/nat/outbound/mapping"
)

type firewallNATOutboundModel struct {
	ID              types.String `tfsdk:"id"`
	Interface       types.String `tfsdk:"interface"`
	Protocol        types.String `tfsdk:"protocol"`
	Disabled        types.Bool   `tfsdk:"disabled"`
	NoNAT           types.Bool   `tfsdk:"nonat"`
	NoSync          types.Bool   `tfsdk:"nosync"`
	Source          types.String `tfsdk:"source"`
	SourcePort      types.String `tfsdk:"source_port"`
	Destination     types.String `tfsdk:"destination"`
	DestinationPort types.String `tfsdk:"destination_port"`
	Target          types.String `tfsdk:"target"`
	TargetSubnet    types.Int64  `tfsdk:"target_subnet"`
	NATPort         types.String `tfsdk:"nat_port"`
	StaticNATPort   types.Bool   `tfsdk:"static_nat_port"`
	PoolOpts        types.String `tfsdk:"poolopts"`
	SourceHashKey   types.String `tfsdk:"source_hash_key"`
	Descr           types.String `tfsdk:"descr"`
}

func NewFirewallNATOutboundResource() resource.Resource { return &firewallNATOutboundResource{} }

func (r *firewallNATOutboundResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_nat_outbound"
}
func (r *firewallNATOutboundResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *firewallNATOutboundResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a firewall NAT outbound mapping rule. Identified by the combination of `interface`, `protocol`, `source`, `destination` and `target`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"interface": schema.StringAttribute{
				Required:    true,
				Description: "The interface on which traffic is matched as it exits the firewall.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: "The protocol this rule should match.",
				Validators: []validator.String{
					stringvalidator.OneOf("tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "ipv6", "igmp", "pim", "ospf"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"disabled": optionalBoolAttribute("Disable this outbound NAT rule."),
			"nonat":    optionalBoolAttribute("Do not NAT traffic matching this rule."),
			"nosync":   optionalBoolAttribute("Do not sync this rule to HA peers."),
			"source": schema.StringAttribute{
				Required:    true,
				Description: "The source network this rule should match.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_port": optionalStringAttribute("The source port this rule should match."),
			"destination": schema.StringAttribute{
				Required:    true,
				Description: "The destination network this rule should match.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"destination_port": optionalStringAttribute("The destination port this rule should match."),
			"target": schema.StringAttribute{
				Required:    true,
				Description: "The target network traffic matching this rule should be translated to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_subnet":   optionalIntAttribute("The subnet bits for the assigned `target`."),
			"nat_port":        optionalStringAttribute("The external source port or port range used for rewriting the original source port."),
			"static_nat_port": optionalBoolAttribute("Do not rewrite the source port for traffic matching this rule."),
			"poolopts":        enumAttribute("Pool option for load balancing across multiple target addresses.", "round-robin", "round-robin sticky-address", "random", "random sticky-address", "source-hash", "bitmask"),
			"source_hash_key": optionalStringAttribute("The key fed to the hashing algorithm, as a 16 byte (32 character) hex string prefixed with `0x`."),
			"descr":           optionalStringAttribute("A description for the outbound NAT mapping."),
		},
	}
}

func (r *firewallNATOutboundResource) key(iface, protocol, source, destination, target string) string {
	return iface + "|" + protocol + "|" + source + "|" + destination + "|" + target
}

func (r *firewallNATOutboundResource) find(ctx context.Context, iface, protocol, source, destination, target string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, firewallNATOutboundPlural, map[string]string{
		"interface":   iface,
		"protocol":    protocol,
		"source":      source,
		"destination": destination,
		"target":      target,
	})
}

func (r *firewallNATOutboundResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallNATOutboundModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "interface", plan.Interface)
	setString(payload, "protocol", plan.Protocol)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nonat", plan.NoNAT)
	setBool(payload, "nosync", plan.NoSync)
	setString(payload, "source", plan.Source)
	setString(payload, "source_port", plan.SourcePort)
	setString(payload, "destination", plan.Destination)
	setString(payload, "destination_port", plan.DestinationPort)
	setString(payload, "target", plan.Target)
	setInt(payload, "target_subnet", plan.TargetSubnet)
	setString(payload, "nat_port", plan.NATPort)
	setBool(payload, "static_nat_port", plan.StaticNATPort)
	setString(payload, "poolopts", plan.PoolOpts)
	setString(payload, "source_hash_key", plan.SourceHashKey)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, firewallNATOutboundSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create outbound NAT mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan.Interface.ValueString(), plan.Protocol.ValueString(), plan.Source.ValueString(), plan.Destination.ValueString(), plan.Target.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATOutboundResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallNATOutboundModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	protocol := state.Protocol.ValueString()
	source := state.Source.ValueString()
	destination := state.Destination.ValueString()
	target := state.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(state.ID.ValueString(), 5)
		iface, protocol, source, destination, target = parts[0], parts[1], parts[2], parts[3], parts[4]
	}
	_, obj, found, err := r.find(ctx, iface, protocol, source, destination, target)
	if err != nil {
		resp.Diagnostics.AddError("failed to read outbound NAT mapping", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, protocol, source, destination, target))
	state.Interface = strValue(getString(obj, "interface"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.NoNAT = boolValue(getBool(obj, "nonat"))
	state.NoSync = boolValue(getBool(obj, "nosync"))
	state.Source = strValue(getString(obj, "source"))
	state.SourcePort = strValue(getString(obj, "source_port"))
	state.Destination = strValue(getString(obj, "destination"))
	state.DestinationPort = strValue(getString(obj, "destination_port"))
	state.Target = strValue(getString(obj, "target"))
	state.TargetSubnet = intValue(getInt(obj, "target_subnet"))
	state.NATPort = strValue(getString(obj, "nat_port"))
	state.StaticNATPort = boolValue(getBool(obj, "static_nat_port"))
	state.PoolOpts = strValue(getString(obj, "poolopts"))
	state.SourceHashKey = strValue(getString(obj, "source_hash_key"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallNATOutboundResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallNATOutboundModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.Interface.ValueString()
	protocol := plan.Protocol.ValueString()
	source := plan.Source.ValueString()
	destination := plan.Destination.ValueString()
	target := plan.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(plan.ID.ValueString(), 5)
		iface, protocol, source, destination, target = parts[0], parts[1], parts[2], parts[3], parts[4]
	}
	id, _, found, err := r.find(ctx, iface, protocol, source, destination, target)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update outbound NAT mapping", err.Error())
		} else {
			resp.Diagnostics.AddError("outbound NAT mapping not found", "mapping "+iface+"|"+protocol+"|"+source+"|"+destination+"|"+target+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "interface", plan.Interface)
	setString(payload, "protocol", plan.Protocol)
	setBool(payload, "disabled", plan.Disabled)
	setBool(payload, "nonat", plan.NoNAT)
	setBool(payload, "nosync", plan.NoSync)
	setString(payload, "source", plan.Source)
	setString(payload, "source_port", plan.SourcePort)
	setString(payload, "destination", plan.Destination)
	setString(payload, "destination_port", plan.DestinationPort)
	setString(payload, "target", plan.Target)
	setInt(payload, "target_subnet", plan.TargetSubnet)
	setString(payload, "nat_port", plan.NATPort)
	setBool(payload, "static_nat_port", plan.StaticNATPort)
	setString(payload, "poolopts", plan.PoolOpts)
	setString(payload, "source_hash_key", plan.SourceHashKey)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, firewallNATOutboundSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update outbound NAT mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, protocol, source, destination, target))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallNATOutboundResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallNATOutboundModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	protocol := state.Protocol.ValueString()
	source := state.Source.ValueString()
	destination := state.Destination.ValueString()
	target := state.Target.ValueString()
	if iface == "" {
		parts := splitKeyN(state.ID.ValueString(), 5)
		iface, protocol, source, destination, target = parts[0], parts[1], parts[2], parts[3], parts[4]
	}
	id, _, found, err := r.find(ctx, iface, protocol, source, destination, target)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete outbound NAT mapping", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, firewallNATOutboundSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete outbound NAT mapping", err.Error())
	}
}

func (r *firewallNATOutboundResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// splitKeyN splits a pipe-joined composite key into n parts, treating any
// extra separators as part of the final component (so values may contain
// '|'). Missing trailing parts are returned as empty strings.
func splitKeyN(id string, n int) []string {
	parts := strings.SplitN(id, "|", n)
	if len(parts) < n {
		out := make([]string, n)
		copy(out, parts)
		for i := len(parts); i < n; i++ {
			out[i] = ""
		}
		return out
	}
	return parts
}
