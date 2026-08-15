package provider

import (
	"context"

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

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_static_mapping
// Parent: DHCP server (natural key = interface). Child key: mac.
// Resource ID = <interface>|<mac>.
// ---------------------------------------------------------------------------

type dhcpStaticMappingResource struct{ client *client.Client }

var _ resource.Resource = (*dhcpStaticMappingResource)(nil)
var _ resource.ResourceWithConfigure = (*dhcpStaticMappingResource)(nil)
var _ resource.ResourceWithImportState = (*dhcpStaticMappingResource)(nil)

const (
	dhcpStaticMappingPlural   = "/api/v2/services/dhcp_server/static_mappings"
	dhcpStaticMappingSingular = "/api/v2/services/dhcp_server/static_mapping"
	dhcpParentPlural          = "/api/v2/services/dhcp_servers"
)

type dhcpStaticMappingModel struct {
	ID                  types.String `tfsdk:"id"`
	ParentID            types.String `tfsdk:"parent_id"`
	MAC                 types.String `tfsdk:"mac"`
	Ipaddr              types.String `tfsdk:"ipaddr"`
	CID                 types.String `tfsdk:"cid"`
	Hostname            types.String `tfsdk:"hostname"`
	Domain              types.String `tfsdk:"domain"`
	Domainsearchlist    types.List   `tfsdk:"domainsearchlist"`
	DefaultLeaseTime    types.Int64  `tfsdk:"defaultleasetime"`
	MaxLeaseTime        types.Int64  `tfsdk:"maxleasetime"`
	Gateway             types.String `tfsdk:"gateway"`
	DNSServer           types.List   `tfsdk:"dnsserver"`
	WINSServer          types.List   `tfsdk:"winsserver"`
	NTPServer           types.List   `tfsdk:"ntpserver"`
	ARPTableStaticEntry types.Bool   `tfsdk:"arp_table_static_entry"`
	Descr               types.String `tfsdk:"descr"`
}

func NewDHCPServerStaticMappingResource() resource.Resource { return &dhcpStaticMappingResource{} }

func (r *dhcpStaticMappingResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dhcp_static_mapping"
}
func (r *dhcpStaticMappingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dhcpStaticMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCP static mapping on a DHCP server. Identified by the parent interface and the client MAC address.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The interface of the parent DHCP server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mac": schema.StringAttribute{
				Required:    true,
				Description: "The MAC address of the client this mapping is for.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ipaddr":                 optionalStringAttribute("The IP address to assign this client via DHCP."),
			"cid":                    optionalStringAttribute("The client identifier of the client this mapping is for."),
			"hostname":               optionalStringAttribute("The hostname to assign this client via DHCP."),
			"domain":                 optionalStringAttribute("The domain to be assigned via DHCP."),
			"domainsearchlist":       schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The domain search list to provide via DHCP."},
			"defaultleasetime":       optionalIntAttribute("The default DHCP lease validity period (in seconds)."),
			"maxleasetime":           optionalIntAttribute("The maximum DHCP lease validity period (in seconds)."),
			"gateway":                optionalStringAttribute("The gateway IPv4 address to provide via DHCP."),
			"dnsserver":              schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The DNS servers to provide via DHCP."},
			"winsserver":             schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The WINS servers to provide via DHCP."},
			"ntpserver":              schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The NTP servers to provide via DHCP."},
			"arp_table_static_entry": optionalBoolAttribute("Create an ARP table static entry for this client."),
			"descr":                  optionalStringAttribute("The description for this static mapping."),
		},
	}
}

func (r *dhcpStaticMappingResource) key(iface, mac string) string { return iface + "|" + mac }

