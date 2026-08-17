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

// ---------------------------------------------------------------------------
// pfsense_network_interface
// ---------------------------------------------------------------------------

type networkInterfaceResource struct{ client *client.Client }

var _ resource.Resource = (*networkInterfaceResource)(nil)
var _ resource.ResourceWithConfigure = (*networkInterfaceResource)(nil)
var _ resource.ResourceWithImportState = (*networkInterfaceResource)(nil)

const (
	networkInterfacePlural   = "/api/v2/interfaces"
	networkInterfaceSingular = "/api/v2/interface"
)

type networkInterfaceModel struct {
	ID                            types.String `tfsdk:"id"`
	If                            types.String `tfsdk:"if"`
	Enable                        types.Bool   `tfsdk:"enable"`
	Descr                         types.String `tfsdk:"descr"`
	Spoofmac                      types.String `tfsdk:"spoofmac"`
	MTU                           types.Int64  `tfsdk:"mtu"`
	Mss                           types.Int64  `tfsdk:"mss"`
	Media                         types.String `tfsdk:"media"`
	Mediaopt                      types.String `tfsdk:"mediaopt"`
	Blockpriv                     types.Bool   `tfsdk:"blockpriv"`
	Blockbogons                   types.Bool   `tfsdk:"blockbogons"`
	Typev4                        types.String `tfsdk:"typev4"`
	Ipaddr                        types.String `tfsdk:"ipaddr"`
	Subnet                        types.Int64  `tfsdk:"subnet"`
	Gateway                       types.String `tfsdk:"gateway"`
	Dhcphostname                  types.String `tfsdk:"dhcphostname"`
	AliasAddress                  types.String `tfsdk:"alias_address"`
	AliasSubnet                   types.Int64  `tfsdk:"alias_subnet"`
	Dhcprejectfrom                types.List   `tfsdk:"dhcprejectfrom"`
	AdvDHCPConfigAdvanced         types.Bool   `tfsdk:"adv_dhcp_config_advanced"`
	AdvDHCPPtValues               types.String `tfsdk:"adv_dhcp_pt_values"`
	AdvDHCPPtTimeout              types.Int64  `tfsdk:"adv_dhcp_pt_timeout"`
	AdvDHCPPtRetry                types.Int64  `tfsdk:"adv_dhcp_pt_retry"`
	AdvDHCPPtSelectTimeout        types.Int64  `tfsdk:"adv_dhcp_pt_select_timeout"`
	AdvDHCPPtReboot               types.Int64  `tfsdk:"adv_dhcp_pt_reboot"`
	AdvDHCPPtBackoffCutoff        types.Int64  `tfsdk:"adv_dhcp_pt_backoff_cutoff"`
	AdvDHCPPtInitialInterval      types.Int64  `tfsdk:"adv_dhcp_pt_initial_interval"`
	AdvDHCPSendOptions            types.String `tfsdk:"adv_dhcp_send_options"`
	AdvDHCPRequestOptions         types.String `tfsdk:"adv_dhcp_request_options"`
	AdvDHCPRequiredOptions        types.String `tfsdk:"adv_dhcp_required_options"`
	AdvDHCPOptionModifiers        types.String `tfsdk:"adv_dhcp_option_modifiers"`
	AdvDHCPConfigFileOverride     types.Bool   `tfsdk:"adv_dhcp_config_file_override"`
	AdvDHCPConfigFileOverridePath types.String `tfsdk:"adv_dhcp_config_file_override_path"`
	Typev6                        types.String `tfsdk:"typev6"`
	Ipaddrv6                      types.String `tfsdk:"ipaddrv6"`
	Subnetv6                      types.Int64  `tfsdk:"subnetv6"`
	Gatewayv6                     types.String `tfsdk:"gatewayv6"`
	Ipv6usev4iface                types.Bool   `tfsdk:"ipv6usev4iface"`
	Slaacusev4iface               types.Bool   `tfsdk:"slaacusev4iface"`
	Prefix6rd                     types.String `tfsdk:"prefix_6rd"`
	Gateway6rd                    types.String `tfsdk:"gateway_6rd"`
	Prefix6rdV4plen               types.Int64  `tfsdk:"prefix_6rd_v4plen"`
	Track6Interface               types.String `tfsdk:"track6_interface"`
	Track6PrefixIDHex             types.String `tfsdk:"track6_prefix_id_hex"`
}

func NewNetworkInterfaceResource() resource.Resource { return &networkInterfaceResource{} }

