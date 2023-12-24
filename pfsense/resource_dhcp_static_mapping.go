package pfsense

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceDHCPStaticMapping() *resource[pfsenseapi.DHCPStaticMappingRequest, pfsenseapi.DHCPStaticMapping, string] {
	return &resource[pfsenseapi.DHCPStaticMappingRequest, pfsenseapi.DHCPStaticMapping, string]{
		name:        "pfsense_dhcp_static_mapping",
		description: "IPv4 DHCP Static Mapping ",
		delete: func(ctx context.Context, client *pfsenseapi.Client, iface string, macAddress string) error {
			return client.DHCP.DeleteStaticMapping(ctx, iface, macAddress)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, iface string) ([]*pfsenseapi.DHCPStaticMapping, error) {
			return client.DHCP.ListStaticMappings(ctx, iface)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, macAddress string, request *pfsenseapi.DHCPStaticMappingRequest) (*pfsenseapi.DHCPStaticMapping, error) {
			return client.DHCP.UpdateStaticMapping(ctx, macAddress, *request)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.DHCPStaticMappingRequest) (*pfsenseapi.DHCPStaticMapping, error) {
			return client.DHCP.CreateStaticMapping(ctx, *request)
		},
		properties: map[string]*resourceProperty[pfsenseapi.DHCPStaticMappingRequest, pfsenseapi.DHCPStaticMapping]{
			"interface": {
				partition: true,
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Description: "Interface to assign this static mapping to. You may specify either the interface's descriptive name, the pfSense interface ID (e.g. wan, lan, optx), or the real interface ID (e.g. igb0).",
					Required:    true,
					ForceNew:    true,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Interface = d.Get(name).(string)
					return nil
				},
			},
			"mac": {
				idProperty: true,
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					Description:  "MAC address of the host this mapping will apply to.",
					ValidateFunc: validation.IsMACAddress,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Mac = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Mac, nil
				},
			},
			"client_identifier": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set a client identifier.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Cid = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Cid, nil
				},
			},
			"ip_address": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "IPv4 address the MAC address will be assigned.",
					ValidateFunc: validation.IsIPv4Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Ipaddr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.IPaddr, nil
				},
			},
			"hostname": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Hostname for this host.",
					ValidateFunc: dnsValidator,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Hostname = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Hostname, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description for this mapping",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Descr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Descr, nil
				},
			},
			"gateway": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Gateway to assign this host. This value must be a valid IPv4 address within the interface's subnet.",
					ValidateFunc: validation.IsIPv4Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Gateway = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Gateway, nil
				},
			},
			"domain": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Domain for this host.",
					ValidateFunc: dnsValidator,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.Domain = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.Domain, nil
				},
			},
			"domain_search_list": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Search domains to assign to this host. Each value be a valid domain name.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: dnsValidator,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.DomainSearchList = result
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return splitIntoArray(req.DomainSearchList, ";"), nil
				},
			},
			"dns_servers": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "DNS servers to assign this client. Each value must be a valid IPv4 address.",
					MaxItems:    4,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsIPv4Address,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.DNSServer = result
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.DNSServers, nil
				},
			},
			"arp_table_static_entry": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Create a static ARP entry for this static mapping.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.DHCPStaticMappingRequest) error {
					req.ArpTableStaticEntry = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.DHCPStaticMapping) (interface{}, error) {
					return req.ArpTableStaticEntry, nil
				},
			},
		},
	}
}
