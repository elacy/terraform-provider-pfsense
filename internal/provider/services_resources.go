package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_server
// ---------------------------------------------------------------------------

type dhcpServerResource struct{ client *client.Client }

var _ resource.Resource = (*dhcpServerResource)(nil)
var _ resource.ResourceWithConfigure = (*dhcpServerResource)(nil)
var _ resource.ResourceWithImportState = (*dhcpServerResource)(nil)

const (
	dhcpServerPlural   = "/api/v2/services/dhcp_servers"
	dhcpServerSingular = "/api/v2/services/dhcp_server"
)

var dhcpStaticMapAttrTypes = map[string]attr.Type{
	"mac":      types.StringType,
	"ipaddr":   types.StringType,
	"cid":      types.StringType,
	"hostname": types.StringType,
	"descr":    types.StringType,
}

type dhcpServerModel struct {
	ID           types.String `tfsdk:"id"`
	Interface    types.String `tfsdk:"interface"`
	Enable       types.Bool   `tfsdk:"enable"`
	RangeFrom    types.String `tfsdk:"range_from"`
	RangeTo      types.String `tfsdk:"range_to"`
	Domain       types.String `tfsdk:"domain"`
	DefaultLease types.Int64  `tfsdk:"defaultleasetime"`
	MaxLease     types.Int64  `tfsdk:"maxleasetime"`
	Gateway      types.String `tfsdk:"gateway"`
	DNSServer    types.List   `tfsdk:"dnsserver"`
	WINSServer   types.List   `tfsdk:"winsserver"`
	NTPServer    types.List   `tfsdk:"ntpserver"`
	DenyUnknown  types.String `tfsdk:"denyunknown"`
	StaticMap    types.List   `tfsdk:"staticmap"`
}

func NewDHCPServerResource() resource.Resource { return &dhcpServerResource{} }

func (r *dhcpServerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dhcp_server"
}
func (r *dhcpServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dhcpServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "Manages a DHCP server on an interface.",
		Attributes: map[string]schema.Attribute{
			"id":               computedIDAttribute(),
			"interface":        requiredNameAttribute(),
			"enable":           optionalBoolAttribute("Enable the DHCP server."),
			"range_from":       optionalStringAttribute("Start of the DHCP address range."),
			"range_to":         optionalStringAttribute("End of the DHCP address range."),
			"domain":           optionalStringAttribute("Domain name handed out to clients."),
			"defaultleasetime": optionalIntAttribute("Default lease time (seconds)."),
			"maxleasetime":     optionalIntAttribute("Maximum lease time (seconds)."),
			"gateway":          optionalStringAttribute("Gateway address handed out to clients."),
			"dnsserver":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "DNS servers handed out to clients."},
			"winsserver":       schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "WINS servers handed out to clients."},
			"ntpserver":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "NTP servers handed out to clients."},
			"denyunknown":      optionalStringAttribute("Deny unknown clients (enabled/disabled)."),
			"staticmap": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Static DHCP mappings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"mac":      requiredStringAttribute("Client MAC address."),
						"ipaddr":   requiredStringAttribute("Reserved IP address."),
						"cid":      optionalStringAttribute("Client identifier."),
						"hostname": optionalStringAttribute("Client hostname."),
						"descr":    optionalStringAttribute("Description."),
					},
				},
			},
		},
	}
}

func (r *dhcpServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dhcpServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, dhcpServerSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create DHCP server", err.Error())
		return
	}
	plan.ID = plan.Interface
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpServerResource) payload(m dhcpServerModel) map[string]any {
	p := map[string]any{}
	setString(p, "interface", m.Interface)
	setBool(p, "enable", m.Enable)
	setString(p, "range_from", m.RangeFrom)
	setString(p, "range_to", m.RangeTo)
	setString(p, "domain", m.Domain)
	setInt(p, "defaultleasetime", m.DefaultLease)
	setInt(p, "maxleasetime", m.MaxLease)
	setString(p, "gateway", m.Gateway)
	setStringList(p, "dnsserver", m.DNSServer)
	setStringList(p, "winsserver", m.WINSServer)
	setStringList(p, "ntpserver", m.NTPServer)
	setString(p, "denyunknown", m.DenyUnknown)
	p["staticmap"] = listObjects(m.StaticMap)
	return p
}