func (r *networkInterfaceResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_network_interface"
}
func (r *networkInterfaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *networkInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a pfSense network interface (e.g. WAN, LAN, OPT). Identified by the interface key (`if`).",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"if": schema.StringAttribute{
				Required:    true,
				Description: "The interface key (e.g. `wan`, `lan`, `opt1`). Unique; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enable":      optionalBoolAttribute("Whether the interface is enabled."),
			"descr":       requiredStringAttribute("Description (name) of the interface."),
			"spoofmac":    optionalStringAttribute("MAC address to spoof on this interface."),
			"mtu":         optionalIntAttribute("MTU for the interface."),
			"mss":         optionalIntAttribute("MSS for the interface."),
			"media":       optionalStringAttribute("Media type (e.g. `1000baseT`)."),
			"mediaopt":    optionalStringAttribute("Media option (e.g. `full-duplex`)."),
			"blockpriv":   optionalBoolAttribute("Block RFC1918 private networks on this interface."),
			"blockbogons": optionalBoolAttribute("Block bogon networks on this interface."),
			"typev4": schema.StringAttribute{
				Required:    true,
				Description: "IPv4 address configuration type: `static`, `dhcp` or `none`.",
				Validators: []validator.String{
					stringvalidator.OneOf("static", "dhcp", "none"),
				},
			},
			"ipaddr":                             requiredStringAttribute("IPv4 address (used when `typev4` is `static`)."),
			"subnet":                             schema.Int64Attribute{Required: true, Description: "IPv4 subnet bit count (used when `typev4` is `static`)."},
			"gateway":                            optionalStringAttribute("IPv4 gateway (gateway or gateway group name)."),
			"dhcphostname":                       optionalStringAttribute("Hostname to send to the DHCP server."),
			"alias_address":                      optionalStringAttribute("IPv4 alias address for the interface."),
			"alias_subnet":                       optionalIntAttribute("IPv4 alias subnet bit count."),
			"dhcprejectfrom":                     schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "DHCP server addresses to reject."},
			"adv_dhcp_config_advanced":           optionalBoolAttribute("Enable advanced DHCP client configuration."),
			"adv_dhcp_pt_values":                 enumAttribute("Which advanced DHCP values to use (currently only `SavedCfg`).", "SavedCfg"),
			"adv_dhcp_pt_timeout":                optionalIntAttribute("DHCP config timeout (seconds)."),
			"adv_dhcp_pt_retry":                  optionalIntAttribute("DHCP config retry count."),
			"adv_dhcp_pt_select_timeout":         optionalIntAttribute("DHCP select timeout (seconds)."),
			"adv_dhcp_pt_reboot":                 optionalIntAttribute("DHCP reboot timeout (seconds)."),
			"adv_dhcp_pt_backoff_cutoff":         optionalIntAttribute("DHCP backoff cutoff (seconds)."),
			"adv_dhcp_pt_initial_interval":       optionalIntAttribute("DHCP initial interval (seconds)."),
			"adv_dhcp_send_options":              optionalStringAttribute("DHCP options to send (comma-separated)."),
			"adv_dhcp_request_options":           optionalStringAttribute("DHCP options to request (comma-separated)."),
			"adv_dhcp_required_options":          optionalStringAttribute("DHCP options to require (comma-separated)."),
			"adv_dhcp_option_modifiers":          optionalStringAttribute("DHCP option modifiers (comma-separated)."),
			"adv_dhcp_config_file_override":      optionalBoolAttribute("Override the DHCP client config file."),
			"adv_dhcp_config_file_override_path": optionalStringAttribute("Path to the DHCP client config file override."),
			"typev6": schema.StringAttribute{
				Optional:    true,
				Description: "IPv6 address configuration type: `staticv6`, `dhcp6`, `slaac`, `6rd`, `track6`, `6to4` or `none`.",
				Validators: []validator.String{
					stringvalidator.OneOf("staticv6", "dhcp6", "slaac", "6rd", "track6", "6to4", "none"),
				},
			},
			"ipaddrv6":             requiredStringAttribute("IPv6 address (used when `typev6` is `staticv6`)."),
			"subnetv6":             schema.Int64Attribute{Required: true, Description: "IPv6 subnet bit count (used when `typev6` is `staticv6`)."},
			"gatewayv6":            optionalStringAttribute("Optional IPv6 gateway (gateway name)."),
			"ipv6usev4iface":       optionalBoolAttribute("Use IPv4 gateway as the IPv6 gateway."),
			"slaacusev4iface":      optionalBoolAttribute("Use IPv4 gateway as the IPv6 SLAAC gateway."),
			"prefix_6rd":           requiredStringAttribute("6rd prefix (used when `typev6` is `6rd`)."),
			"gateway_6rd":          requiredStringAttribute("6rd gateway address (used when `typev6` is `6rd`)."),
			"prefix_6rd_v4plen":    schema.Int64Attribute{Required: true, Description: "6rd IPv4 prefix length (used when `typev6` is `6rd`)."},
			"track6_interface":     requiredStringAttribute("Interface to track for IPv6 (used when `typev6` is `track6`)."),
			"track6_prefix_id_hex": optionalStringAttribute("IPv6 prefix ID in hex (used when `typev6` is `track6`)."),
		},
	}
}

