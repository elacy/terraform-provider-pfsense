package pfsense

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func splitDns(dns string) (string, string) {
	parts := strings.Split(dns, ".")
	return parts[0], strings.Join(parts[1:], ".")
}

func resourceUnboundHostOverride() *resource[pfsenseapi.UnboundHostOverride, pfsenseapi.UnboundHostOverride, string] {
	return &resource[pfsenseapi.UnboundHostOverride, pfsenseapi.UnboundHostOverride, string]{
		name:        "pfsense_unbound_host_override",
		description: "Unbound Host Override",
		delete: func(ctx context.Context, client *pfsenseapi.Client, _ string, dns string) error {
			host_name, domain_name := splitDns(dns)
			return client.Unbound.DeleteHostOverride(ctx, host_name, domain_name, true)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.UnboundHostOverride, error) {
			return client.Unbound.ListHostOverrides(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, _ string, request *pfsenseapi.UnboundHostOverride) (*pfsenseapi.UnboundHostOverride, error) {
			return client.Unbound.UpdateHostOverride(ctx, request, true)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.UnboundHostOverride) (*pfsenseapi.UnboundHostOverride, error) {
			return client.Unbound.CreateHostOverride(ctx, request, true)
		},
		properties: map[string]*resourceProperty[pfsenseapi.UnboundHostOverride, pfsenseapi.UnboundHostOverride]{
			"dns": {
				idProperty: true,
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: dnsValidator,
					Description:  "Hostname of the host override.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.UnboundHostOverride) error {
					req.Host, req.Domain = splitDns(d.Get(name).(string))
					return nil
				},
				getFromResponse: func(req *pfsenseapi.UnboundHostOverride) (interface{}, error) {
					return fmt.Sprintf("%s.%s", req.Host, req.Domain), nil
				},
			},
			"ip_addresses": {
				schema: &schema.Schema{
					Type:        schema.TypeList,
					Required:    true,
					MinItems:    1,
					Description: "IPv4 or IPv6 of the host override.",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validation.IsIPAddress,
					},
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.UnboundHostOverride) error {
					var err error
					req.IP, err = interfaceToStringArray(d.Get(name))

					if err != nil {
						return err
					}

					return nil
				},
				getFromResponse: func(req *pfsenseapi.UnboundHostOverride) (interface{}, error) {
					return req.IP, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description of the host override.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.UnboundHostOverride) error {
					req.Description = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.UnboundHostOverride) (interface{}, error) {
					return req.Description, nil
				},
			},
			"aliases": {
				schema: &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"host_name": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Hostname of the host override alias.",
							},
							"domain_name": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Domnain Name of the host override alias.",
							},
							"description": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Description of the host override alias.",
							},
						},
					},
					Description: "Host override aliases to associate with this host override. For more information on alias object fields, see documentation for /api/v1/services/dnsmasq/host_override/alias.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.UnboundHostOverride) error {
					aliases := d.Get(name).([]interface{})

					req.Aliases = &pfsenseapi.UnboundAliasesList{
						Items: make([]*pfsenseapi.UnboundHostOverrideAlias, len(aliases)),
					}

					for i, a := range aliases {
						m := a.(map[string]interface{})

						req.Aliases.Items[i] = &pfsenseapi.UnboundHostOverrideAlias{
							Host:        m["host_name"].(string),
							Description: m["description"].(string),
							Domain:      m["domain_name"].(string),
						}
					}

					return nil
				},
				getFromResponse: func(response *pfsenseapi.UnboundHostOverride) (interface{}, error) {
					if response.Aliases == nil || response.Aliases.Items == nil || len(response.Aliases.Items) == 0 {
						return nil, nil
					}

					aliases := make([]interface{}, len(response.Aliases.Items))

					for i, alias := range response.Aliases.Items {
						aliases[i] = map[string]interface{}{
							"host_name":   alias.Host,
							"domain_name": alias.Domain,
							"description": alias.Description,
						}
					}

					return aliases, nil
				},
			},
		},
	}
}
