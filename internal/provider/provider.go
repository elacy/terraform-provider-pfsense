// Package provider implements the pfsense Terraform provider backed by the
// pfSense REST API v2 (pfrest/pfSense-pkg-RESTAPI).
package provider

import (
	"context"
	"time"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure pfsenseProvider satisfies the provider.Provider interface.
var _ provider.Provider = (*pfsenseProvider)(nil)

// pfsenseProvider is the framework provider for pfSense REST API v2.
type pfsenseProvider struct {
	version string
}

// pfsenseProviderModel maps the provider schema to Go types.
type pfsenseProviderModel struct {
	URL           types.String `tfsdk:"url"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	APIKey        types.String `tfsdk:"api_key"`
	UseJWT        types.Bool   `tfsdk:"use_jwt"`
	SkipTLSVerify types.Bool   `tfsdk:"skip_tls_verify"`
	Timeout       types.Int64  `tfsdk:"timeout"`
}

// New returns a new pfsense provider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &pfsenseProvider{version: version}
	}
}

// Metadata returns the provider type name.
func (p *pfsenseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pfsense"
	resp.Version = p.version
}

// Schema returns the provider schema.
func (p *pfsenseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The base URL of the pfSense instance (e.g. https://192.168.1.1).",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Local database username for Basic or JWT authentication.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Local database password for Basic or JWT authentication.",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "REST API key used for X-API-Key authentication. Takes precedence over username/password.",
			},
			"use_jwt": schema.BoolAttribute{
				Optional:    true,
				Description: "Exchange username/password for a JSON Web Token and use bearer auth. Ignored unless `username` is set.",
			},
			"skip_tls_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification. Required for pfSense boxes using self-signed certificates.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "HTTP request timeout in seconds. Defaults to 30.",
			},
		},
	}
}

// Configure builds the API client from the provider configuration.
func (p *pfsenseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config pfsenseProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.URL.IsNull() || config.URL.ValueString() == "" {
		resp.Diagnostics.AddError("missing URL", "The `url` attribute is required.")
		return
	}

	c, err := client.New(client.Config{
		URL:           config.URL.ValueString(),
		Username:      config.Username.ValueString(),
		Password:      config.Password.ValueString(),
		APIKey:        config.APIKey.ValueString(),
		UseJWT:        config.UseJWT.ValueBool(),
		SkipTLSVerify: config.SkipTLSVerify.ValueBool(),
		Timeout:       seconds(config.Timeout),
	})
	if err != nil {
		resp.Diagnostics.AddError("unable to create pfSense client", err.Error())
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

// Resources returns the resources supported by the provider.
func (p *pfsenseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFirewallAliasResource,
		NewFirewallRuleResource,
		NewFirewallScheduleResource,
		NewInterfaceVLANResource,
		NewRoutingGatewayResource,
		NewRoutingGatewayGroupResource,
		NewRoutingStaticRouteResource,
		NewDHCPServerResource,
		NewDNSResolverHostOverrideResource,
		NewSystemUserResource,
		NewSystemGroupResource,
		NewSystemCAResource,
		NewSystemCertificateResource,
		NewSystemTunableResource,
		NewSystemHostnameResource,
		NewSystemDNSResource,
		NewSystemTimezoneResource,
		NewServicesNTPSettingsResource,
		NewServicesCronJobResource,
		// firewall NAT
		NewFirewallNATPortForwardResource,
		NewFirewallNATOneToOneResource,
		NewFirewallNATOutboundResource,
		// traffic shaper / limiter / queues / virtual IPs
		NewVirtualIPResource,
		NewTrafficShaperResource,
		NewTrafficShaperLimiterResource,
		NewTrafficShaperQueueResource,
		NewTrafficShaperLimiterQueueResource,
		// interfaces
		NewNetworkInterfaceResource,
		NewInterfaceBridgeResource,
		NewInterfaceGREResource,
		NewInterfaceLAGGResource,
		NewInterfaceGroupResource,
		// services extras
		NewDHCPServerStaticMappingResource,
		NewDHCPServerAddressPoolResource,
		NewDHCPServerCustomOptionResource,
		NewDNSResolverDomainOverrideResource,
		NewDNSResolverHostOverrideAliasResource,
		NewDNSForwarderHostOverrideResource,
		NewDNSForwarderHostOverrideAliasResource,
		NewNTPTimeServerResource,
		NewServiceWatchdogResource,
		// BIND / FreeRADIUS
		NewBindAccessListResource,
		NewBindViewResource,
		NewBindZoneResource,
		NewFreeradiusMACResource,
		NewFreeradiusUserResource,
	}
}

// DataSources returns the data sources supported by the provider.
func (p *pfsenseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFirewallAliasesDataSource,
		NewInterfacesDataSource,
		NewRoutingGatewaysDataSource,
		NewSystemCertificatesDataSource,
	}
}

func seconds(d types.Int64) time.Duration {
	if d.IsNull() || d.IsUnknown() {
		return 0
	}
	return time.Duration(d.ValueInt64()) * time.Second
}