func (r *networkInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan networkInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "if", plan.If)
	setBool(payload, "enable", plan.Enable)
	setString(payload, "descr", plan.Descr)
	setString(payload, "spoofmac", plan.Spoofmac)
	setInt(payload, "mtu", plan.MTU)
	setInt(payload, "mss", plan.Mss)
	setString(payload, "media", plan.Media)
	setString(payload, "mediaopt", plan.Mediaopt)
	setBool(payload, "blockpriv", plan.Blockpriv)
	setBool(payload, "blockbogons", plan.Blockbogons)
	setString(payload, "typev4", plan.Typev4)
	setString(payload, "ipaddr", plan.Ipaddr)
	setInt(payload, "subnet", plan.Subnet)
	setString(payload, "gateway", plan.Gateway)
	setString(payload, "dhcphostname", plan.Dhcphostname)
	setString(payload, "alias_address", plan.AliasAddress)
	setInt(payload, "alias_subnet", plan.AliasSubnet)
	setStringList(payload, "dhcprejectfrom", plan.Dhcprejectfrom)
	setBool(payload, "adv_dhcp_config_advanced", plan.AdvDHCPConfigAdvanced)
	setString(payload, "adv_dhcp_pt_values", plan.AdvDHCPPtValues)
	setInt(payload, "adv_dhcp_pt_timeout", plan.AdvDHCPPtTimeout)
	setInt(payload, "adv_dhcp_pt_retry", plan.AdvDHCPPtRetry)
	setInt(payload, "adv_dhcp_pt_select_timeout", plan.AdvDHCPPtSelectTimeout)
	setInt(payload, "adv_dhcp_pt_reboot", plan.AdvDHCPPtReboot)
	setInt(payload, "adv_dhcp_pt_backoff_cutoff", plan.AdvDHCPPtBackoffCutoff)
	setInt(payload, "adv_dhcp_pt_initial_interval", plan.AdvDHCPPtInitialInterval)
	setString(payload, "adv_dhcp_send_options", plan.AdvDHCPSendOptions)
	setString(payload, "adv_dhcp_request_options", plan.AdvDHCPRequestOptions)
	setString(payload, "adv_dhcp_required_options", plan.AdvDHCPRequiredOptions)
	setString(payload, "adv_dhcp_option_modifiers", plan.AdvDHCPOptionModifiers)
	setBool(payload, "adv_dhcp_config_file_override", plan.AdvDHCPConfigFileOverride)
	setString(payload, "adv_dhcp_config_file_override_path", plan.AdvDHCPConfigFileOverridePath)
	setString(payload, "typev6", plan.Typev6)
	setString(payload, "ipaddrv6", plan.Ipaddrv6)
	setInt(payload, "subnetv6", plan.Subnetv6)
	setString(payload, "gatewayv6", plan.Gatewayv6)
	setBool(payload, "ipv6usev4iface", plan.Ipv6usev4iface)
	setBool(payload, "slaacusev4iface", plan.Slaacusev4iface)
	setString(payload, "prefix_6rd", plan.Prefix6rd)
	setString(payload, "gateway_6rd", plan.Gateway6rd)
	setInt(payload, "prefix_6rd_v4plen", plan.Prefix6rdV4plen)
	setString(payload, "track6_interface", plan.Track6Interface)
	setString(payload, "track6_prefix_id_hex", plan.Track6PrefixIDHex)
	applyNow(payload)
	if _, err := r.client.Create(ctx, networkInterfaceSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create network interface", err.Error())
		return
	}
	plan.ID = plan.If
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state networkInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, networkInterfacePlural, "if", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to read network interface", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(iface)
	state.If = types.StringValue(iface)
	state.Enable = boolValue(getBool(obj, "enable"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Spoofmac = strValue(getString(obj, "spoofmac"))
	state.MTU = intValue(getInt(obj, "mtu"))
	state.Mss = intValue(getInt(obj, "mss"))
	state.Media = strValue(getString(obj, "media"))
	state.Mediaopt = strValue(getString(obj, "mediaopt"))
	state.Blockpriv = boolValue(getBool(obj, "blockpriv"))
	state.Blockbogons = boolValue(getBool(obj, "blockbogons"))
	state.Typev4 = strValue(getString(obj, "typev4"))
	state.Ipaddr = strValue(getString(obj, "ipaddr"))
	state.Subnet = intValue(getInt(obj, "subnet"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.Dhcphostname = strValue(getString(obj, "dhcphostname"))
	state.AliasAddress = strValue(getString(obj, "alias_address"))
	state.AliasSubnet = intValue(getInt(obj, "alias_subnet"))
	state.Dhcprejectfrom = strListValue(ctx, getStringSlice(obj, "dhcprejectfrom"))
	state.AdvDHCPConfigAdvanced = boolValue(getBool(obj, "adv_dhcp_config_advanced"))
	state.AdvDHCPPtValues = strValue(getString(obj, "adv_dhcp_pt_values"))
	state.AdvDHCPPtTimeout = intValue(getInt(obj, "adv_dhcp_pt_timeout"))
	state.AdvDHCPPtRetry = intValue(getInt(obj, "adv_dhcp_pt_retry"))
	state.AdvDHCPPtSelectTimeout = intValue(getInt(obj, "adv_dhcp_pt_select_timeout"))
	state.AdvDHCPPtReboot = intValue(getInt(obj, "adv_dhcp_pt_reboot"))
	state.AdvDHCPPtBackoffCutoff = intValue(getInt(obj, "adv_dhcp_pt_backoff_cutoff"))
	state.AdvDHCPPtInitialInterval = intValue(getInt(obj, "adv_dhcp_pt_initial_interval"))
	state.AdvDHCPSendOptions = strValue(getString(obj, "adv_dhcp_send_options"))
	state.AdvDHCPRequestOptions = strValue(getString(obj, "adv_dhcp_request_options"))
	state.AdvDHCPRequiredOptions = strValue(getString(obj, "adv_dhcp_required_options"))
	state.AdvDHCPOptionModifiers = strValue(getString(obj, "adv_dhcp_option_modifiers"))
	state.AdvDHCPConfigFileOverride = boolValue(getBool(obj, "adv_dhcp_config_file_override"))
	state.AdvDHCPConfigFileOverridePath = strValue(getString(obj, "adv_dhcp_config_file_override_path"))
	state.Typev6 = strValue(getString(obj, "typev6"))
	state.Ipaddrv6 = strValue(getString(obj, "ipaddrv6"))
	state.Subnetv6 = intValue(getInt(obj, "subnetv6"))
	state.Gatewayv6 = strValue(getString(obj, "gatewayv6"))
	state.Ipv6usev4iface = boolValue(getBool(obj, "ipv6usev4iface"))
	state.Slaacusev4iface = boolValue(getBool(obj, "slaacusev4iface"))
	state.Prefix6rd = strValue(getString(obj, "prefix_6rd"))
	state.Gateway6rd = strValue(getString(obj, "gateway_6rd"))
	state.Prefix6rdV4plen = intValue(getInt(obj, "prefix_6rd_v4plen"))
	state.Track6Interface = strValue(getString(obj, "track6_interface"))
	state.Track6PrefixIDHex = strValue(getString(obj, "track6_prefix_id_hex"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *networkInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan networkInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.If.ValueString()
	if iface == "" {
		iface = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, networkInterfacePlural, "if", iface)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update network interface", err.Error())
		} else {
			resp.Diagnostics.AddError("network interface not found", iface+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "if", plan.If)
	setBool(payload, "enable", plan.Enable)
	setString(payload, "descr", plan.Descr)
	setString(payload, "spoofmac", plan.Spoofmac)
	setInt(payload, "mtu", plan.MTU)
	setInt(payload, "mss", plan.Mss)
	setString(payload, "media", plan.Media)
	setString(payload, "mediaopt", plan.Mediaopt)
	setBool(payload, "blockpriv", plan.Blockpriv)
	setBool(payload, "blockbogons", plan.Blockbogons)
	setString(payload, "typev4", plan.Typev4)
	setString(payload, "ipaddr", plan.Ipaddr)
	setInt(payload, "subnet", plan.Subnet)
	setString(payload, "gateway", plan.Gateway)
	setString(payload, "dhcphostname", plan.Dhcphostname)
	setString(payload, "alias_address", plan.AliasAddress)
	setInt(payload, "alias_subnet", plan.AliasSubnet)
	setStringList(payload, "dhcprejectfrom", plan.Dhcprejectfrom)
	setBool(payload, "adv_dhcp_config_advanced", plan.AdvDHCPConfigAdvanced)
	setString(payload, "adv_dhcp_pt_values", plan.AdvDHCPPtValues)
	setInt(payload, "adv_dhcp_pt_timeout", plan.AdvDHCPPtTimeout)
	setInt(payload, "adv_dhcp_pt_retry", plan.AdvDHCPPtRetry)
	setInt(payload, "adv_dhcp_pt_select_timeout", plan.AdvDHCPPtSelectTimeout)
	setInt(payload, "adv_dhcp_pt_reboot", plan.AdvDHCPPtReboot)
	setInt(payload, "adv_dhcp_pt_backoff_cutoff", plan.AdvDHCPPtBackoffCutoff)
	setInt(payload, "adv_dhcp_pt_initial_interval", plan.AdvDHCPPtInitialInterval)
	setString(payload, "adv_dhcp_send_options", plan.AdvDHCPSendOptions)
	setString(payload, "adv_dhcp_request_options", plan.AdvDHCPRequestOptions)
	setString(payload, "adv_dhcp_required_options", plan.AdvDHCPRequiredOptions)
	setString(payload, "adv_dhcp_option_modifiers", plan.AdvDHCPOptionModifiers)
	setBool(payload, "adv_dhcp_config_file_override", plan.AdvDHCPConfigFileOverride)
	setString(payload, "adv_dhcp_config_file_override_path", plan.AdvDHCPConfigFileOverridePath)
	setString(payload, "typev6", plan.Typev6)
	setString(payload, "ipaddrv6", plan.Ipaddrv6)
	setInt(payload, "subnetv6", plan.Subnetv6)
	setString(payload, "gatewayv6", plan.Gatewayv6)
	setBool(payload, "ipv6usev4iface", plan.Ipv6usev4iface)
	setBool(payload, "slaacusev4iface", plan.Slaacusev4iface)
	setString(payload, "prefix_6rd", plan.Prefix6rd)
	setString(payload, "gateway_6rd", plan.Gateway6rd)
	setInt(payload, "prefix_6rd_v4plen", plan.Prefix6rdV4plen)
	setString(payload, "track6_interface", plan.Track6Interface)
	setString(payload, "track6_prefix_id_hex", plan.Track6PrefixIDHex)
	applyNow(payload)
	if _, err := r.client.Update(ctx, networkInterfaceSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update network interface", err.Error())
		return
	}
	plan.ID = types.StringValue(iface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *networkInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state networkInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, networkInterfacePlural, "if", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete network interface", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, networkInterfaceSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete network interface", err.Error())
	}
}

func (r *networkInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_interface_bridge
// ---------------------------------------------------------------------------

type interfaceBridgeResource struct{ client *client.Client }

var _ resource.Resource = (*interfaceBridgeResource)(nil)
var _ resource.ResourceWithConfigure = (*interfaceBridgeResource)(nil)
var _ resource.ResourceWithImportState = (*interfaceBridgeResource)(nil)

const (
	interfaceBridgePlural   = "/api/v2/interface/bridges"
	interfaceBridgeSingular = "/api/v2/interface/bridge"
)

type interfaceBridgeModel struct {
	ID       types.String `tfsdk:"id"`
	Members  types.List   `tfsdk:"members"`
	Descr    types.String `tfsdk:"descr"`
	Bridgeif types.String `tfsdk:"bridgeif"`
}

func NewInterfaceBridgeResource() resource.Resource { return &interfaceBridgeResource{} }

func (r *interfaceBridgeResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_interface_bridge"
}
func (r *interfaceBridgeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *interfaceBridgeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a bridge interface. Identified by its description.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"members": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Physical interfaces that are members of the bridge.",
			},
			"descr": schema.StringAttribute{
				Required:    true,
				Description: "Description (name) of the bridge. Unique among bridges; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bridgeif": schema.StringAttribute{Computed: true, Description: "The generated bridge interface name (e.g. `bridge0`)."},
		},
	}
}

func (r *interfaceBridgeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceBridgeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	applyNow(payload)
	raw, err := r.client.Create(ctx, interfaceBridgeSingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to create interface bridge", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create interface bridge", err.Error())
		return
	}
	plan.Bridgeif = strValue(getString(obj, "bridgeif"))
	plan.ID = plan.Descr
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceBridgeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceBridgeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, interfaceBridgePlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read interface bridge", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.Descr = types.StringValue(descr)
	state.Members = strListValue(ctx, getStringSlice(obj, "members"))
	state.Bridgeif = strValue(getString(obj, "bridgeif"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interfaceBridgeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceBridgeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceBridgePlural, "descr", descr)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update interface bridge", err.Error())
		} else {
			resp.Diagnostics.AddError("interface bridge not found", descr+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	applyNow(payload)
	raw, err := r.client.Update(ctx, interfaceBridgeSingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to update interface bridge", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update interface bridge", err.Error())
		return
	}
	plan.Bridgeif = strValue(getString(obj, "bridgeif"))
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceBridgeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interfaceBridgeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceBridgePlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete interface bridge", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, interfaceBridgeSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete interface bridge", err.Error())
	}
}

func (r *interfaceBridgeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_interface_gre
// ---------------------------------------------------------------------------

type interfaceGREResource struct{ client *client.Client }

var _ resource.Resource = (*interfaceGREResource)(nil)
var _ resource.ResourceWithConfigure = (*interfaceGREResource)(nil)
var _ resource.ResourceWithImportState = (*interfaceGREResource)(nil)

const (
	interfaceGREPlural   = "/api/v2/interface/gres"
	interfaceGRESingular = "/api/v2/interface/gre"
)

type interfaceGREModel struct {
	ID                types.String `tfsdk:"id"`
	If                types.String `tfsdk:"if"`
	Greif             types.String `tfsdk:"greif"`
	Descr             types.String `tfsdk:"descr"`
	AddStaticRoute    types.Bool   `tfsdk:"add_static_route"`
	RemoteAddr        types.String `tfsdk:"remote_addr"`
	TunnelLocalAddr   types.String `tfsdk:"tunnel_local_addr"`
	TunnelRemoteAddr  types.String `tfsdk:"tunnel_remote_addr"`
	TunnelRemoteNet   types.Int64  `tfsdk:"tunnel_remote_net"`
	TunnelLocalAddr6  types.String `tfsdk:"tunnel_local_addr6"`
	TunnelRemoteAddr6 types.String `tfsdk:"tunnel_remote_addr6"`
	TunnelRemoteNet6  types.Int64  `tfsdk:"tunnel_remote_net6"`
}

func NewInterfaceGREResource() resource.Resource { return &interfaceGREResource{} }

func (r *interfaceGREResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_interface_gre"
}
func (r *interfaceGREResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *interfaceGREResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a GRE tunnel interface. Identified by its interface key (`if`).",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"if": schema.StringAttribute{
				Required:    true,
				Description: "The interface key to assign the GRE tunnel to. Unique; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"greif":               schema.StringAttribute{Computed: true, Description: "The generated GRE interface name (e.g. `gre0`)."},
			"descr":               optionalStringAttribute("Description (name) of the GRE tunnel."),
			"add_static_route":    optionalBoolAttribute("Add a static route to the local GRE tunnel address."),
			"remote_addr":         requiredStringAttribute("Remote address of the GRE tunnel."),
			"tunnel_local_addr":   optionalStringAttribute("Local IPv4 address for the tunnel."),
			"tunnel_remote_addr":  requiredStringAttribute("Remote IPv4 address for the tunnel."),
			"tunnel_remote_net":   optionalIntAttribute("Remote IPv4 subnet bit count for the tunnel."),
			"tunnel_local_addr6":  optionalStringAttribute("Local IPv6 address for the tunnel."),
			"tunnel_remote_addr6": requiredStringAttribute("Remote IPv6 address for the tunnel."),
			"tunnel_remote_net6":  optionalIntAttribute("Remote IPv6 subnet bit count for the tunnel."),
		},
	}
}

