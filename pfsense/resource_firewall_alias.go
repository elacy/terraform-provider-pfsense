package pfsense

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

const addressSplitter = " "
const detailSplitter = "||"

func resourceFirewallAlias() *resource[pfsenseapi.FirewallAliasRequest, pfsenseapi.FirewallAlias, string] {
	return &resource[pfsenseapi.FirewallAliasRequest, pfsenseapi.FirewallAlias, string]{
		name:        "pfsense_firewall_alias",
		description: "Firewall Alias",
		delete: func(ctx context.Context, client *pfsenseapi.Client, _ string, name string) error {
			return client.Firewall.DeleteAlias(ctx, name, true)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.FirewallAlias, error) {
			return client.Firewall.ListAliases(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, name string, request *pfsenseapi.FirewallAliasRequest) (*pfsenseapi.FirewallAlias, error) {
			return client.Firewall.UpdateAlias(ctx, name, *request, true)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.FirewallAliasRequest) (*pfsenseapi.FirewallAlias, error) {
			return client.Firewall.CreateAlias(ctx, *request, true)
		},
		properties: map[string]*resourceProperty[pfsenseapi.FirewallAliasRequest, pfsenseapi.FirewallAlias]{
			"name": {
				idProperty: true,
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringMatch(regexValidator(`^\w+$`), "Only alpha-numeric and underscore characters are allowed"),
					Description:  "Name of the new alias. Only alpha-numeric and underscore characters are allowed",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallAliasRequest) error {
					req.Name = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.FirewallAlias) (interface{}, error) {
					return req.Name, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description of alias.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallAliasRequest) error {
					req.Descr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.FirewallAlias) (interface{}, error) {
					return req.Descr, nil
				},
			},
			"type": {
				schema: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: validation.StringInSlice([]string{"host", "network", "port"}, false),
					Description:  "Type of alias.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallAliasRequest) error {
					req.Type = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.FirewallAlias) (interface{}, error) {
					return req.Type, nil
				},
			},
			"target": {
				schema: &schema.Schema{
					Type:     schema.TypeList,
					Required: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"address": {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "Host, network or port values to add to the alias.",
								ValidateFunc: validation.StringDoesNotContainAny(" "),
							},
							"description": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  "Description of the address",
								ValidateFunc: validation.StringDoesNotContainAny("|"),
							},
						},
					},
					Description: "Hosts, networks or port values to add to the alias.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.FirewallAliasRequest) error {
					targets := d.Get("target").([]interface{})

					addressStrings := make([]string, len(targets))
					detailStrings := make([]string, len(targets))

					for i, target := range targets {
						targetMap := target.(map[string]interface{})
						addressStrings[i] = targetMap["address"].(string)
						if detail, ok := targetMap["description"].(string); ok {
							detailStrings[i] = detail
						} else {
							detailStrings[i] = ""
						}
					}

					req.Address = addressStrings
					req.Detail = detailStrings

					return nil
				},
				getFromResponse: func(response *pfsenseapi.FirewallAlias) (interface{}, error) {
					var addresses []map[string]interface{}
					details := splitIntoArray(response.Detail, detailSplitter)

					for i, addr := range splitIntoArray(response.Address, addressSplitter) {
						addressData := map[string]interface{}{
							"address": addr,
						}
						if len(details) > i {
							addressData["description"] = details[i]
						}
						addresses = append(addresses, addressData)
					}

					return addresses, nil
				},
			},
		},
	}
}
