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
// pfsense_services_bind_access_list
// ---------------------------------------------------------------------------

type bindAccessListResource struct{ client *client.Client }

var _ resource.Resource = (*bindAccessListResource)(nil)
var _ resource.ResourceWithConfigure = (*bindAccessListResource)(nil)
var _ resource.ResourceWithImportState = (*bindAccessListResource)(nil)

const (
	bindAccessListPlural   = "/api/v2/services/bind/access_lists"
	bindAccessListSingular = "/api/v2/services/bind/access_list"
)

// bindAccessListEntryAttrTypes is the nested object shape for the `entries`
// member of a BIND access list (BINDAccessListEntry).
var bindAccessListEntryAttrTypes = map[string]attr.Type{
	"value":       types.StringType,
	"description": types.StringType,
}

type bindAccessListModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Entries     types.List   `tfsdk:"entries"`
}

func NewBindAccessListResource() resource.Resource { return &bindAccessListResource{} }

func (r *bindAccessListResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_bind_access_list"
}
func (r *bindAccessListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *bindAccessListResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a BIND access list (a named set of network CIDRs).",
		Attributes: map[string]schema.Attribute{
			"id":   computedIDAttribute(),
			"name": requiredNameAttribute(),
			"description": optionalStringAttribute(
				"A description for the access list.",
			),
			"entries": schema.ListNestedAttribute{
				Required:    true,
				Description: "The network entries for this access list.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The network CIDR to allow.",
						},
						"description": optionalStringAttribute(
							"A description of the access list entry.",
						),
					},
				},
			},
		},
	}
}

func (r *bindAccessListResource) payload(m bindAccessListModel) map[string]any {
	p := map[string]any{}
	setString(p, "name", m.Name)
	setString(p, "description", m.Description)
	p["entries"] = listObjects(m.Entries)
	return p
}

func (r *bindAccessListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bindAccessListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, bindAccessListSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create BIND access list", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindAccessListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bindAccessListModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, bindAccessListPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read BIND access list", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Description = strValue(getString(obj, "description"))
	state.Entries = objectsToListValue(ctx, getSliceMap(obj, "entries"), types.ObjectType{AttrTypes: bindAccessListEntryAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bindAccessListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bindAccessListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindAccessListPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update BIND access list", err.Error())
		} else {
			resp.Diagnostics.AddError("BIND access list not found", "access list "+name+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, bindAccessListSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update BIND access list", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindAccessListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bindAccessListModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindAccessListPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete BIND access list", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, bindAccessListSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete BIND access list", err.Error())
	}
}

func (r *bindAccessListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_bind_view
// ---------------------------------------------------------------------------

type bindViewResource struct{ client *client.Client }

var _ resource.Resource = (*bindViewResource)(nil)
var _ resource.ResourceWithConfigure = (*bindViewResource)(nil)
var _ resource.ResourceWithImportState = (*bindViewResource)(nil)

const (
	bindViewPlural   = "/api/v2/services/bind/views"
	bindViewSingular = "/api/v2/services/bind/view"
)

type bindViewModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Descr             types.String `tfsdk:"descr"`
	Recursion         types.Bool   `tfsdk:"recursion"`
	MatchClients      types.List   `tfsdk:"match_clients"`
	AllowRecursion    types.List   `tfsdk:"allow_recursion"`
	BindCustomOptions types.String `tfsdk:"bind_custom_options"`
}

func NewBindViewResource() resource.Resource { return &bindViewResource{} }

func (r *bindViewResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_bind_view"
}
func (r *bindViewResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *bindViewResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a BIND view (a filtering context for queries).",
		Attributes: map[string]schema.Attribute{
			"id":    computedIDAttribute(),
			"name":  requiredNameAttribute(),
			"descr": optionalStringAttribute("A description for the view."),
			"recursion": optionalBoolAttribute(
				"Enables or disables recursion for the view.",
			),
			"match_clients": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The access lists to match clients against.",
			},
			"allow_recursion": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The access lists to allow recursion for.",
			},
			"bind_custom_options": optionalStringAttribute(
				"Custom BIND options for the view.",
			),
		},
	}
}

func (r *bindViewResource) payload(m bindViewModel) map[string]any {
	p := map[string]any{}
	setString(p, "name", m.Name)
	setString(p, "descr", m.Descr)
	setBool(p, "recursion", m.Recursion)
	setStringList(p, "match_clients", m.MatchClients)
	setStringList(p, "allow_recursion", m.AllowRecursion)
	setString(p, "bind_custom_options", m.BindCustomOptions)
	return p
}