func (r *interfaceGREResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceGREModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "if", plan.If)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "add_static_route", plan.AddStaticRoute)
	setString(payload, "remote_addr", plan.RemoteAddr)
	setString(payload, "tunnel_local_addr", plan.TunnelLocalAddr)
	setString(payload, "tunnel_remote_addr", plan.TunnelRemoteAddr)
	setInt(payload, "tunnel_remote_net", plan.TunnelRemoteNet)
	setString(payload, "tunnel_local_addr6", plan.TunnelLocalAddr6)
	setString(payload, "tunnel_remote_addr6", plan.TunnelRemoteAddr6)
	setInt(payload, "tunnel_remote_net6", plan.TunnelRemoteNet6)
	applyNow(payload)
	raw, err := r.client.Create(ctx, interfaceGRESingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to create GRE interface", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create GRE interface", err.Error())
		return
	}
	plan.Greif = strValue(getString(obj, "greif"))
	plan.ID = plan.If
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceGREResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceGREModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, interfaceGREPlural, "if", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to read GRE interface", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(iface)
	state.If = types.StringValue(iface)
	state.Greif = strValue(getString(obj, "greif"))
	state.Descr = strValue(getString(obj, "descr"))
	state.AddStaticRoute = boolValue(getBool(obj, "add_static_route"))
	state.RemoteAddr = strValue(getString(obj, "remote_addr"))
	state.TunnelLocalAddr = strValue(getString(obj, "tunnel_local_addr"))
	state.TunnelRemoteAddr = strValue(getString(obj, "tunnel_remote_addr"))
	state.TunnelRemoteNet = intValue(getInt(obj, "tunnel_remote_net"))
	state.TunnelLocalAddr6 = strValue(getString(obj, "tunnel_local_addr6"))
	state.TunnelRemoteAddr6 = strValue(getString(obj, "tunnel_remote_addr6"))
	state.TunnelRemoteNet6 = intValue(getInt(obj, "tunnel_remote_net6"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interfaceGREResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceGREModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.If.ValueString()
	if iface == "" {
		iface = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceGREPlural, "if", iface)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update GRE interface", err.Error())
		} else {
			resp.Diagnostics.AddError("GRE interface not found", iface+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "if", plan.If)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "add_static_route", plan.AddStaticRoute)
	setString(payload, "remote_addr", plan.RemoteAddr)
	setString(payload, "tunnel_local_addr", plan.TunnelLocalAddr)
	setString(payload, "tunnel_remote_addr", plan.TunnelRemoteAddr)
	setInt(payload, "tunnel_remote_net", plan.TunnelRemoteNet)
	setString(payload, "tunnel_local_addr6", plan.TunnelLocalAddr6)
	setString(payload, "tunnel_remote_addr6", plan.TunnelRemoteAddr6)
	setInt(payload, "tunnel_remote_net6", plan.TunnelRemoteNet6)
	applyNow(payload)
	raw, err := r.client.Update(ctx, interfaceGRESingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to update GRE interface", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update GRE interface", err.Error())
		return
	}
	plan.Greif = strValue(getString(obj, "greif"))
	plan.ID = types.StringValue(iface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceGREResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interfaceGREModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.If.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceGREPlural, "if", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete GRE interface", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, interfaceGRESingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete GRE interface", err.Error())
	}
}