func (r *dhcpStaticMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dhcpStaticMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "mac", plan.MAC)
	setString(payload, "ipaddr", plan.Ipaddr)
	setString(payload, "cid", plan.CID)
	setString(payload, "hostname", plan.Hostname)
	setString(payload, "domain", plan.Domain)
	setStringList(payload, "domainsearchlist", plan.Domainsearchlist)
	setInt(payload, "defaultleasetime", plan.DefaultLeaseTime)
	setInt(payload, "maxleasetime", plan.MaxLeaseTime)
	setString(payload, "gateway", plan.Gateway)
	setStringList(payload, "dnsserver", plan.DNSServer)
	setStringList(payload, "winsserver", plan.WINSServer)
	setStringList(payload, "ntpserver", plan.NTPServer)
	setBool(payload, "arp_table_static_entry", plan.ARPTableStaticEntry)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, dhcpStaticMappingSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create DHCP static mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, plan.MAC.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpStaticMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dhcpStaticMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	mac := state.MAC.ValueString()
	if iface == "" {
		iface, mac = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, dhcpStaticMappingPlural, formatID(pid), "mac", mac)
	if err != nil {
		resp.Diagnostics.AddError("failed to read DHCP static mapping", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, mac))
	state.ParentID = types.StringValue(iface)
	state.MAC = types.StringValue(mac)
	state.Ipaddr = strValue(getString(obj, "ipaddr"))
	state.CID = strValue(getString(obj, "cid"))
	state.Hostname = strValue(getString(obj, "hostname"))
	state.Domain = strValue(getString(obj, "domain"))
	state.Domainsearchlist = strListValue(ctx, getStringSlice(obj, "domainsearchlist"))
	state.DefaultLeaseTime = intValue(getInt(obj, "defaultleasetime"))
	state.MaxLeaseTime = intValue(getInt(obj, "maxleasetime"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.DNSServer = strListValue(ctx, getStringSlice(obj, "dnsserver"))
	state.WINSServer = strListValue(ctx, getStringSlice(obj, "winsserver"))
	state.NTPServer = strListValue(ctx, getStringSlice(obj, "ntpserver"))
	state.ARPTableStaticEntry = boolValue(getBool(obj, "arp_table_static_entry"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dhcpStaticMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dhcpStaticMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	mac := plan.MAC.ValueString()
	if iface == "" {
		iface, mac = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpStaticMappingPlural, formatID(pid), "mac", mac)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update DHCP static mapping", err.Error())
		} else {
			resp.Diagnostics.AddError("DHCP static mapping not found", "mapping for "+mac+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "mac", plan.MAC)
	setString(payload, "ipaddr", plan.Ipaddr)
	setString(payload, "cid", plan.CID)
	setString(payload, "hostname", plan.Hostname)
	setString(payload, "domain", plan.Domain)
	setStringList(payload, "domainsearchlist", plan.Domainsearchlist)
	setInt(payload, "defaultleasetime", plan.DefaultLeaseTime)
	setInt(payload, "maxleasetime", plan.MaxLeaseTime)
	setString(payload, "gateway", plan.Gateway)
	setStringList(payload, "dnsserver", plan.DNSServer)
	setStringList(payload, "winsserver", plan.WINSServer)
	setStringList(payload, "ntpserver", plan.NTPServer)
	setBool(payload, "arp_table_static_entry", plan.ARPTableStaticEntry)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, dhcpStaticMappingSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update DHCP static mapping", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, mac))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpStaticMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dhcpStaticMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	mac := state.MAC.ValueString()
	if iface == "" {
		iface, mac = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpStaticMappingPlural, formatID(pid), "mac", mac)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP static mapping", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dhcpStaticMappingSingular, client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP static mapping", err.Error())
	}
}

func (r *dhcpStaticMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_address_pool
// Parent: DHCP server (natural key = interface). Child key: range_from.
// Resource ID = <interface>|<range_from>.
// ---------------------------------------------------------------------------

type dhcpAddressPoolResource struct{ client *client.Client }

var _ resource.Resource = (*dhcpAddressPoolResource)(nil)
var _ resource.ResourceWithConfigure = (*dhcpAddressPoolResource)(nil)
var _ resource.ResourceWithImportState = (*dhcpAddressPoolResource)(nil)

const (
	dhcpAddressPoolPlural   = "/api/v2/services/dhcp_server/address_pools"
	dhcpAddressPoolSingular = "/api/v2/services/dhcp_server/address_pool"
)

type dhcpAddressPoolModel struct {
	ID               types.String `tfsdk:"id"`
	ParentID         types.String `tfsdk:"parent_id"`
	RangeFrom        types.String `tfsdk:"range_from"`
	RangeTo          types.String `tfsdk:"range_to"`
	Domain           types.String `tfsdk:"domain"`
	MACAllow         types.List   `tfsdk:"mac_allow"`
	MACDeny          types.List   `tfsdk:"mac_deny"`
	Domainsearchlist types.List   `tfsdk:"domainsearchlist"`
	DefaultLeaseTime types.Int64  `tfsdk:"defaultleasetime"`
	MaxLeaseTime     types.Int64  `tfsdk:"maxleasetime"`
	Gateway          types.String `tfsdk:"gateway"`
	DNSServer        types.List   `tfsdk:"dnsserver"`
	WINSServer       types.List   `tfsdk:"winsserver"`
	NTPServer        types.List   `tfsdk:"ntpserver"`
	IgnoreBootP      types.Bool   `tfsdk:"ignorebootp"`
	IgnoreClientUIDs types.Bool   `tfsdk:"ignoreclientuids"`
	DenyUnknown      types.String `tfsdk:"denyunknown"`
}

func NewDHCPServerAddressPoolResource() resource.Resource { return &dhcpAddressPoolResource{} }

func (r *dhcpAddressPoolResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dhcp_address_pool"
}
func (r *dhcpAddressPoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dhcpAddressPoolResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCP address pool on a DHCP server. Identified by the parent interface and the starting address.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The interface of the parent DHCP server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"range_from": schema.StringAttribute{
				Required:    true,
				Description: "The starting IP address for this address pool.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"range_to":         requiredStringAttribute("The ending IP address for this address pool."),
			"domain":           optionalStringAttribute("The domain to be assigned via DHCP."),
			"mac_allow":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "MAC addresses this pool is allowed to provide leases for."},
			"mac_deny":         schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "MAC addresses this pool is not allowed to provide leases for."},
			"domainsearchlist": schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The domain search list to provide via DHCP."},
			"defaultleasetime": optionalIntAttribute("The default DHCP lease validity period (in seconds)."),
			"maxleasetime":     optionalIntAttribute("The maximum DHCP lease validity period (in seconds)."),
			"gateway":          optionalStringAttribute("The gateway IPv4 address to provide via DHCP."),
			"dnsserver":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The DNS servers to provide via DHCP."},
			"winsserver":       schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The WINS servers to provide via DHCP."},
			"ntpserver":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "The NTP servers to provide via DHCP."},
			"ignorebootp":      optionalBoolAttribute("Ignore BOOTP requests."),
			"ignoreclientuids": optionalBoolAttribute("Ignore client unique identifiers (UIDs)."),
			"denyunknown":      enumAttribute("Deny unknown clients: enabled or class.", "enabled", "class"),
		},
	}
}

