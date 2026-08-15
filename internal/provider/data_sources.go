package provider

import (
	"context"
	"encoding/json"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_firewall_aliases (data source)
// ---------------------------------------------------------------------------

type firewallAliasesDataSource struct{ client *client.Client }

var _ datasource.DataSource = (*firewallAliasesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*firewallAliasesDataSource)(nil)

type firewallAliasesDataSourceModel struct {
	Name    types.String            `tfsdk:"name"`
	Aliases []firewallAliasListItem `tfsdk:"aliases"`
}

// firewallAliasListItem is the data-source representation of an alias (a subset
// of the resource model).
type firewallAliasListItem struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Descr   types.String `tfsdk:"descr"`
	Address types.List   `tfsdk:"address"`
}

func NewFirewallAliasesDataSource() datasource.DataSource { return &firewallAliasesDataSource{} }

func (d *firewallAliasesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pfsense_firewall_aliases"
}
func (d *firewallAliasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(ctx, req, resp)
}
func (d *firewallAliasesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists firewall aliases.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{Optional: true, Description: "Filter aliases by name (exact match)."},
			"aliases": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":    schema.StringAttribute{Computed: true},
						"type":    schema.StringAttribute{Computed: true},
						"descr":   schema.StringAttribute{Computed: true},
						"address": schema.ListAttribute{Computed: true, ElementType: types.StringType},
					},
				},
			},
		},
	}
}

func (d *firewallAliasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config firewallAliasesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	var q client.Query
	if !config.Name.IsNull() && config.Name.ValueString() != "" {
		q = client.Query{}.Set("name__exact", config.Name.ValueString())
	}
	items, err := d.client.List(ctx, firewallAliasPlural, q)
	if err != nil {
		resp.Diagnostics.AddError("failed to list aliases", err.Error())
		return
	}
	aliases := make([]firewallAliasListItem, 0, len(items))
	for _, item := range items {
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			continue
		}
		aliases = append(aliases, firewallAliasListItem{
			Name:    strValue(getString(obj, "name")),
			Type:    strValue(getString(obj, "type")),
			Descr:   strValue(getString(obj, "descr")),
			Address: strListValue(ctx, getStringSlice(obj, "address")),
		})
	}
	config.Aliases = aliases
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ---------------------------------------------------------------------------
// pfsense_interfaces (data source)
// ---------------------------------------------------------------------------

type interfacesDataSource struct{ client *client.Client }

var _ datasource.DataSource = (*interfacesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*interfacesDataSource)(nil)

type interfaceItem struct {
	If     types.String `tfsdk:"if"`
	Descr  types.String `tfsdk:"descr"`
	Enable types.Bool   `tfsdk:"enable"`
	IPAddr types.String `tfsdk:"ipaddr"`
	Subnet types.Int64  `tfsdk:"subnet"`
}

type interfacesDataSourceModel struct {
	Interfaces []interfaceItem `tfsdk:"interfaces"`
}

func NewInterfacesDataSource() datasource.DataSource { return &interfacesDataSource{} }