func (r *interfaceGREResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_interface_lagg
// ---------------------------------------------------------------------------

type interfaceLAGGResource struct{ client *client.Client }

var _ resource.Resource = (*interfaceLAGGResource)(nil)
var _ resource.ResourceWithConfigure = (*interfaceLAGGResource)(nil)
var _ resource.ResourceWithImportState = (*interfaceLAGGResource)(nil)

const (
	interfaceLAGGPlural   = "/api/v2/interface/laggs"
	interfaceLAGGSingular = "/api/v2/interface/lagg"
)

type interfaceLAGGModel struct {
	ID             types.String `tfsdk:"id"`
	Laggif         types.String `tfsdk:"laggif"`
	Descr          types.String `tfsdk:"descr"`
	Members        types.List   `tfsdk:"members"`
	Proto          types.String `tfsdk:"proto"`
	Lacptimeout    types.String `tfsdk:"lacptimeout"`
	Lagghash       types.String `tfsdk:"lagghash"`
	Failovermaster types.String `tfsdk:"failovermaster"`
}

func NewInterfaceLAGGResource() resource.Resource { return &interfaceLAGGResource{} }

func (r *interfaceLAGGResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_interface_lagg"
}
func (r *interfaceLAGGResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *interfaceLAGGResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a link aggregation (LAGG) interface. Identified by its description.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"members": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Physical interfaces that are members of the LAGG.",
			},
			"descr": schema.StringAttribute{
				Required:    true,
				Description: "Description (name) of the LAGG. Unique among laggs; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"laggif": schema.StringAttribute{Computed: true, Description: "The generated LAGG interface name (e.g. `lagg0`)."},
			"proto": schema.StringAttribute{
				Required:    true,
				Description: "LAGG protocol: `lacp`, `failover`, `loadbalance`, `roundrobin` or `none`.",
				Validators: []validator.String{
					stringvalidator.OneOf("lacp", "failover", "loadbalance", "roundrobin", "none"),
				},
			},
			"lacptimeout":    enumAttribute("LACP timeout: `slow` or `fast`.", "slow", "fast"),
			"lagghash":       enumAttribute("LAGG hash algorithm: `l2`, `l3`, `l4`, `l2,l3`, `l2,l4`, `l3,l4` or `l2,l3,l4`.", "l2", "l3", "l4", "l2,l3", "l2,l4", "l3,l4", "l2,l3,l4"),
			"failovermaster": optionalStringAttribute("Failover master member interface."),
		},
	}
}