func (r *dhcpAddressPoolResource) key(iface, rangeFrom string) string { return iface + "|" + rangeFrom }

func (r *dhcpAddressPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dhcpAddressPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "range_from", plan.RangeFrom)
	setString(payload, "range_to", plan.RangeTo)
	setString(payload, "domain", plan.Domain)
	setStringList(payload, "mac_allow", plan.MACAllow)
	setStringList(payload, "mac_deny", plan.MACDeny)
	setStringList(payload, "domainsearchlist", plan.Domainsearchlist)
	setInt(payload, "defaultleasetime", plan.DefaultLeaseTime)
	setInt(payload, "maxleasetime", plan.MaxLeaseTime)
	setString(payload, "gateway", plan.Gateway)
	setStringList(payload, "dnsserver", plan.DNSServer)
	setStringList(payload, "winsserver", plan.WINSServer)
	setStringList(payload, "ntpserver", plan.NTPServer)
	setBool(payload, "ignorebootp", plan.IgnoreBootP)
	setBool(payload, "ignoreclientuids", plan.IgnoreClientUIDs)
	setString(payload, "denyunknown", plan.DenyUnknown)
	if _, err := r.client.Create(ctx, dhcpAddressPoolSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create DHCP address pool", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, plan.RangeFrom.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpAddressPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dhcpAddressPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	rangeFrom := state.RangeFrom.ValueString()
	if iface == "" {
		iface, rangeFrom = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, dhcpAddressPoolPlural, formatID(pid), "range_from", rangeFrom)
	if err != nil {
		resp.Diagnostics.AddError("failed to read DHCP address pool", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, rangeFrom))
	state.ParentID = types.StringValue(iface)
	state.RangeFrom = types.StringValue(rangeFrom)
	state.RangeTo = strValue(getString(obj, "range_to"))
	state.Domain = strValue(getString(obj, "domain"))
	state.MACAllow = strListValue(ctx, getStringSlice(obj, "mac_allow"))
	state.MACDeny = strListValue(ctx, getStringSlice(obj, "mac_deny"))
	state.Domainsearchlist = strListValue(ctx, getStringSlice(obj, "domainsearchlist"))
	state.DefaultLeaseTime = intValue(getInt(obj, "defaultleasetime"))
	state.MaxLeaseTime = intValue(getInt(obj, "maxleasetime"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.DNSServer = strListValue(ctx, getStringSlice(obj, "dnsserver"))
	state.WINSServer = strListValue(ctx, getStringSlice(obj, "winsserver"))
	state.NTPServer = strListValue(ctx, getStringSlice(obj, "ntpserver"))
	state.IgnoreBootP = boolValue(getBool(obj, "ignorebootp"))
	state.IgnoreClientUIDs = boolValue(getBool(obj, "ignoreclientuids"))
	state.DenyUnknown = strValue(getString(obj, "denyunknown"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dhcpAddressPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dhcpAddressPoolModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	rangeFrom := plan.RangeFrom.ValueString()
	if iface == "" {
		iface, rangeFrom = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpAddressPoolPlural, formatID(pid), "range_from", rangeFrom)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update DHCP address pool", err.Error())
		} else {
			resp.Diagnostics.AddError("DHCP address pool not found", "pool starting at "+rangeFrom+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "range_from", plan.RangeFrom)
	setString(payload, "range_to", plan.RangeTo)
	setString(payload, "domain", plan.Domain)
	setStringList(payload, "mac_allow", plan.MACAllow)
	setStringList(payload, "mac_deny", plan.MACDeny)
	setStringList(payload, "domainsearchlist", plan.Domainsearchlist)
	setInt(payload, "defaultleasetime", plan.DefaultLeaseTime)
	setInt(payload, "maxleasetime", plan.MaxLeaseTime)
	setString(payload, "gateway", plan.Gateway)
	setStringList(payload, "dnsserver", plan.DNSServer)
	setStringList(payload, "winsserver", plan.WINSServer)
	setStringList(payload, "ntpserver", plan.NTPServer)
	setBool(payload, "ignorebootp", plan.IgnoreBootP)
	setBool(payload, "ignoreclientuids", plan.IgnoreClientUIDs)
	setString(payload, "denyunknown", plan.DenyUnknown)
	if _, err := r.client.Update(ctx, dhcpAddressPoolSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update DHCP address pool", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, rangeFrom))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpAddressPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dhcpAddressPoolModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	rangeFrom := state.RangeFrom.ValueString()
	if iface == "" {
		iface, rangeFrom = splitRouteKey(state.ID.ValueString())
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpAddressPoolPlural, formatID(pid), "range_from", rangeFrom)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP address pool", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dhcpAddressPoolSingular, client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP address pool", err.Error())
	}
}

func (r *dhcpAddressPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_custom_option
// Parent: DHCP server (natural key = interface). Child key: number.
// Resource ID = <interface>|<number>.
// ---------------------------------------------------------------------------

type dhcpCustomOptionResource struct{ client *client.Client }

var _ resource.Resource = (*dhcpCustomOptionResource)(nil)
var _ resource.ResourceWithConfigure = (*dhcpCustomOptionResource)(nil)
var _ resource.ResourceWithImportState = (*dhcpCustomOptionResource)(nil)

const (
	dhcpCustomOptionPlural   = "/api/v2/services/dhcp_server/custom_options"
	dhcpCustomOptionSingular = "/api/v2/services/dhcp_server/custom_option"
)

type dhcpCustomOptionModel struct {
	ID       types.String `tfsdk:"id"`
	ParentID types.String `tfsdk:"parent_id"`
	Number   types.Int64  `tfsdk:"number"`
	Type     types.String `tfsdk:"type"`
	Value    types.String `tfsdk:"value"`
}

func NewDHCPServerCustomOptionResource() resource.Resource { return &dhcpCustomOptionResource{} }

func (r *dhcpCustomOptionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dhcp_custom_option"
}
func (r *dhcpCustomOptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dhcpCustomOptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DHCP custom option on a DHCP server. Identified by the parent interface and the option number.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The interface of the parent DHCP server.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"number": schema.Int64Attribute{
				Required:    true,
				Description: "The DHCP option number to configure.",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of value to configure for the option.",
				Validators: []validator.String{
					stringvalidator.OneOf("text", "string", "boolean", "unsigned integer 8", "unsigned integer 16", "unsigned integer 32", "signed integer 8", "signed integer 16", "signed integer 32", "ip-address"),
				},
			},
			"value": requiredStringAttribute("The value to configure for the option."),
		},
	}
}

func (r *dhcpCustomOptionResource) key(iface string, number int64) string {
	return iface + "|" + formatID(number)
}

func (r *dhcpCustomOptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dhcpCustomOptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setInt(payload, "number", plan.Number)
	setString(payload, "type", plan.Type)
	setString(payload, "value", plan.Value)
	if _, err := r.client.Create(ctx, dhcpCustomOptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create DHCP custom option", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, plan.Number.ValueInt64()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpCustomOptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dhcpCustomOptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	number := state.Number.ValueInt64()
	if iface == "" {
		var numberStr string
		iface, numberStr = splitRouteKey(state.ID.ValueString())
		number = atoi64(numberStr)
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, dhcpCustomOptionPlural, formatID(pid), "number", formatID(number))
	if err != nil {
		resp.Diagnostics.AddError("failed to read DHCP custom option", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(iface, number))
	state.ParentID = types.StringValue(iface)
	state.Number = types.Int64Value(number)
	state.Type = strValue(getString(obj, "type"))
	state.Value = strValue(getString(obj, "value"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dhcpCustomOptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dhcpCustomOptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.ParentID.ValueString()
	number := plan.Number.ValueInt64()
	if iface == "" {
		var numberStr string
		iface, numberStr = splitRouteKey(plan.ID.ValueString())
		number = atoi64(numberStr)
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpCustomOptionPlural, formatID(pid), "number", formatID(number))
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update DHCP custom option", err.Error())
		} else {
			resp.Diagnostics.AddError("DHCP custom option not found", "option "+formatID(number)+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setInt(payload, "number", plan.Number)
	setString(payload, "type", plan.Type)
	setString(payload, "value", plan.Value)
	if _, err := r.client.Update(ctx, dhcpCustomOptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update DHCP custom option", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(iface, number))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpCustomOptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dhcpCustomOptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.ParentID.ValueString()
	number := state.Number.ValueInt64()
	if iface == "" {
		var numberStr string
		iface, numberStr = splitRouteKey(state.ID.ValueString())
		number = atoi64(numberStr)
	}
	pid, err := resolveParentID(ctx, r.client, dhcpParentPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent DHCP server", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dhcpCustomOptionPlural, formatID(pid), "number", formatID(number))
	if err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP custom option", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dhcpCustomOptionSingular, client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP custom option", err.Error())
	}
}

func (r *dhcpCustomOptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_domain_override
// Key: domain.
// ---------------------------------------------------------------------------

type dnsResolverDomainOverrideResource struct{ client *client.Client }

var _ resource.Resource = (*dnsResolverDomainOverrideResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsResolverDomainOverrideResource)(nil)
var _ resource.ResourceWithImportState = (*dnsResolverDomainOverrideResource)(nil)

const (
	dnsResolverDomainOverridePlural   = "/api/v2/services/dns_resolver/domain_overrides"
	dnsResolverDomainOverrideSingular = "/api/v2/services/dns_resolver/domain_override"
)

type dnsResolverDomainOverrideModel struct {
	ID                 types.String `tfsdk:"id"`
	Domain             types.String `tfsdk:"domain"`
	IP                 types.String `tfsdk:"ip"`
	Descr              types.String `tfsdk:"descr"`
	ForwardTLSUpstream types.Bool   `tfsdk:"forward_tls_upstream"`
	TLSHostname        types.String `tfsdk:"tls_hostname"`
}

func NewDNSResolverDomainOverrideResource() resource.Resource {
	return &dnsResolverDomainOverrideResource{}
}

func (r *dnsResolverDomainOverrideResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_resolver_domain_override"
}
func (r *dnsResolverDomainOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dnsResolverDomainOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS resolver domain override. Identified by `domain`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain to override.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip":                   requiredStringAttribute("The IP address to which the domain should resolve."),
			"descr":                optionalStringAttribute("The description for this domain override."),
			"forward_tls_upstream": optionalBoolAttribute("Forward DNS queries to the upstream DNS server using TLS."),
			"tls_hostname":         optionalStringAttribute("The hostname to use for the TLS connection to the upstream DNS server."),
		},
	}
}

func (r *dnsResolverDomainOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsResolverDomainOverrideModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "domain", plan.Domain)
	setString(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "forward_tls_upstream", plan.ForwardTLSUpstream)
	setString(payload, "tls_hostname", plan.TLSHostname)
	if _, err := r.client.Create(ctx, dnsResolverDomainOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create domain override", err.Error())
		return
	}
	plan.ID = plan.Domain
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverDomainOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsResolverDomainOverrideModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	domain := state.Domain.ValueString()
	if domain == "" {
		domain = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, dnsResolverDomainOverridePlural, "domain", domain)
	if err != nil {
		resp.Diagnostics.AddError("failed to read domain override", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(domain)
	state.Domain = types.StringValue(domain)
	state.IP = strValue(getString(obj, "ip"))
	state.Descr = strValue(getString(obj, "descr"))
	state.ForwardTLSUpstream = boolValue(getBool(obj, "forward_tls_upstream"))
	state.TLSHostname = strValue(getString(obj, "tls_hostname"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsResolverDomainOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsResolverDomainOverrideModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	domain := plan.Domain.ValueString()
	if domain == "" {
		domain = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, dnsResolverDomainOverridePlural, "domain", domain)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update domain override", err.Error())
		} else {
			resp.Diagnostics.AddError("domain override not found", domain+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "domain", plan.Domain)
	setString(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	setBool(payload, "forward_tls_upstream", plan.ForwardTLSUpstream)
	setString(payload, "tls_hostname", plan.TLSHostname)
	if _, err := r.client.Update(ctx, dnsResolverDomainOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update domain override", err.Error())
		return
	}
	plan.ID = types.StringValue(domain)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverDomainOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsResolverDomainOverrideModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	domain := state.Domain.ValueString()
	if domain == "" {
		domain = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, dnsResolverDomainOverridePlural, "domain", domain)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete domain override", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dnsResolverDomainOverrideSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete domain override", err.Error())
	}
}

func (r *dnsResolverDomainOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_host_override_alias
// Parent: DNS resolver host override (composite key host|domain). Child key: host.
// Resource ID = <parent host>|<parent domain>|<alias host>.
// ---------------------------------------------------------------------------

type dnsResolverHostOverrideAliasResource struct{ client *client.Client }

var _ resource.Resource = (*dnsResolverHostOverrideAliasResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsResolverHostOverrideAliasResource)(nil)
var _ resource.ResourceWithImportState = (*dnsResolverHostOverrideAliasResource)(nil)

const (
	dnsResolverHostOverrideAliasPlural   = "/api/v2/services/dns_resolver/host_override/aliases"
	dnsResolverHostOverrideAliasSingular = "/api/v2/services/dns_resolver/host_override/alias"
	dnsResolverHostOverrideParentPlural  = "/api/v2/services/dns_resolver/host_overrides"
)

type dnsResolverHostOverrideAliasModel struct {
	ID       types.String `tfsdk:"id"`
	ParentID types.String `tfsdk:"parent_id"`
	Host     types.String `tfsdk:"host"`
	Domain   types.String `tfsdk:"domain"`
	Descr    types.String `tfsdk:"descr"`
}

func NewDNSResolverHostOverrideAliasResource() resource.Resource {
	return &dnsResolverHostOverrideAliasResource{}
}

func (r *dnsResolverHostOverrideAliasResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_resolver_host_override_alias"
}
func (r *dnsResolverHostOverrideAliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dnsResolverHostOverrideAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS resolver host override alias. Identified by the parent host override (`host|domain`) and the alias hostname.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent host override key, `host|domain`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname portion of the host override alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain portion of the host override alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"descr": optionalStringAttribute("A detailed description for this host override alias."),
		},
	}
}

func (r *dnsResolverHostOverrideAliasResource) key(parentKey, aliasHost string) string {
	return parentKey + "|" + aliasHost
}

func (r *dnsResolverHostOverrideAliasResource) resolveParent(ctx context.Context, parentKey string) (any, error) {
	parentHost, parentDomain := splitRouteKey(parentKey)
	return resolveParentIDByKeys(ctx, r.client, dnsResolverHostOverrideParentPlural, map[string]string{"host": parentHost, "domain": parentDomain})
}

func (r *dnsResolverHostOverrideAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsResolverHostOverrideAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, dnsResolverHostOverrideAliasSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create host override alias", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parentKey, plan.Host.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverHostOverrideAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsResolverHostOverrideAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	aliasHost := state.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(state.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, dnsResolverHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil {
		resp.Diagnostics.AddError("failed to read host override alias", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parentKey, aliasHost))
	state.ParentID = types.StringValue(parentKey)
	state.Host = types.StringValue(aliasHost)
	state.Domain = strValue(getString(obj, "domain"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsResolverHostOverrideAliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsResolverHostOverrideAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	aliasHost := plan.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dnsResolverHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update host override alias", err.Error())
		} else {
			resp.Diagnostics.AddError("host override alias not found", aliasHost+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, dnsResolverHostOverrideAliasSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update host override alias", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parentKey, aliasHost))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverHostOverrideAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsResolverHostOverrideAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	aliasHost := state.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(state.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dnsResolverHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete host override alias", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dnsResolverHostOverrideAliasSingular, client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete host override alias", err.Error())
	}
}

func (r *dnsResolverHostOverrideAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_forwarder_host_override
// Key: composite host|domain.
// ---------------------------------------------------------------------------

type dnsForwarderHostOverrideResource struct{ client *client.Client }

var _ resource.Resource = (*dnsForwarderHostOverrideResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsForwarderHostOverrideResource)(nil)
var _ resource.ResourceWithImportState = (*dnsForwarderHostOverrideResource)(nil)

const (
	dnsForwarderHostOverridePlural   = "/api/v2/services/dns_forwarder/host_overrides"
	dnsForwarderHostOverrideSingular = "/api/v2/services/dns_forwarder/host_override"
)

var dnsForwarderAliasAttrTypes = map[string]attr.Type{
	"host":        types.StringType,
	"domain":      types.StringType,
	"description": types.StringType,
}

type dnsForwarderHostOverrideModel struct {
	ID      types.String `tfsdk:"id"`
	Host    types.String `tfsdk:"host"`
	Domain  types.String `tfsdk:"domain"`
	IP      types.String `tfsdk:"ip"`
	Descr   types.String `tfsdk:"descr"`
	Aliases types.List   `tfsdk:"aliases"`
}

func NewDNSForwarderHostOverrideResource() resource.Resource {
	return &dnsForwarderHostOverrideResource{}
}

func (r *dnsForwarderHostOverrideResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_forwarder_host_override"
}
func (r *dnsForwarderHostOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dnsForwarderHostOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS forwarder host override. Identified by `host` and `domain`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname of this override.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain of this override.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ip":    requiredStringAttribute("The IP address of this override."),
			"descr": optionalStringAttribute("The description for this override."),
			"aliases": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The aliases for this override.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host":        requiredStringAttribute("Alias hostname."),
						"domain":      requiredStringAttribute("Alias domain."),
						"description": optionalStringAttribute("The description of this override alias."),
					},
				},
			},
		},
	}
}

func (r *dnsForwarderHostOverrideResource) key(host, domain string) string {
	return host + "|" + domain
}

func (r *dnsForwarderHostOverrideResource) find(ctx context.Context, host, domain string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, dnsForwarderHostOverridePlural, map[string]string{"host": host, "domain": domain})
}

func (r *dnsForwarderHostOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsForwarderHostOverrideModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	payload["aliases"] = listObjects(plan.Aliases)
	if _, err := r.client.Create(ctx, dnsForwarderHostOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create host override", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan.Host.ValueString(), plan.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsForwarderHostOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsForwarderHostOverrideModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	host := state.Host.ValueString()
	domain := state.Domain.ValueString()
	if host == "" {
		host, domain = splitRouteKey(state.ID.ValueString())
	}
	_, obj, found, err := r.find(ctx, host, domain)
	if err != nil {
		resp.Diagnostics.AddError("failed to read host override", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(host, domain))
	state.Host = types.StringValue(host)
	state.Domain = types.StringValue(domain)
	state.IP = strValue(getString(obj, "ip"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Aliases = objectsToListValue(ctx, getSliceMap(obj, "aliases"), types.ObjectType{AttrTypes: dnsForwarderAliasAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsForwarderHostOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsForwarderHostOverrideModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	host := plan.Host.ValueString()
	domain := plan.Domain.ValueString()
	if host == "" {
		host, domain = splitRouteKey(plan.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, host, domain)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update host override", err.Error())
		} else {
			resp.Diagnostics.AddError("host override not found", host+"."+domain+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	payload["aliases"] = listObjects(plan.Aliases)
	if _, err := r.client.Update(ctx, dnsForwarderHostOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update host override", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(host, domain))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsForwarderHostOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsForwarderHostOverrideModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	host := state.Host.ValueString()
	domain := state.Domain.ValueString()
	if host == "" {
		host, domain = splitRouteKey(state.ID.ValueString())
	}
	id, _, found, err := r.find(ctx, host, domain)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete host override", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dnsForwarderHostOverrideSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete host override", err.Error())
	}
}

func (r *dnsForwarderHostOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_forwarder_host_override_alias
// Parent: DNS forwarder host override (composite key host|domain). Child key: host.
// Resource ID = <parent host>|<parent domain>|<alias host>.
// ---------------------------------------------------------------------------

type dnsForwarderHostOverrideAliasResource struct{ client *client.Client }

var _ resource.Resource = (*dnsForwarderHostOverrideAliasResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsForwarderHostOverrideAliasResource)(nil)
var _ resource.ResourceWithImportState = (*dnsForwarderHostOverrideAliasResource)(nil)

const (
	dnsForwarderHostOverrideAliasPlural   = "/api/v2/services/dns_forwarder/host_override/aliases"
	dnsForwarderHostOverrideAliasSingular = "/api/v2/services/dns_forwarder/host_override/alias"
	dnsForwarderHostOverrideParentPlural  = "/api/v2/services/dns_forwarder/host_overrides"
)

type dnsForwarderHostOverrideAliasModel struct {
	ID          types.String `tfsdk:"id"`
	ParentID    types.String `tfsdk:"parent_id"`
	Host        types.String `tfsdk:"host"`
	Domain      types.String `tfsdk:"domain"`
	Description types.String `tfsdk:"description"`
}

func NewDNSForwarderHostOverrideAliasResource() resource.Resource {
	return &dnsForwarderHostOverrideAliasResource{}
}

func (r *dnsForwarderHostOverrideAliasResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_forwarder_host_override_alias"
}
func (r *dnsForwarderHostOverrideAliasResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dnsForwarderHostOverrideAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS forwarder host override alias. Identified by the parent host override (`host|domain`) and the alias hostname.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"parent_id": schema.StringAttribute{
				Required:    true,
				Description: "The parent host override key, `host|domain`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname of this override alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"domain": schema.StringAttribute{
				Required:    true,
				Description: "The domain of this override alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": optionalStringAttribute("The description of this override alias."),
		},
	}
}

func (r *dnsForwarderHostOverrideAliasResource) key(parentKey, aliasHost string) string {
	return parentKey + "|" + aliasHost
}

func (r *dnsForwarderHostOverrideAliasResource) resolveParent(ctx context.Context, parentKey string) (any, error) {
	parentHost, parentDomain := splitRouteKey(parentKey)
	return resolveParentIDByKeys(ctx, r.client, dnsForwarderHostOverrideParentPlural, map[string]string{"host": parentHost, "domain": parentDomain})
}

func (r *dnsForwarderHostOverrideAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsForwarderHostOverrideAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "description", plan.Description)
	if _, err := r.client.Create(ctx, dnsForwarderHostOverrideAliasSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create host override alias", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parentKey, plan.Host.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsForwarderHostOverrideAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsForwarderHostOverrideAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	aliasHost := state.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(state.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, dnsForwarderHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil {
		resp.Diagnostics.AddError("failed to read host override alias", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parentKey, aliasHost))
	state.ParentID = types.StringValue(parentKey)
	state.Host = types.StringValue(aliasHost)
	state.Domain = strValue(getString(obj, "domain"))
	state.Description = strValue(getString(obj, "description"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsForwarderHostOverrideAliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsForwarderHostOverrideAliasModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := plan.ParentID.ValueString()
	aliasHost := plan.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(plan.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dnsForwarderHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update host override alias", err.Error())
		} else {
			resp.Diagnostics.AddError("host override alias not found", aliasHost+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setString(payload, "description", plan.Description)
	if _, err := r.client.Update(ctx, dnsForwarderHostOverrideAliasSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update host override alias", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parentKey, aliasHost))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsForwarderHostOverrideAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsForwarderHostOverrideAliasModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parentKey := state.ParentID.ValueString()
	aliasHost := state.Host.ValueString()
	if parentKey == "" {
		parentKey, aliasHost = splitRouteKey(state.ID.ValueString())
	}
	pid, err := r.resolveParent(ctx, parentKey)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent host override", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, dnsForwarderHostOverrideAliasPlural, formatID(pid), "host", aliasHost)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete host override alias", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dnsForwarderHostOverrideAliasSingular, client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete host override alias", err.Error())
	}
}

func (r *dnsForwarderHostOverrideAliasResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_ntp_time_server
// Key: timeserver.
// ---------------------------------------------------------------------------

type ntpTimeServerResource struct{ client *client.Client }

var _ resource.Resource = (*ntpTimeServerResource)(nil)
var _ resource.ResourceWithConfigure = (*ntpTimeServerResource)(nil)
var _ resource.ResourceWithImportState = (*ntpTimeServerResource)(nil)

const (
	ntpTimeServerPlural   = "/api/v2/services/ntp/time_servers"
	ntpTimeServerSingular = "/api/v2/services/ntp/time_server"
)

type ntpTimeServerModel struct {
	ID         types.String `tfsdk:"id"`
	Timeserver types.String `tfsdk:"timeserver"`
	Type       types.String `tfsdk:"type"`
	Prefer     types.Bool   `tfsdk:"prefer"`
	Noselect   types.Bool   `tfsdk:"noselect"`
}

func NewNTPTimeServerResource() resource.Resource { return &ntpTimeServerResource{} }

func (r *ntpTimeServerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_ntp_time_server"
}
func (r *ntpTimeServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *ntpTimeServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NTP time server. Identified by `timeserver`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"timeserver": schema.StringAttribute{
				Required:    true,
				Description: "The IP or hostname of the remote NTP time server, pool or peer.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type":     enumAttribute("The type of this timeserver: server, pool or peer.", "server", "pool", "peer"),
			"prefer":   optionalBoolAttribute("Enable NTP favoring the use of this server more than all others."),
			"noselect": optionalBoolAttribute("Prevent NTP from using this timeserver, but continue collecting stats."),
		},
	}
}

func (r *ntpTimeServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ntpTimeServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "timeserver", plan.Timeserver)
	setString(payload, "type", plan.Type)
	setBool(payload, "prefer", plan.Prefer)
	setBool(payload, "noselect", plan.Noselect)
	if _, err := r.client.Create(ctx, ntpTimeServerSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create NTP time server", err.Error())
		return
	}
	plan.ID = plan.Timeserver
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ntpTimeServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ntpTimeServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	timeserver := state.Timeserver.ValueString()
	if timeserver == "" {
		timeserver = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, ntpTimeServerPlural, "timeserver", timeserver)
	if err != nil {
		resp.Diagnostics.AddError("failed to read NTP time server", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(timeserver)
	state.Timeserver = types.StringValue(timeserver)
	state.Type = strValue(getString(obj, "type"))
	state.Prefer = boolValue(getBool(obj, "prefer"))
	state.Noselect = boolValue(getBool(obj, "noselect"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ntpTimeServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ntpTimeServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	timeserver := plan.Timeserver.ValueString()
	if timeserver == "" {
		timeserver = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ntpTimeServerPlural, "timeserver", timeserver)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update NTP time server", err.Error())
		} else {
			resp.Diagnostics.AddError("NTP time server not found", timeserver+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "timeserver", plan.Timeserver)
	setString(payload, "type", plan.Type)
	setBool(payload, "prefer", plan.Prefer)
	setBool(payload, "noselect", plan.Noselect)
	if _, err := r.client.Update(ctx, ntpTimeServerSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update NTP time server", err.Error())
		return
	}
	plan.ID = types.StringValue(timeserver)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ntpTimeServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ntpTimeServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	timeserver := state.Timeserver.ValueString()
	if timeserver == "" {
		timeserver = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ntpTimeServerPlural, "timeserver", timeserver)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete NTP time server", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, ntpTimeServerSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete NTP time server", err.Error())
	}
}

func (r *ntpTimeServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_service_watchdog
// Key: name.
// ---------------------------------------------------------------------------

type serviceWatchdogResource struct{ client *client.Client }

var _ resource.Resource = (*serviceWatchdogResource)(nil)
var _ resource.ResourceWithConfigure = (*serviceWatchdogResource)(nil)
var _ resource.ResourceWithImportState = (*serviceWatchdogResource)(nil)

const (
	serviceWatchdogPlural   = "/api/v2/services/service_watchdogs"
	serviceWatchdogSingular = "/api/v2/services/service_watchdog"
)

type serviceWatchdogModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Notify      types.Bool   `tfsdk:"notify"`
	Enabled     types.Bool   `tfsdk:"enabled"`
}

func NewServiceWatchdogResource() resource.Resource { return &serviceWatchdogResource{} }

func (r *serviceWatchdogResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_service_watchdog"
}
func (r *serviceWatchdogResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *serviceWatchdogResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Service Watchdog watch for a service. Identified by the service `name`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the service to be watched.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{Computed: true, Description: "The description for the service being watched."},
			"notify":      optionalBoolAttribute("Enable or disable notifications being sent when Service Watchdog finds this service stopped."),
			"enabled":     schema.BoolAttribute{Computed: true, Description: "Indicates if this Service Watchdog is enabled or disabled."},
		},
	}
}

func (r *serviceWatchdogResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceWatchdogModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setBool(payload, "notify", plan.Notify)
	if _, err := r.client.Create(ctx, serviceWatchdogSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create service watchdog", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceWatchdogResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceWatchdogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, serviceWatchdogPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read service watchdog", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Description = strValue(getString(obj, "description"))
	state.Notify = boolValue(getBool(obj, "notify"))
	state.Enabled = boolValue(getBool(obj, "enabled"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceWatchdogResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceWatchdogModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, serviceWatchdogPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update service watchdog", err.Error())
		} else {
			resp.Diagnostics.AddError("service watchdog not found", name+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "name", plan.Name)
	setBool(payload, "notify", plan.Notify)
	if _, err := r.client.Update(ctx, serviceWatchdogSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update service watchdog", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceWatchdogResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceWatchdogModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, serviceWatchdogPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete service watchdog", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, serviceWatchdogSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete service watchdog", err.Error())
	}
}

func (r *serviceWatchdogResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
