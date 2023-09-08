package pfsense

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

const addressSplitter = " "
const detailSplitter = "||"

func resourceFirewallAlias() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceFirewallAliasCreate,
		ReadContext:   resourceFirewallAliasRead,
		UpdateContext: resourceFirewallAliasUpdate,
		DeleteContext: resourceFirewallAliasDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true, // if the name cannot be changed after creation
			},
			"descr": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"host", "network", "port", "url", "url_ports", "urltable", "urltable_ports"}, false),
			},
			"addresses": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type:     schema.TypeString,
							Required: true,
						},
						"detail": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

// getAliasFromResource is used to convert the Terraform Resource data provided by the user into a Request that the API will accept.
func getAliasFromResource(d *schema.ResourceData) pfsenseapi.FirewallAliasRequest {
	addresses := d.Get("addresses").([]interface{})
	addressStrings := make([]string, len(addresses))
	detailStrings := make([]string, len(addresses))

	for i, address := range addresses {
		addrMap := address.(map[string]interface{})
		addressStrings[i] = addrMap["address"].(string)
		if detail, ok := addrMap["detail"].(string); ok {
			detailStrings[i] = detail
		} else {
			detailStrings[i] = ""
		}
	}

	return pfsenseapi.FirewallAliasRequest{
		Address: addressStrings,
		Descr:   d.Get("descr").(string),
		Detail:  detailStrings,
		Name:    d.Get("name").(string),
		Type:    d.Get("type").(string),
	}
}

func resourceFirewallAliasCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*pfsenseapi.Client)
	request := getAliasFromResource(d)

	if err := client.Firewall.CreateAlias(ctx, request, d.Get("apply_immediately").(bool)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(request.Name) // Aliases are unique by name

	return resourceFirewallAliasRead(ctx, d, m)
}

func resourceFirewallAliasDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*pfsenseapi.Client)
	if err := client.Firewall.DeleteAlias(ctx, d.Id(), d.Get("apply_immediately").(bool)); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("") // Remove the resource from state
	return nil
}

func resourceFirewallAliasRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := m.(*pfsenseapi.Client)

	aliases, err := client.Firewall.ListAliases(ctx)

	if err != nil {
		return diag.FromErr(err)
	}

	var alias *pfsenseapi.FirewallAlias

	for _, a := range aliases {
		if a.Name == d.Id() {
			alias = a
			break
		}
	}

	if alias == nil {
		return diag.Errorf("Alias named '%s' not found", d.Id())
	}

	// Set the simple fields directly
	if err := d.Set("name", alias.Name); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("descr", alias.Descr); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}
	if err := d.Set("type", alias.Type); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	var addresses []map[string]interface{}
	details := strings.Split(alias.Detail, detailSplitter)

	for i, addr := range strings.Split(alias.Address, addressSplitter) {
		addressData := map[string]interface{}{
			"address": addr,
		}
		if len(details) > i {
			addressData["detail"] = details[i]
		}
		addresses = append(addresses, addressData)
	}

	if err := d.Set("addresses", addresses); err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}

func resourceFirewallAliasUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*pfsenseapi.Client)
	request := getAliasFromResource(d)

	if err := client.Firewall.UpdateAlias(ctx, d.Id(), request, d.Get("apply_immediately").(bool)); err != nil {
		return diag.FromErr(err)
	}

	return resourceFirewallAliasRead(ctx, d, m)
}
