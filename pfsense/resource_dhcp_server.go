package pfsense

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceDHCPServer() *resource[pfsenseapi.DHCPServerConfigurationRequest, pfsenseapi.DHCPServerConfiguration, string] {
	return &resource[pfsenseapi.DHCPServerConfigurationRequest, pfsenseapi.DHCPServerConfiguration, string]{
		name:        "pfsense_dhcp_server",
		description: "IPv4 DHCP Server Configuration",
		disable: func(request *pfsenseapi.DHCPServerConfigurationRequest) error {
			request.Enable = false
			return nil
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.DHCPServerConfiguration, error) {
			return client.DHCP.ListServerConfigurations(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, id string, request *pfsenseapi.DHCPServerConfigurationRequest) (*pfsenseapi.DHCPServerConfiguration, error) {
			return client.DHCP.UpdateServerConfiguration(ctx, *request)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.DHCPServerConfigurationRequest) (*pfsenseapi.DHCPServerConfiguration, error) {
			return client.DHCP.UpdateServerConfiguration(ctx, *request)
		},
		properties: map[string]*resourceProperty[pfsenseapi.DHCPServerConfigurationRequest, pfsenseapi.DHCPServerConfiguration]{
			"default_lease_time": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Default DHCP lease time. This must be a value of `60` or greater and must be less than `maxleasetime`. This field can be unset to the system default by passing in an empty string.",
					ValidateFunc: validation.IntAtLeast(60),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.DefaultLeaseTime = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.DefaultLeaseTime, nil
				},
			},
			"deny_unknown": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Deny unknown MAC addresses. If true, you must specify  MAC addresses in the `mac_allow` field or add a static DHCP entry to receive DHCP requests.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.DenyUnknown = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.DenyUnknown, nil
				},
			},
			"dns_server": {
				schema: &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 4,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsIPv4Address,
					},
					Description: "DNS servers to hand out in DHCP leases.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.DNSServer = result
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.DNSServer, nil
				},
			},
			"domain": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: dnsValidator,
					Description:  "Domain name to include in DHCP leases. This must be a valid domain name or an empty string to assume the system default.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.Domain = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.Domain, nil
				},
			},
			"domain_search_list": {
				schema: &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: dnsValidator,
					},
					Description: "Search domains to include in DHCP leases. Each entry must be a valid domain name.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}
					req.DomainSearchList = result
					return nil
				},
				getFromResponse: func(res *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return strings.Split(res.DomainSearchList, ";"), nil
				},
			},
			"enable": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Enable the DHCP server for this interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.Enable = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.Enable, nil
				},
			},
			"gateway": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsIPv4Address,
					Description:  "Gateway to hand out in DHCP leases. This value must be a valid IPv4 address within the interface's subnet. This field can be unset to the system default by passing in an empty string.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.Gateway = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.Gateway, nil
				},
			},
			"ignore_bootp": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Ignore BOOTP requests.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.IgnoreBootP = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.IgnoreBootP, nil
				},
			},
			"interface": {
				idProperty: true,
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Interface of DHCP server configuration to update. You may specify either the interface's descriptive name, the pfSense ID (wan, lan, optx), or the real interface ID (e.g. igb0). This interface must host a static IPv4 subnet that has more than one available within the subnet.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.Interface = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.Interface, nil
				},
			},
			"mac_allow_list": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "MAC addresses allowed to register DHCP leases.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsMACAddress,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}
					req.MacAllow = result
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.MacAllow, nil
				},
			},
			"mac_deny_list": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "MAC addresses denied from registering DHCP leases.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsMACAddress,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.MacDeny = result
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.MacDeny, nil
				},
			},
			"max_lease_time": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Maximum DHCP lease time. This must be a value of `60` or greater and must be greater than `defaultleasetime`. This field can be unset to the system default by passing in an empty string.",
					ValidateFunc: validation.IntAtLeast(60),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.MaxLeaseTime = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					return req.MaxLeaseTime, nil
				},
			},
			"range_from": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "DHCP pool's starting IPv4 address. This must be an available address within the interface's subnet and be less than the `range_to` value. This field is required if no `range_from` value has been set previously.",
					ValidateFunc: validation.IsIPv4Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.RangeFrom = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					if req.Range == nil {
						return nil, nil
					}

					return req.Range.From, nil
				},
			},
			"range_to": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsIPv4Address,
					Description:  "DHCP pool's ending IPv4 address. This must be an available address within the interface's subnet and be greater than the `range_from` value. This field is required if no `range_to` has been set previously.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPServerConfigurationRequest) error {
					req.RangeTo = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPServerConfiguration) (interface{}, error) {
					if req.Range == nil {
						return nil, nil
					}

					return req.Range.To, nil
				},
			},
		},
	}
}