func (d *interfacesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pfsense_interfaces"
}
func (d *interfacesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(ctx, req, resp)
}
func (d *interfacesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists network interfaces.",
		Attributes: map[string]schema.Attribute{
			"interfaces": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"if":     schema.StringAttribute{Computed: true},
						"descr":  schema.StringAttribute{Computed: true},
						"enable": schema.BoolAttribute{Computed: true},
						"ipaddr": schema.StringAttribute{Computed: true},
						"subnet": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *interfacesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config interfacesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	items, err := d.client.List(ctx, "/api/v2/interfaces", nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to list interfaces", err.Error())
		return
	}
	out := make([]interfaceItem, 0, len(items))
	for _, item := range items {
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			continue
		}
		out = append(out, interfaceItem{
			If:     strValue(getString(obj, "if")),
			Descr:  strValue(getString(obj, "descr")),
			Enable: boolValue(getBool(obj, "enable")),
			IPAddr: strValue(getString(obj, "ipaddr")),
			Subnet: intValue(getInt(obj, "subnet")),
		})
	}
	config.Interfaces = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ---------------------------------------------------------------------------
// pfsense_routing_gateways (data source)
// ---------------------------------------------------------------------------

type routingGatewaysDataSource struct{ client *client.Client }

var _ datasource.DataSource = (*routingGatewaysDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*routingGatewaysDataSource)(nil)

type gatewayItem struct {
	Name      types.String `tfsdk:"name"`
	Gateway   types.String `tfsdk:"gateway"`
	Interface types.String `tfsdk:"interface"`
	Disabled  types.Bool   `tfsdk:"disabled"`
}

type routingGatewaysDataSourceModel struct {
	Gateways []gatewayItem `tfsdk:"gateways"`
}

func NewRoutingGatewaysDataSource() datasource.DataSource { return &routingGatewaysDataSource{} }

func (d *routingGatewaysDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pfsense_routing_gateways"
}
func (d *routingGatewaysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(ctx, req, resp)
}
func (d *routingGatewaysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists routing gateways.",
		Attributes: map[string]schema.Attribute{
			"gateways": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      schema.StringAttribute{Computed: true},
						"gateway":   schema.StringAttribute{Computed: true},
						"interface": schema.StringAttribute{Computed: true},
						"disabled":  schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *routingGatewaysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config routingGatewaysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	items, err := d.client.List(ctx, routingGatewayPlural, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to list gateways", err.Error())
		return
	}
	out := make([]gatewayItem, 0, len(items))
	for _, item := range items {
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			continue
		}
		out = append(out, gatewayItem{
			Name:      strValue(getString(obj, "name")),
			Gateway:   strValue(getString(obj, "gateway")),
			Interface: strValue(getString(obj, "interface")),
			Disabled:  boolValue(getBool(obj, "disabled")),
		})
	}
	config.Gateways = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// ---------------------------------------------------------------------------
// pfsense_system_certificates (data source)
// ---------------------------------------------------------------------------

type systemCertificatesDataSource struct{ client *client.Client }

var _ datasource.DataSource = (*systemCertificatesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*systemCertificatesDataSource)(nil)

type certificateItem struct {
	Descr      types.String `tfsdk:"descr"`
	RefID      types.String `tfsdk:"refid"`
	Type       types.String `tfsdk:"type"`
	ValidUntil types.String `tfsdk:"valid_until"`
}

type systemCertificatesDataSourceModel struct {
	Certificates []certificateItem `tfsdk:"certificates"`
}

func NewSystemCertificatesDataSource() datasource.DataSource { return &systemCertificatesDataSource{} }

func (d *systemCertificatesDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pfsense_system_certificates"
}
func (d *systemCertificatesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = dataSourceClient(ctx, req, resp)
}
func (d *systemCertificatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists certificates.",
		Attributes: map[string]schema.Attribute{
			"certificates": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"descr":       schema.StringAttribute{Computed: true},
						"refid":       schema.StringAttribute{Computed: true},
						"type":        schema.StringAttribute{Computed: true},
						"valid_until": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *systemCertificatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config systemCertificatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || d.client == nil {
		return
	}
	items, err := d.client.List(ctx, systemCertificatePlural, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to list certificates", err.Error())
		return
	}
	out := make([]certificateItem, 0, len(items))
	for _, item := range items {
		var obj map[string]any
		if json.Unmarshal(item, &obj) != nil {
			continue
		}
		out = append(out, certificateItem{
			Descr:      strValue(getString(obj, "descr")),
			RefID:      strValue(getString(obj, "refid")),
			Type:       strValue(getString(obj, "type")),
			ValidUntil: strValue(getString(obj, "valid_until")),
		})
	}
	config.Certificates = out
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