func (r *interfaceLAGGResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceLAGGModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	setString(payload, "proto", plan.Proto)
	setString(payload, "lacptimeout", plan.Lacptimeout)
	setString(payload, "lagghash", plan.Lagghash)
	setString(payload, "failovermaster", plan.Failovermaster)
	applyNow(payload)
	raw, err := r.client.Create(ctx, interfaceLAGGSingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to create LAGG interface", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create LAGG interface", err.Error())
		return
	}
	plan.Laggif = strValue(getString(obj, "laggif"))
	plan.ID = plan.Descr
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceLAGGResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceLAGGModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, interfaceLAGGPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read LAGG interface", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.Descr = types.StringValue(descr)
	state.Members = strListValue(ctx, getStringSlice(obj, "members"))
	state.Proto = strValue(getString(obj, "proto"))
	state.Lacptimeout = strValue(getString(obj, "lacptimeout"))
	state.Lagghash = strValue(getString(obj, "lagghash"))
	state.Failovermaster = strValue(getString(obj, "failovermaster"))
	state.Laggif = strValue(getString(obj, "laggif"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interfaceLAGGResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceLAGGModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceLAGGPlural, "descr", descr)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update LAGG interface", err.Error())
		} else {
			resp.Diagnostics.AddError("LAGG interface not found", descr+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	setString(payload, "proto", plan.Proto)
	setString(payload, "lacptimeout", plan.Lacptimeout)
	setString(payload, "lagghash", plan.Lagghash)
	setString(payload, "failovermaster", plan.Failovermaster)
	applyNow(payload)
	raw, err := r.client.Update(ctx, interfaceLAGGSingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to update LAGG interface", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update LAGG interface", err.Error())
		return
	}
	plan.Laggif = strValue(getString(obj, "laggif"))
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceLAGGResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interfaceLAGGModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceLAGGPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete LAGG interface", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, interfaceLAGGSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete LAGG interface", err.Error())
	}
}