func (r *dhcpServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dhcpServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, dhcpServerPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to read DHCP server", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(iface)
	state.Interface = types.StringValue(iface)
	state.Enable = boolValue(getBool(obj, "enable"))
	state.RangeFrom = strValue(getString(obj, "range_from"))
	state.RangeTo = strValue(getString(obj, "range_to"))
	state.Domain = strValue(getString(obj, "domain"))
	state.DefaultLease = intValue(getInt(obj, "defaultleasetime"))
	state.MaxLease = intValue(getInt(obj, "maxleasetime"))
	state.Gateway = strValue(getString(obj, "gateway"))
	state.DNSServer = strListValue(ctx, getStringSlice(obj, "dnsserver"))
	state.WINSServer = strListValue(ctx, getStringSlice(obj, "winsserver"))
	state.NTPServer = strListValue(ctx, getStringSlice(obj, "ntpserver"))
	state.DenyUnknown = strValue(getString(obj, "denyunknown"))
	state.StaticMap = objectsToListValue(ctx, getSliceMap(obj, "staticmap"), types.ObjectType{AttrTypes: dhcpStaticMapAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dhcpServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dhcpServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := plan.Interface.ValueString()
	if iface == "" {
		iface = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, dhcpServerPlural, "interface", iface)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update DHCP server", err.Error())
		} else {
			resp.Diagnostics.AddError("DHCP server not found", "no DHCP server on "+iface)
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, dhcpServerSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update DHCP server", err.Error())
		return
	}
	plan.ID = types.StringValue(iface)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dhcpServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dhcpServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	iface := state.Interface.ValueString()
	if iface == "" {
		iface = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, dhcpServerPlural, "interface", iface)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP server", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, dhcpServerSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete DHCP server", err.Error())
	}
}

func (r *dhcpServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_host_override
// ---------------------------------------------------------------------------

type dnsResolverHostOverrideResource struct{ client *client.Client }

var _ resource.Resource = (*dnsResolverHostOverrideResource)(nil)
var _ resource.ResourceWithConfigure = (*dnsResolverHostOverrideResource)(nil)
var _ resource.ResourceWithImportState = (*dnsResolverHostOverrideResource)(nil)

const (
	dnsResolverHostOverridePlural   = "/api/v2/services/dns_resolver/host_overrides"
	dnsResolverHostOverrideSingular = "/api/v2/services/dns_resolver/host_override"
)

var dnsAliasAttrTypes = map[string]attr.Type{
	"host":   types.StringType,
	"domain": types.StringType,
	"descr":  types.StringType,
}

type dnsResolverHostOverrideModel struct {
	ID      types.String `tfsdk:"id"`
	Host    types.String `tfsdk:"host"`
	Domain  types.String `tfsdk:"domain"`
	IP      types.List   `tfsdk:"ip"`
	Descr   types.String `tfsdk:"descr"`
	Aliases types.List   `tfsdk:"aliases"`
}

func NewDNSResolverHostOverrideResource() resource.Resource {
	return &dnsResolverHostOverrideResource{}
}

func (r *dnsResolverHostOverrideResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_resolver_host_override"
}
func (r *dnsResolverHostOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *dnsResolverHostOverrideResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a DNS resolver host override. Identified by `host` and `domain`.",
		Version:     1,
		Attributes: map[string]schema.Attribute{
			"id":     computedIDAttribute(),
			"host":   requiredStringAttribute("The hostname (e.g. `www` or `*`)."),
			"domain": requiredStringAttribute("The domain (e.g. `example.com`)."),
			"ip":     schema.ListAttribute{ElementType: types.StringType, Required: true, Description: "IP addresses to resolve the host to."},
			"descr":  optionalStringAttribute("Description."),
			"aliases": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Aliases for this host override.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host":   requiredStringAttribute("Alias hostname."),
						"domain": requiredStringAttribute("Alias domain."),
						"descr":  optionalStringAttribute("Description."),
					},
				},
			},
		},
	}
}

func (r *dnsResolverHostOverrideResource) key(host, domain string) string { return host + "|" + domain }

func (r *dnsResolverHostOverrideResource) find(ctx context.Context, host, domain string) (any, map[string]any, bool, error) {
	return findByKeys(ctx, r.client, dnsResolverHostOverridePlural, map[string]string{"host": host, "domain": domain})
}

