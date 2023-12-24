package pfsense

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceInterface() *resource[pfsenseapi.InterfaceRequest, pfsenseapi.Interface, string] {
	r := &resource[pfsenseapi.InterfaceRequest, pfsenseapi.Interface, string]{
		name:        "pfsense_interface",
		description: "Interface",
		delete: func(ctx context.Context, client *pfsenseapi.Client, _ string, id string) error {
			return client.Interface.DeleteInterface(ctx, id)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.Interface, error) {
			return client.Interface.ListInterfaces(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, id string, request *pfsenseapi.InterfaceRequest) (*pfsenseapi.Interface, error) {
			request.Apply = true
			return client.Interface.UpdateInterface(ctx, id, *request)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.InterfaceRequest) (*pfsenseapi.Interface, error) {
			request.Apply = true
			return client.Interface.CreateInterface(ctx, *request)
		},
		properties: map[string]*resourceProperty[pfsenseapi.InterfaceRequest, pfsenseapi.Interface]{
			"adv_dhcp_config_advanced": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Enable the IPv4 DHCP advanced configuration options. This enables the DHCP options: `adv_dhcp_send_options`, `adv_dhcp_request_options`, `adv_dhcp_required_options`, `adv_dhcp_option_modifiers`. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpConfigAdvanced = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpConfigAdvanced, nil
				},
			},
			"adv_dhcp_config_file_override": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Enable local DHCP configuration file override. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpConfigFileOverride = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpConfigFileOverride, nil
				},
			},
			"adv_dhcp_config_file_override_file": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set the custom DHCP configuration file's absolute path. This file must exist beforehand. This parameter is only available when `type` is set to `dhcp` and `adv_dhcp_config_file_override` is set to `true`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpConfigFileOverrideFile = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpConfigFileOverrideFile, nil
				},
			},
			"adv_dhcp_option_modifiers": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set a custom IPv4 option modifier. This parameter is only available when `type` is set to `dhcp` and `adv_dhcp_config_advanced` is set to `true`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpOptionModifiers = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpOptionModifiers, nil
				},
			},
			"adv_dhcp_pt_backoff_cutoff": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Set the IPv4 DHCP protocol backoff cutoff interval. Must be numeric value greater than 1. This parameter is only available when `type` is set to `dhcp`.",
					ValidateFunc: validation.IntAtLeast(1),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtBackoffCutoff = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtBackoffCutoff, nil
				},
			},
			"adv_dhcp_pt_initial_interval": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Set the IPv4 DHCP protocol initial interval. Must be numeric value greater than 1. This parameter is only available when `type` is set to `dhcp`.",
					ValidateFunc: validation.IntAtLeast(1),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtInitialInterval = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtInitialInterval, nil
				},
			},
			"adv_dhcp_pt_reboot": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "Set the IPv4 DHCP protocol reboot interval. Must be numeric value greater than 1. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtReboot = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtReboot, nil
				},
			},
			"adv_dhcp_pt_retry": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "Set the IPv4 DHCP protocol retry interval. Must be numeric value greater than 1. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtRetry = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtRetry, nil
				},
			},
			"adv_dhcp_pt_select_timeout": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "Set the IPv4 DHCP protocol select timeout interval. Must be numeric value greater than 0. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtSelectTimeout = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtSelectTimeout, nil
				},
			},
			"adv_dhcp_pt_timeout": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntAtLeast(1),
					Description:  "Set the IPv4 DHCP protocol timeout interval. Must be numeric value greater than 1. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpPtTimeout = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpPtTimeout, nil
				},
			},
			"adv_dhcp_request_options": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set a custom IPv4 request option. This parameter is only available when `type` is set to `dhcp` and `adv_dhcp_config_advanced` is set to `true`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpRequestOptions = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpRequestOptions, nil
				},
			},

			"adv_dhcp_required_options": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set a custom IPv4 required option. This parameter is only available when `type` is set to `dhcp` and `adv_dhcp_config_advanced` is set to `true`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpRequiredOptions = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpRequiredOptions, nil
				},
			},
			"adv_dhcp_send_options": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set a custom IPv4 send option. This parameter is only available when `type` is set to `dhcp` and `adv_dhcp_config_advanced` is set to `true`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AdvDhcpSendOptions = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AdvDhcpSendOptions, nil
				},
			},
			"alias_address": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsIPv4Address,
					Description:  "Set the IPv4 DHCP address alias. The value in this field is used as a fixed alias IPv4 address by the DHCP  client. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AliasAddress = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AliasAddress, nil
				},
			},
			"alias_subnet": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Set the IPv4 DHCP address aliases subnet. This parameter is only available when `type` is set to `dhcp`.",
					ValidateFunc: validation.IntBetween(1, 32),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.AliasSubnet = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.AliasSubnet, nil
				},
			},
			"block_bogons": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Block bogon networks from routing over this interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Blockbogons = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Blockbogons, nil
				},
			},
			"block_private": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Block RFC1918 traffic from routing over this interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Blockpriv = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Blockpriv, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Description: "Descriptive name for the new interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Descr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Descr, nil
				},
			},
			"dhcp_cv_pt": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Set the DHCP VLAN priority. This parameter is only available when `type` is set to `dhcp` and `dhcpvlanenable` is set to `true`.",
					ValidateFunc: validation.IntBetween(0, 7),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Dhcpcvpt = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Dhcpcvpt, nil
				},
			},
			"dhcp_hostname": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: dnsValidator,
					Description:  "Assign IPv4 DHCP hostname. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Dhcphostname = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Dhcphostname, nil
				},
			},
			"dhcp_reject_from": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Assign IPv4 DHCP rejected servers by IP. This parameter is only available when `type` is set to `dhcp`.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsIPv4Address,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					result, err := interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					req.Dhcprejectfrom = result
					return nil
				},
				getFromResponse: func(res *pfsenseapi.Interface) (interface{}, error) {
					return splitIntoArray(res.Dhcprejectfrom, ","), nil
				},
			},
			"dhcp_vlan_enable": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Enable DHCP VLAN prioritization. This parameter is only available when `type` is set to `dhcp`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Dhcpvlanenable = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Dhcpvlanenable, nil
				},
			},
			"enable": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
					Description: "Enable interface upon creation.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Enable = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Enable, nil
				},
			},
			"gateway": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Name of upstream IPv4 gateway for this interface. This is only necessary on WAN/UPLINK interfaces. This parameter is only available when `type` is set to `staticv4`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Gateway = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Gateway, nil
				},
			},
			"gateway_6_rd": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Set the 6RD interface IPv4 gateway address. This parameter is only required when `type6` is set to `6rd`",
					ValidateFunc: validation.IsIPv6Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Gateway6Rd = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Gateway6Rd, nil
				},
			},
			"gateway_v6": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Name of upstream IPv6 gateway for this interface. This is only necessary for WAN/UPLINK interfaces. This parameter is only available when `type6` is set to `staticv6`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Gatewayv6 = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Gatewayv6, nil
				},
			},
			"if": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Real interface ID to configure.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.If = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.If, nil
				},
			},
			"ip_address": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Interface's static IPv4 address. Required if `type` is set to `staticv4`.",
					ValidateFunc: validation.IsIPv4Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Ipaddr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Ipaddr, nil
				},
			},
			"ip_address_v6": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "Interface's static IPv6 address. Required if `type6` is set to `staticv6`.",
					ValidateFunc: validation.IsIPv6Address,
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Ipaddrv6 = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Ipaddrv6, nil
				},
			},
			"ip_v6_use_v4_iface": {
				schema: &schema.Schema{
					Type:        schema.TypeBool,
					Optional:    true,
					Description: "Allow IPv6 to use IPv4 uplink connection.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Ipv6Usev4Iface = d.Get(name).(bool)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Ipv6Usev4Iface, nil
				},
			},
			"media": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Speed/duplex setting for this interface. Options are dependent on physical interface capabilities.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Media = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Media, nil
				},
			},
			"mss": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "MSS for this interface.",
					ValidateFunc: validation.IntBetween(576, 65535),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Mss = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Mss, nil
				},
			},
			"mtu": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "MTU for this interface. If a VLAN interface, this value must be greater than parent.",
					ValidateFunc: validation.IntBetween(1280, 8192),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Mtu = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Mtu, nil
				},
			},
			"prefix_v6_rd": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set the 6RD IPv6 prefix assigned by the ISP. This parameter is only required when `type6` is set to `6rd`",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Prefix6Rd = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Prefix6Rd, nil
				},
			},
			"prefix_6_rd_v4_plen": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Set the 6RD IPv4 prefix length. This is typically assigned by the ISP. This parameter is only available when `type6` is set to `6rd`.",
					ValidateFunc: validation.IntBetween(0, 32),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Prefix6RdV4Plen = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Prefix6RdV4Plen, nil
				},
			},
			"spoof_mac": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validation.IsMACAddress,
					Description:  "Custom MAC address to assign to the interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Spoofmac = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Spoofmac, nil
				},
			},
			"subnet": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					Description:  "Interface's static IPv4 address's subnet bitmask. Required if `type` is set to `staticv4`.",
					ValidateFunc: validation.IntBetween(1, 32),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					i := d.Get(name).(int)
					req.Subnet = &i
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Subnet, nil
				},
			},
			"subnet_v6": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Interface's static IPv6 address's subnet bitmask. Required if `type6` is set to `staticv6`.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Subnetv6 = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Subnetv6, nil
				},
			},
			"track_v6_interface": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set the Track6 dynamic IPv6 interface. This must be a dynamically configured IPv6 interface. You may specify either the interface's descriptive name, the pfSense ID (wan, lan, optx), or the physical interface id (e.g. igb0). This parameter is only required with `type6` is set to `track6`",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Track6Interface = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Track6Interface, nil
				},
			},
			"track_v6_prefix_id_hex": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Set the IPv6 prefix ID. The value in this field is the (Delegated) IPv6 prefix ID. This determines the configurable network ID based on the dynamic IPv6 connection. The default value is 0. This parameter is only available when `type6` is set to",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Track6PrefixIdHex = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Track6PrefixIdHex, nil
				},
			},
			"type": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "IPv4 configuration type.",
					ValidateFunc: validation.StringInSlice([]string{"staticv4", "dhcp"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Type = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					if req.Ipaddr == "dhcp" {
						return req.Ipaddr, nil
					} else if req.Ipaddr != "" {
						return "staticv4", nil
					}

					return "", nil
				},
			},
			"type_v6": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					Description:  "IPv6 configuration type.",
					ValidateFunc: validation.StringInSlice([]string{"staticv6", "dhcp6", "slaac", "6rd", "track6", "6to4"}, false),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.InterfaceRequest) error {
					req.Type6 = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.Interface) (interface{}, error) {
					return req.Type6, nil
				},
			},
		},
	}

	r.getId = func(ctx context.Context, client *pfsenseapi.Client, i *pfsenseapi.Interface) (string, error) {
		ifaces, err := r.list(ctx, client, "")

		if err != nil {
			return "", err
		}

		for _, iface := range ifaces {
			if iface.If == i.If {
				return iface.Name, nil
			}
		}

		return "", fmt.Errorf("Unable to find interface with If %s after creation", i.If)
	}

	return r
}