func (r *interfaceLAGGResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_interface_group
// ---------------------------------------------------------------------------

type interfaceGroupResource struct{ client *client.Client }

var _ resource.Resource = (*interfaceGroupResource)(nil)
var _ resource.ResourceWithConfigure = (*interfaceGroupResource)(nil)
var _ resource.ResourceWithImportState = (*interfaceGroupResource)(nil)

const (
	interfaceGroupPlural   = "/api/v2/interface/groups"
	interfaceGroupSingular = "/api/v2/interface/group"
)

type interfaceGroupModel struct {
	ID      types.String `tfsdk:"id"`
	Ifname  types.String `tfsdk:"ifname"`
	Members types.List   `tfsdk:"members"`
	Descr   types.String `tfsdk:"descr"`
}

func NewInterfaceGroupResource() resource.Resource { return &interfaceGroupResource{} }

func (r *interfaceGroupResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_interface_group"
}
func (r *interfaceGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *interfaceGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an interface group. Identified by its interface name (`ifname`).",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"ifname": schema.StringAttribute{
				Required:    true,
				Description: "Name of the interface group. Unique; immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Interfaces that are members of the group.",
			},
			"descr": optionalStringAttribute("Description of the interface group."),
		},
	}
}

func (r *interfaceGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan interfaceGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "ifname", plan.Ifname)
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	applyNow(payload)
	if _, err := r.client.Create(ctx, interfaceGroupSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create interface group", err.Error())
		return
	}
	plan.ID = plan.Ifname
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state interfaceGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	ifname := state.Ifname.ValueString()
	if ifname == "" {
		ifname = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, interfaceGroupPlural, "ifname", ifname)
	if err != nil {
		resp.Diagnostics.AddError("failed to read interface group", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(ifname)
	state.Ifname = types.StringValue(ifname)
	state.Members = strListValue(ctx, getStringSlice(obj, "members"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *interfaceGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan interfaceGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	ifname := plan.Ifname.ValueString()
	if ifname == "" {
		ifname = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceGroupPlural, "ifname", ifname)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update interface group", err.Error())
		} else {
			resp.Diagnostics.AddError("interface group not found", ifname+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "ifname", plan.Ifname)
	setStringList(payload, "members", plan.Members)
	setString(payload, "descr", plan.Descr)
	applyNow(payload)
	if _, err := r.client.Update(ctx, interfaceGroupSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update interface group", err.Error())
		return
	}
	plan.ID = types.StringValue(ifname)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *interfaceGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state interfaceGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	ifname := state.Ifname.ValueString()
	if ifname == "" {
		ifname = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, interfaceGroupPlural, "ifname", ifname)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete interface group", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, interfaceGroupSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete interface group", err.Error())
	}
}

func (r *interfaceGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