func (r *dnsResolverHostOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsResolverHostOverrideModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "host", plan.Host)
	setString(payload, "domain", plan.Domain)
	setStringList(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	payload["aliases"] = listObjects(plan.Aliases)
	if _, err := r.client.Create(ctx, dnsResolverHostOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create host override", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan.Host.ValueString(), plan.Domain.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverHostOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsResolverHostOverrideModel
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
	state.IP = strListValue(ctx, getStringSlice(obj, "ip"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Aliases = objectsToListValue(ctx, getSliceMap(obj, "aliases"), types.ObjectType{AttrTypes: dnsAliasAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *dnsResolverHostOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsResolverHostOverrideModel
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
	setStringList(payload, "ip", plan.IP)
	setString(payload, "descr", plan.Descr)
	payload["aliases"] = listObjects(plan.Aliases)
	if _, err := r.client.Update(ctx, dnsResolverHostOverrideSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update host override", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(host, domain))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *dnsResolverHostOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsResolverHostOverrideModel
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
	if err := r.client.Delete(ctx, dnsResolverHostOverrideSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete host override", err.Error())
	}
}

func (r *dnsResolverHostOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_cron_job
// ---------------------------------------------------------------------------

type cronJobResource struct{ client *client.Client }

var _ resource.Resource = (*cronJobResource)(nil)
var _ resource.ResourceWithConfigure = (*cronJobResource)(nil)
var _ resource.ResourceWithImportState = (*cronJobResource)(nil)

const (
	cronJobPlural   = "/api/v2/services/cron/jobs"
	cronJobSingular = "/api/v2/services/cron/job"
)

type cronJobModel struct {
	ID      types.String `tfsdk:"id"`
	Minute  types.String `tfsdk:"minute"`
	Hour    types.String `tfsdk:"hour"`
	Mday    types.String `tfsdk:"mday"`
	Month   types.String `tfsdk:"month"`
	Wday    types.String `tfsdk:"wday"`
	Who     types.String `tfsdk:"who"`
	Command types.String `tfsdk:"command"`
}

func NewServicesCronJobResource() resource.Resource { return &cronJobResource{} }

func (r *cronJobResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_cron_job"
}
func (r *cronJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *cronJobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a cron job.",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"minute":  requiredStringAttribute("Minute field (cron syntax)."),
			"hour":    requiredStringAttribute("Hour field (cron syntax)."),
			"mday":    requiredStringAttribute("Day-of-month field (cron syntax)."),
			"month":   requiredStringAttribute("Month field (cron syntax)."),
			"wday":    requiredStringAttribute("Day-of-week field (cron syntax)."),
			"who":     requiredStringAttribute("User to run the command as (usually `root`)."),
			"command": requiredStringAttribute("The command to execute."),
		},
	}
}

func (r *cronJobResource) payload(m cronJobModel) map[string]any {
	p := map[string]any{}
	setString(p, "minute", m.Minute)
	setString(p, "hour", m.Hour)
	setString(p, "mday", m.Mday)
	setString(p, "month", m.Month)
	setString(p, "wday", m.Wday)
	setString(p, "who", m.Who)
	setString(p, "command", m.Command)
	return p
}

func (r *cronJobResource) key(m cronJobModel) string {
	return m.Minute.ValueString() + " " + m.Hour.ValueString() + " " + m.Mday.ValueString() + " " +
		m.Month.ValueString() + " " + m.Wday.ValueString() + " " + m.Who.ValueString() + " " + m.Command.ValueString()
}

func (r *cronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cronJobModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, cronJobSingular, r.payload(plan)); err != nil {
		resp.Diagnostics.AddError("failed to create cron job", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cronJobModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	filters := map[string]string{
		"minute":  state.Minute.ValueString(),
		"hour":    state.Hour.ValueString(),
		"mday":    state.Mday.ValueString(),
		"month":   state.Month.ValueString(),
		"wday":    state.Wday.ValueString(),
		"who":     state.Who.ValueString(),
		"command": state.Command.ValueString(),
	}
	_, obj, found, err := findByKeys(ctx, r.client, cronJobPlural, filters)
	if err != nil {
		resp.Diagnostics.AddError("failed to read cron job", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(state))
	state.Minute = strValue(getString(obj, "minute"))
	state.Hour = strValue(getString(obj, "hour"))
	state.Mday = strValue(getString(obj, "mday"))
	state.Month = strValue(getString(obj, "month"))
	state.Wday = strValue(getString(obj, "wday"))
	state.Who = strValue(getString(obj, "who"))
	state.Command = strValue(getString(obj, "command"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cronJobModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	// Locate the existing job using the prior state values.
	var state cronJobModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	filters := map[string]string{
		"minute":  state.Minute.ValueString(),
		"hour":    state.Hour.ValueString(),
		"mday":    state.Mday.ValueString(),
		"month":   state.Month.ValueString(),
		"wday":    state.Wday.ValueString(),
		"who":     state.Who.ValueString(),
		"command": state.Command.ValueString(),
	}
	id, _, found, err := findByKeys(ctx, r.client, cronJobPlural, filters)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update cron job", err.Error())
		} else {
			resp.Diagnostics.AddError("cron job not found", "cron job no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, cronJobSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update cron job", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cronJobModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	filters := map[string]string{
		"minute":  state.Minute.ValueString(),
		"hour":    state.Hour.ValueString(),
		"mday":    state.Mday.ValueString(),
		"month":   state.Month.ValueString(),
		"wday":    state.Wday.ValueString(),
		"who":     state.Who.ValueString(),
		"command": state.Command.ValueString(),
	}
	id, _, found, err := findByKeys(ctx, r.client, cronJobPlural, filters)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete cron job", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, cronJobSingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete cron job", err.Error())
	}
}

func (r *cronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