func (r *bindViewResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bindViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, bindViewSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create BIND view", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindViewResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bindViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, bindViewPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read BIND view", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Descr = strValue(getString(obj, "descr"))
	state.Recursion = boolValue(getBool(obj, "recursion"))
	state.MatchClients = strListValue(ctx, getStringSlice(obj, "match_clients"))
	state.AllowRecursion = strListValue(ctx, getStringSlice(obj, "allow_recursion"))
	state.BindCustomOptions = strValue(getString(obj, "bind_custom_options"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bindViewResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bindViewModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindViewPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update BIND view", err.Error())
		} else {
			resp.Diagnostics.AddError("BIND view not found", "view "+name+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, bindViewSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update BIND view", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindViewResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bindViewModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindViewPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete BIND view", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, bindViewSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete BIND view", err.Error())
	}
}

func (r *bindViewResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_bind_zone
// ---------------------------------------------------------------------------

type bindZoneResource struct{ client *client.Client }

var _ resource.Resource = (*bindZoneResource)(nil)
var _ resource.ResourceWithConfigure = (*bindZoneResource)(nil)
var _ resource.ResourceWithImportState = (*bindZoneResource)(nil)

const (
	bindZonePlural   = "/api/v2/services/bind/zones"
	bindZoneSingular = "/api/v2/services/bind/zone"
)

// bindZoneRecordAttrTypes is the nested shape for the `records` member of a
// BIND zone (BINDZoneRecord).
var bindZoneRecordAttrTypes = map[string]attr.Type{
	"name":     types.StringType,
	"type":     types.StringType,
	"rdata":    types.StringType,
	"priority": types.Int64Type,
}

type bindZoneModel struct {
	ID                 types.String `tfsdk:"id"`
	Disabled           types.Bool   `tfsdk:"disabled"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Type               types.String `tfsdk:"type"`
	View               types.List   `tfsdk:"view"`
	Reversev4          types.Bool   `tfsdk:"reversev4"`
	Reversev6          types.Bool   `tfsdk:"reversev6"`
	Rpz                types.Bool   `tfsdk:"rpz"`
	Custom             types.String `tfsdk:"custom"`
	Dnssec             types.Bool   `tfsdk:"dnssec"`
	Backupkeys         types.Bool   `tfsdk:"backupkeys"`
	Slaveip            types.String `tfsdk:"slaveip"`
	Forwarders         types.List   `tfsdk:"forwarders"`
	TTL                types.Int64  `tfsdk:"ttl"`
	Baseip             types.String `tfsdk:"baseip"`
	Nameserver         types.String `tfsdk:"nameserver"`
	Mail               types.String `tfsdk:"mail"`
	Serial             types.Int64  `tfsdk:"serial"`
	Refresh            types.String `tfsdk:"refresh"`
	Retry              types.String `tfsdk:"retry"`
	Expire             types.String `tfsdk:"expire"`
	Minimum            types.String `tfsdk:"minimum"`
	EnableUpdatepolicy types.Bool   `tfsdk:"enable_updatepolicy"`
	Updatepolicy       types.String `tfsdk:"updatepolicy"`
	Allowupdate        types.List   `tfsdk:"allowupdate"`
	Allowtransfer      types.List   `tfsdk:"allowtransfer"`
	Allowquery         types.List   `tfsdk:"allowquery"`
	Regdhcpstatic      types.Bool   `tfsdk:"regdhcpstatic"`
	Customzonerecords  types.String `tfsdk:"customzonerecords"`
	Records            types.List   `tfsdk:"records"`
}

func NewBindZoneResource() resource.Resource { return &bindZoneResource{} }

func (r *bindZoneResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_bind_zone"
}
func (r *bindZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *bindZoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a BIND zone (authoritative or forwarding).",
		Attributes: map[string]schema.Attribute{
			"id":          computedIDAttribute(),
			"name":        requiredNameAttribute(),
			"disabled":    optionalBoolAttribute("Disable this BIND zone."),
			"description": optionalStringAttribute("A description for this BIND zone."),
			"type": enumAttribute(
				"The type of this BIND zone: master, slave, forward or redirect.",
				"master", "slave", "forward", "redirect",
			),
			"view": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The views this BIND zone belongs to.",
			},
			"reversev4": optionalBoolAttribute("Enable reverse DNS for this BIND zone."),
			"reversev6": optionalBoolAttribute("Enable reverse IPv6 DNS for this BIND zone."),
			"rpz":       optionalBoolAttribute("Enable this zone as part of a response policy."),
			"custom":    optionalStringAttribute("Custom BIND options for this BIND zone."),
			"dnssec":    optionalBoolAttribute("Enable DNSSEC for this BIND zone."),
			"backupkeys": optionalBoolAttribute(
				"Enable backing up DNSSEC keys in the XML configuration for this BIND zone.",
			),
			"slaveip": optionalStringAttribute(
				"The IP address of the slave server for this BIND zone.",
			),
			"forwarders": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "The forwarders for this BIND zone.",
			},
			"ttl": optionalIntAttribute(
				"The default TTL interval (in seconds) for records within this BIND zone.",
			),
			"baseip":     requiredStringAttribute("The IP address of the base domain for this zone."),
			"nameserver": requiredStringAttribute("The SOA nameserver for this zone."),
			"mail":       requiredStringAttribute("The SOA email address (RNAME) for this zone."),
			"serial": schema.Int64Attribute{
				Required:    true,
				Description: "The SOA serial number for this zone.",
			},
			"refresh": optionalStringAttribute("The SOA refresh interval for this zone."),
			"retry":   optionalStringAttribute("The SOA retry interval for this zone."),
			"expire":  optionalStringAttribute("The SOA expiry interval for this zone."),
			"minimum": optionalStringAttribute("The SOA minimum TTL interval for this zone."),
			"enable_updatepolicy": optionalBoolAttribute(
				"Enable a specific dynamic update policy for this BIND zone.",
			),
			"updatepolicy": optionalStringAttribute(
				"The update policy for this BIND zone.",
			),
			"allowupdate": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The access lists allowed to submit dynamic updates for 'master' zones.",
			},
			"allowtransfer": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The access lists allowed to transfer this BIND zone.",
			},
			"allowquery": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "The access lists allowed to query this BIND zone.",
			},
			"regdhcpstatic": optionalBoolAttribute(
				"Register DHCP static mappings as records in this BIND zone.",
			),
			"customzonerecords": optionalStringAttribute(
				"Custom records for this BIND zone.",
			),
			"records": schema.ListNestedAttribute{
				Optional:    true,
				Description: "The records for this BIND zone.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The domain name for this record.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of record.",
							Validators: []validator.String{
								stringvalidator.OneOf("A", "AAAA", "CNAME", "MX", "NS", "LOC", "PTR", "SRV", "TXT", "SPF"),
							},
						},
						"rdata": schema.StringAttribute{
							Required:    true,
							Description: "The data for this record.",
						},
						"priority": schema.Int64Attribute{
							Required:    true,
							Description: "The priority for this record.",
						},
					},
				},
			},
		},
	}
}

func (r *bindZoneResource) payload(m bindZoneModel) map[string]any {
	p := map[string]any{}
	setBool(p, "disabled", m.Disabled)
	setString(p, "name", m.Name)
	setString(p, "description", m.Description)
	setString(p, "type", m.Type)
	setStringList(p, "view", m.View)
	setBool(p, "reversev4", m.Reversev4)
	setBool(p, "reversev6", m.Reversev6)
	setBool(p, "rpz", m.Rpz)
	setString(p, "custom", m.Custom)
	setBool(p, "dnssec", m.Dnssec)
	setBool(p, "backupkeys", m.Backupkeys)
	setString(p, "slaveip", m.Slaveip)
	setStringList(p, "forwarders", m.Forwarders)
	setInt(p, "ttl", m.TTL)
	setString(p, "baseip", m.Baseip)
	setString(p, "nameserver", m.Nameserver)
	setString(p, "mail", m.Mail)
	setInt(p, "serial", m.Serial)
	setString(p, "refresh", m.Refresh)
	setString(p, "retry", m.Retry)
	setString(p, "expire", m.Expire)
	setString(p, "minimum", m.Minimum)
	setBool(p, "enable_updatepolicy", m.EnableUpdatepolicy)
	setString(p, "updatepolicy", m.Updatepolicy)
	setStringList(p, "allowupdate", m.Allowupdate)
	setStringList(p, "allowtransfer", m.Allowtransfer)
	setStringList(p, "allowquery", m.Allowquery)
	setBool(p, "regdhcpstatic", m.Regdhcpstatic)
	setString(p, "customzonerecords", m.Customzonerecords)
	p["records"] = listObjects(m.Records)
	return p
}

func (r *bindZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bindZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, bindZoneSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create BIND zone", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bindZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, bindZonePlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read BIND zone", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.Description = strValue(getString(obj, "description"))
	state.Type = strValue(getString(obj, "type"))
	state.View = strListValue(ctx, getStringSlice(obj, "view"))
	state.Reversev4 = boolValue(getBool(obj, "reversev4"))
	state.Reversev6 = boolValue(getBool(obj, "reversev6"))
	state.Rpz = boolValue(getBool(obj, "rpz"))
	state.Custom = strValue(getString(obj, "custom"))
	state.Dnssec = boolValue(getBool(obj, "dnssec"))
	state.Backupkeys = boolValue(getBool(obj, "backupkeys"))
	state.Slaveip = strValue(getString(obj, "slaveip"))
	state.Forwarders = strListValue(ctx, getStringSlice(obj, "forwarders"))
	state.TTL = intValue(getInt(obj, "ttl"))
	state.Baseip = strValue(getString(obj, "baseip"))
	state.Nameserver = strValue(getString(obj, "nameserver"))
	state.Mail = strValue(getString(obj, "mail"))
	state.Serial = intValue(getInt(obj, "serial"))
	state.Refresh = strValue(getString(obj, "refresh"))
	state.Retry = strValue(getString(obj, "retry"))
	state.Expire = strValue(getString(obj, "expire"))
	state.Minimum = strValue(getString(obj, "minimum"))
	state.EnableUpdatepolicy = boolValue(getBool(obj, "enable_updatepolicy"))
	state.Updatepolicy = strValue(getString(obj, "updatepolicy"))
	state.Allowupdate = strListValue(ctx, getStringSlice(obj, "allowupdate"))
	state.Allowtransfer = strListValue(ctx, getStringSlice(obj, "allowtransfer"))
	state.Allowquery = strListValue(ctx, getStringSlice(obj, "allowquery"))
	state.Regdhcpstatic = boolValue(getBool(obj, "regdhcpstatic"))
	state.Customzonerecords = strValue(getString(obj, "customzonerecords"))
	state.Records = objectsToListValue(ctx, getSliceMap(obj, "records"), types.ObjectType{AttrTypes: bindZoneRecordAttrTypes})
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bindZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bindZoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindZonePlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update BIND zone", err.Error())
		} else {
			resp.Diagnostics.AddError("BIND zone not found", "zone "+name+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, bindZoneSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update BIND zone", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bindZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bindZoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, bindZonePlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete BIND zone", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, bindZoneSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete BIND zone", err.Error())
	}
}

func (r *bindZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_freeradius_mac
// ---------------------------------------------------------------------------

type freeradiusMACResource struct{ client *client.Client }

var _ resource.Resource = (*freeradiusMACResource)(nil)
var _ resource.ResourceWithConfigure = (*freeradiusMACResource)(nil)
var _ resource.ResourceWithImportState = (*freeradiusMACResource)(nil)

const (
	freeradiusMACPlural   = "/api/v2/services/freeradius/macs"
	freeradiusMACSingular = "/api/v2/services/freeradius/mac"
)

type freeradiusMACModel struct {
	ID                          types.String `tfsdk:"id"`
	MAC                         types.String `tfsdk:"mac"`
	Description                 types.String `tfsdk:"description"`
	FramedIPAddress             types.String `tfsdk:"framed_ip_address"`
	FramedIPNetmask             types.String `tfsdk:"framed_ip_netmask"`
	FramedRoute                 types.String `tfsdk:"framed_route"`
	FramedIPv6Address           types.String `tfsdk:"framed_ipv6_address"`
	FramedIPv6Route             types.String `tfsdk:"framed_ipv6_route"`
	VLANID                      types.String `tfsdk:"vlan_id"`
	WisprRedirectionURL         types.String `tfsdk:"wispr_redirection_url"`
	SimultaneousConnect         types.Int64  `tfsdk:"simultaneous_connect"`
	Expiration                  types.String `tfsdk:"expiration"`
	SessionTimeout              types.Int64  `tfsdk:"session_timeout"`
	LoginTime                   types.String `tfsdk:"login_time"`
	AmountOfTime                types.Int64  `tfsdk:"amount_of_time"`
	PointOfTime                 types.String `tfsdk:"point_of_time"`
	MaxTotalOctets              types.Int64  `tfsdk:"max_total_octets"`
	MaxTotalOctetsTimeRange     types.String `tfsdk:"max_total_octets_time_range"`
	MaxBandwidthDown            types.Int64  `tfsdk:"max_bandwidth_down"`
	MaxBandwidthUp              types.Int64  `tfsdk:"max_bandwidth_up"`
	AcctInterimInterval         types.Int64  `tfsdk:"acct_interim_interval"`
	TopAdditionalOptions        types.List   `tfsdk:"top_additional_options"`
	CheckItemsAdditionalOptions types.List   `tfsdk:"check_items_additional_options"`
	ReplyItemsAdditionalOptions types.List   `tfsdk:"reply_items_additional_options"`
}

func NewFreeradiusMACResource() resource.Resource { return &freeradiusMACResource{} }

func (r *freeradiusMACResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_freeradius_mac"
}
func (r *freeradiusMACResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *freeradiusMACResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a FreeRADIUS authorized-MAC entry.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"mac": schema.StringAttribute{
				Required:    true,
				Description: "The MAC address of the entry. May be delimited with '-' or ':' (e.g. 'aa:bb:cc:dd:ee:ff').",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description":       optionalStringAttribute("A description for this entry."),
			"framed_ip_address": optionalStringAttribute("Framed-IP-Address MUST be supported by NAS."),
			"framed_ip_netmask": optionalStringAttribute("Framed-IP-Netmask MUST be supported by NAS."),
			"framed_route":      optionalStringAttribute("Framed-Route must be supported by NAS."),
			"framed_ipv6_address": optionalStringAttribute(
				"Framed IPv6 address or prefix (e.g. 2001:db8:abab::5 or 2001:db8:abab::/64).",
			),
			"framed_ipv6_route": optionalStringAttribute("Framed-IPv6-Route must be supported by NAS."),
			"vlan_id": optionalStringAttribute(
				"The VLAN ID (integer from 1-4095) or the VLAN name for this entry.",
			),
			"wispr_redirection_url": optionalStringAttribute(
				"The URL the user should be redirected to after successful login.",
			),
			"simultaneous_connect": optionalIntAttribute(
				"The maximum number of simultaneous connections with this entry. Leave null for no limit.",
			),
			"expiration": optionalStringAttribute(
				"The date when this account should expire. Required format: Mmm dd yyyy (e.g. Jan 01 2030).",
			),
			"session_timeout": optionalIntAttribute(
				"The time this entry has until relogin (in seconds).",
			),
			"login_time": optionalStringAttribute(
				"The time when this entry should have access. Empty for no time restriction.",
			),
			"amount_of_time": optionalIntAttribute(
				"The amount of time this entry is allowed (in minutes) within the configured time period.",
			),
			"point_of_time": enumAttribute(
				"The time period after which the 'Amount of Time' is reset.",
				"Daily", "Weekly", "Monthly", "Forever",
			),
			"max_total_octets": optionalIntAttribute(
				"The amount of download and upload traffic (summarized) in megabytes (MB) for this entry.",
			),
			"max_total_octets_time_range": enumAttribute(
				"The time period for the amount of download and upload traffic.",
				"daily", "weekly", "monthly", "forever",
			),
			"max_bandwidth_down": optionalIntAttribute(
				"The maximum bandwidth for download in kilobits per second (Kbit/s).",
			),
			"max_bandwidth_up": optionalIntAttribute(
				"The maximum bandwidth for upload in kilobits per second (Kbit/s).",
			),
			"acct_interim_interval": optionalIntAttribute(
				"The interval in seconds which should elapse between interim-updates.",
			),
			"top_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS attributes placed at the TOP of this entry.",
			},
			"check_items_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS check-item attributes for this entry.",
			},
			"reply_items_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS reply-item attributes for this entry.",
			},
		},
	}
}

func (r *freeradiusMACResource) payload(m freeradiusMACModel) map[string]any {
	p := map[string]any{}
	setString(p, "mac", m.MAC)
	setString(p, "description", m.Description)
	setString(p, "framed_ip_address", m.FramedIPAddress)
	setString(p, "framed_ip_netmask", m.FramedIPNetmask)
	setString(p, "framed_route", m.FramedRoute)
	setString(p, "framed_ipv6_address", m.FramedIPv6Address)
	setString(p, "framed_ipv6_route", m.FramedIPv6Route)
	setString(p, "vlan_id", m.VLANID)
	setString(p, "wispr_redirection_url", m.WisprRedirectionURL)
	setInt(p, "simultaneous_connect", m.SimultaneousConnect)
	setString(p, "expiration", m.Expiration)
	setInt(p, "session_timeout", m.SessionTimeout)
	setString(p, "login_time", m.LoginTime)
	setInt(p, "amount_of_time", m.AmountOfTime)
	setString(p, "point_of_time", m.PointOfTime)
	setInt(p, "max_total_octets", m.MaxTotalOctets)
	setString(p, "max_total_octets_time_range", m.MaxTotalOctetsTimeRange)
	setInt(p, "max_bandwidth_down", m.MaxBandwidthDown)
	setInt(p, "max_bandwidth_up", m.MaxBandwidthUp)
	setInt(p, "acct_interim_interval", m.AcctInterimInterval)
	setStringList(p, "top_additional_options", m.TopAdditionalOptions)
	setStringList(p, "check_items_additional_options", m.CheckItemsAdditionalOptions)
	setStringList(p, "reply_items_additional_options", m.ReplyItemsAdditionalOptions)
	return p
}

func (r *freeradiusMACResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan freeradiusMACModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, freeradiusMACSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create FreeRADIUS MAC entry", err.Error())
		return
	}
	plan.ID = plan.MAC
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *freeradiusMACResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state freeradiusMACModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	mac := state.MAC.ValueString()
	if mac == "" {
		mac = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, freeradiusMACPlural, "mac", mac)
	if err != nil {
		resp.Diagnostics.AddError("failed to read FreeRADIUS MAC entry", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(mac)
	state.MAC = types.StringValue(mac)
	state.Description = strValue(getString(obj, "description"))
	state.FramedIPAddress = strValue(getString(obj, "framed_ip_address"))
	state.FramedIPNetmask = strValue(getString(obj, "framed_ip_netmask"))
	state.FramedRoute = strValue(getString(obj, "framed_route"))
	state.FramedIPv6Address = strValue(getString(obj, "framed_ipv6_address"))
	state.FramedIPv6Route = strValue(getString(obj, "framed_ipv6_route"))
	state.VLANID = strValue(getString(obj, "vlan_id"))
	state.WisprRedirectionURL = strValue(getString(obj, "wispr_redirection_url"))
	state.SimultaneousConnect = intValue(getInt(obj, "simultaneous_connect"))
	state.Expiration = strValue(getString(obj, "expiration"))
	state.SessionTimeout = intValue(getInt(obj, "session_timeout"))
	state.LoginTime = strValue(getString(obj, "login_time"))
	state.AmountOfTime = intValue(getInt(obj, "amount_of_time"))
	state.PointOfTime = strValue(getString(obj, "point_of_time"))
	state.MaxTotalOctets = intValue(getInt(obj, "max_total_octets"))
	state.MaxTotalOctetsTimeRange = strValue(getString(obj, "max_total_octets_time_range"))
	state.MaxBandwidthDown = intValue(getInt(obj, "max_bandwidth_down"))
	state.MaxBandwidthUp = intValue(getInt(obj, "max_bandwidth_up"))
	state.AcctInterimInterval = intValue(getInt(obj, "acct_interim_interval"))
	state.TopAdditionalOptions = strListValue(ctx, getStringSlice(obj, "top_additional_options"))
	state.CheckItemsAdditionalOptions = strListValue(ctx, getStringSlice(obj, "check_items_additional_options"))
	state.ReplyItemsAdditionalOptions = strListValue(ctx, getStringSlice(obj, "reply_items_additional_options"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *freeradiusMACResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan freeradiusMACModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	mac := plan.MAC.ValueString()
	if mac == "" {
		mac = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, freeradiusMACPlural, "mac", mac)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update FreeRADIUS MAC entry", err.Error())
		} else {
			resp.Diagnostics.AddError("FreeRADIUS MAC entry not found", "MAC "+mac+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, freeradiusMACSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update FreeRADIUS MAC entry", err.Error())
		return
	}
	plan.ID = types.StringValue(mac)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *freeradiusMACResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state freeradiusMACModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	mac := state.MAC.ValueString()
	if mac == "" {
		mac = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, freeradiusMACPlural, "mac", mac)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete FreeRADIUS MAC entry", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, freeradiusMACSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete FreeRADIUS MAC entry", err.Error())
	}
}

func (r *freeradiusMACResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_freeradius_user
// ---------------------------------------------------------------------------

type freeradiusUserResource struct{ client *client.Client }

var _ resource.Resource = (*freeradiusUserResource)(nil)
var _ resource.ResourceWithConfigure = (*freeradiusUserResource)(nil)
var _ resource.ResourceWithImportState = (*freeradiusUserResource)(nil)

const (
	freeradiusUserPlural   = "/api/v2/services/freeradius/users"
	freeradiusUserSingular = "/api/v2/services/freeradius/user"
)

type freeradiusUserModel struct {
	ID                          types.String `tfsdk:"id"`
	Username                    types.String `tfsdk:"username"`
	Password                    types.String `tfsdk:"password"`
	PasswordEncryption          types.String `tfsdk:"password_encryption"`
	MotpEnable                  types.Bool   `tfsdk:"motp_enable"`
	MotpAuthmethod              types.String `tfsdk:"motp_authmethod"`
	MotpSecret                  types.String `tfsdk:"motp_secret"`
	MotpPin                     types.String `tfsdk:"motp_pin"`
	MotpOffset                  types.Int64  `tfsdk:"motp_offset"`
	Description                 types.String `tfsdk:"description"`
	FramedIPAddress             types.String `tfsdk:"framed_ip_address"`
	FramedIPNetmask             types.String `tfsdk:"framed_ip_netmask"`
	FramedRoute                 types.String `tfsdk:"framed_route"`
	FramedIPv6Address           types.String `tfsdk:"framed_ipv6_address"`
	FramedIPv6Route             types.String `tfsdk:"framed_ipv6_route"`
	VLANID                      types.String `tfsdk:"vlan_id"`
	WisprRedirectionURL         types.String `tfsdk:"wispr_redirection_url"`
	SimultaneousConnect         types.Int64  `tfsdk:"simultaneous_connect"`
	Expiration                  types.String `tfsdk:"expiration"`
	SessionTimeout              types.Int64  `tfsdk:"session_timeout"`
	LoginTime                   types.String `tfsdk:"login_time"`
	AmountOfTime                types.Int64  `tfsdk:"amount_of_time"`
	PointOfTime                 types.String `tfsdk:"point_of_time"`
	MaxTotalOctets              types.Int64  `tfsdk:"max_total_octets"`
	MaxTotalOctetsTimeRange     types.String `tfsdk:"max_total_octets_time_range"`
	MaxBandwidthDown            types.Int64  `tfsdk:"max_bandwidth_down"`
	MaxBandwidthUp              types.Int64  `tfsdk:"max_bandwidth_up"`
	AcctInterimInterval         types.Int64  `tfsdk:"acct_interim_interval"`
	TopAdditionalOptions        types.List   `tfsdk:"top_additional_options"`
	CheckItemsAdditionalOptions types.List   `tfsdk:"check_items_additional_options"`
	ReplyItemsAdditionalOptions types.List   `tfsdk:"reply_items_additional_options"`
}

func NewFreeradiusUserResource() resource.Resource { return &freeradiusUserResource{} }

func (r *freeradiusUserResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_freeradius_user"
}
func (r *freeradiusUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *freeradiusUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a FreeRADIUS user.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username for this user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": requiredStringAttribute("The password for this username."),
			"password_encryption": enumAttribute(
				"The encryption method for the password.",
				"Cleartext-Password", "MD5-Password", "MD5-Password-hashed", "NT-Password-hashed",
			),
			"motp_enable": optionalBoolAttribute(
				"Enable or disable the use of Mobile One-Time Password (MOTP) for this user.",
			),
			"motp_authmethod": enumAttribute(
				"The authentication method for the Mobile One-Time Password (MOTP).",
				"motp", "googleauth",
			),
			"motp_secret": requiredStringAttribute("The secret for the Mobile One-Time Password (MOTP)."),
			"motp_pin":    requiredStringAttribute("The PIN for the Mobile One-Time Password (MOTP). It must be exactly 4 digits."),
			"motp_offset": optionalIntAttribute("The timezone offset for this user."),
			"description": optionalStringAttribute("A description for this entry."),
			"framed_ip_address": optionalStringAttribute(
				"Framed-IP-Address MUST be supported by NAS.",
			),
			"framed_ip_netmask": optionalStringAttribute(
				"Framed-IP-Netmask MUST be supported by NAS.",
			),
			"framed_route": optionalStringAttribute("Framed-Route must be supported by NAS."),
			"framed_ipv6_address": optionalStringAttribute(
				"Framed IPv6 address or prefix (e.g. 2001:db8:abab::5 or 2001:db8:abab::/64).",
			),
			"framed_ipv6_route": optionalStringAttribute("Framed-IPv6-Route must be supported by NAS."),
			"vlan_id": optionalStringAttribute(
				"The VLAN ID (integer from 1-4095) or VLAN name for this entry.",
			),
			"wispr_redirection_url": optionalStringAttribute(
				"The URL the user should be redirected to after successful login.",
			),
			"simultaneous_connect": optionalIntAttribute(
				"The maximum number of simultaneous connections with this entry. Leave null for no limit.",
			),
			"expiration": optionalStringAttribute(
				"The date when this account should expire. Required format: Mmm dd yyyy (e.g. Jan 01 2030).",
			),
			"session_timeout": optionalIntAttribute(
				"The time this entry has until relogin (in seconds).",
			),
			"login_time": optionalStringAttribute(
				"The time when this entry should have access. Empty for no time restriction.",
			),
			"amount_of_time": optionalIntAttribute(
				"The amount of time this entry is allowed (in minutes) within the configured time period.",
			),
			"point_of_time": enumAttribute(
				"The time period after which the 'Amount of Time' is reset.",
				"Daily", "Weekly", "Monthly", "Forever",
			),
			"max_total_octets": optionalIntAttribute(
				"The amount of download and upload traffic (summarized) in megabytes (MB) for this entry.",
			),
			"max_total_octets_time_range": enumAttribute(
				"The time period for the amount of download and upload traffic.",
				"daily", "weekly", "monthly", "forever",
			),
			"max_bandwidth_down": optionalIntAttribute(
				"The maximum bandwidth for download in kilobits per second (Kbit/s).",
			),
			"max_bandwidth_up": optionalIntAttribute(
				"The maximum bandwidth for upload in kilobits per second (Kbit/s).",
			),
			"acct_interim_interval": optionalIntAttribute(
				"The interval in seconds which should elapse between interim-updates.",
			),
			"top_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS attributes placed at the TOP of this entry.",
			},
			"check_items_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS check-item attributes for this entry.",
			},
			"reply_items_additional_options": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Additional RADIUS reply-item attributes for this entry.",
			},
		},
	}
}

func (r *freeradiusUserResource) payload(m freeradiusUserModel) map[string]any {
	p := map[string]any{}
	setString(p, "username", m.Username)
	setString(p, "password", m.Password)
	setString(p, "password_encryption", m.PasswordEncryption)
	setBool(p, "motp_enable", m.MotpEnable)
	setString(p, "motp_authmethod", m.MotpAuthmethod)
	setString(p, "motp_secret", m.MotpSecret)
	setString(p, "motp_pin", m.MotpPin)
	setInt(p, "motp_offset", m.MotpOffset)
	setString(p, "description", m.Description)
	setString(p, "framed_ip_address", m.FramedIPAddress)
	setString(p, "framed_ip_netmask", m.FramedIPNetmask)
	setString(p, "framed_route", m.FramedRoute)
	setString(p, "framed_ipv6_address", m.FramedIPv6Address)
	setString(p, "framed_ipv6_route", m.FramedIPv6Route)
	setString(p, "vlan_id", m.VLANID)
	setString(p, "wispr_redirection_url", m.WisprRedirectionURL)
	setInt(p, "simultaneous_connect", m.SimultaneousConnect)
	setString(p, "expiration", m.Expiration)
	setInt(p, "session_timeout", m.SessionTimeout)
	setString(p, "login_time", m.LoginTime)
	setInt(p, "amount_of_time", m.AmountOfTime)
	setString(p, "point_of_time", m.PointOfTime)
	setInt(p, "max_total_octets", m.MaxTotalOctets)
	setString(p, "max_total_octets_time_range", m.MaxTotalOctetsTimeRange)
	setInt(p, "max_bandwidth_down", m.MaxBandwidthDown)
	setInt(p, "max_bandwidth_up", m.MaxBandwidthUp)
	setInt(p, "acct_interim_interval", m.AcctInterimInterval)
	setStringList(p, "top_additional_options", m.TopAdditionalOptions)
	setStringList(p, "check_items_additional_options", m.CheckItemsAdditionalOptions)
	setStringList(p, "reply_items_additional_options", m.ReplyItemsAdditionalOptions)
	return p
}

func (r *freeradiusUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan freeradiusUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, freeradiusUserSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create FreeRADIUS user", err.Error())
		return
	}
	plan.ID = plan.Username
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *freeradiusUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state freeradiusUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	username := state.Username.ValueString()
	if username == "" {
		username = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, freeradiusUserPlural, "username", username)
	if err != nil {
		resp.Diagnostics.AddError("failed to read FreeRADIUS user", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(username)
	state.Username = types.StringValue(username)
	state.Password = strValue(getString(obj, "password"))
	state.PasswordEncryption = strValue(getString(obj, "password_encryption"))
	state.MotpEnable = boolValue(getBool(obj, "motp_enable"))
	state.MotpAuthmethod = strValue(getString(obj, "motp_authmethod"))
	state.MotpSecret = strValue(getString(obj, "motp_secret"))
	state.MotpPin = strValue(getString(obj, "motp_pin"))
	state.MotpOffset = intValue(getInt(obj, "motp_offset"))
	state.Description = strValue(getString(obj, "description"))
	state.FramedIPAddress = strValue(getString(obj, "framed_ip_address"))
	state.FramedIPNetmask = strValue(getString(obj, "framed_ip_netmask"))
	state.FramedRoute = strValue(getString(obj, "framed_route"))
	state.FramedIPv6Address = strValue(getString(obj, "framed_ipv6_address"))
	state.FramedIPv6Route = strValue(getString(obj, "framed_ipv6_route"))
	state.VLANID = strValue(getString(obj, "vlan_id"))
	state.WisprRedirectionURL = strValue(getString(obj, "wispr_redirection_url"))
	state.SimultaneousConnect = intValue(getInt(obj, "simultaneous_connect"))
	state.Expiration = strValue(getString(obj, "expiration"))
	state.SessionTimeout = intValue(getInt(obj, "session_timeout"))
	state.LoginTime = strValue(getString(obj, "login_time"))
	state.AmountOfTime = intValue(getInt(obj, "amount_of_time"))
	state.PointOfTime = strValue(getString(obj, "point_of_time"))
	state.MaxTotalOctets = intValue(getInt(obj, "max_total_octets"))
	state.MaxTotalOctetsTimeRange = strValue(getString(obj, "max_total_octets_time_range"))
	state.MaxBandwidthDown = intValue(getInt(obj, "max_bandwidth_down"))
	state.MaxBandwidthUp = intValue(getInt(obj, "max_bandwidth_up"))
	state.AcctInterimInterval = intValue(getInt(obj, "acct_interim_interval"))
	state.TopAdditionalOptions = strListValue(ctx, getStringSlice(obj, "top_additional_options"))
	state.CheckItemsAdditionalOptions = strListValue(ctx, getStringSlice(obj, "check_items_additional_options"))
	state.ReplyItemsAdditionalOptions = strListValue(ctx, getStringSlice(obj, "reply_items_additional_options"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *freeradiusUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan freeradiusUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	username := plan.Username.ValueString()
	if username == "" {
		username = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, freeradiusUserPlural, "username", username)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update FreeRADIUS user", err.Error())
		} else {
			resp.Diagnostics.AddError("FreeRADIUS user not found", "user "+username+" no longer exists")
		}
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, freeradiusUserSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update FreeRADIUS user", err.Error())
		return
	}
	plan.ID = types.StringValue(username)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *freeradiusUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state freeradiusUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	username := state.Username.ValueString()
	if username == "" {
		username = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, freeradiusUserPlural, "username", username)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete FreeRADIUS user", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, freeradiusUserSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete FreeRADIUS user", err.Error())
	}
}

func (r *freeradiusUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
