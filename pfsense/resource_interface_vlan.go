package pfsense

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceInterfaceVLAN() *resource[pfsenseapi.VLANRequest, pfsenseapi.VLAN, string] {
	return &resource[pfsenseapi.VLANRequest, pfsenseapi.VLAN, string]{
		name:        "pfsense_interface_vlan",
		description: "VLAN",
		delete: func(ctx context.Context, client *pfsenseapi.Client, _ string, id string) error {
			return client.Interface.DeleteVLAN(ctx, id)
		},
		list: func(ctx context.Context, client *pfsenseapi.Client, _ string) ([]*pfsenseapi.VLAN, error) {
			return client.Interface.ListVLANs(ctx)
		},
		update: func(ctx context.Context, client *pfsenseapi.Client, id string, request *pfsenseapi.VLANRequest) (*pfsenseapi.VLAN, error) {
			return client.Interface.UpdateVLAN(ctx, id, *request)
		},
		create: func(ctx context.Context, client *pfsenseapi.Client, request *pfsenseapi.VLANRequest) (*pfsenseapi.VLAN, error) {
			return client.Interface.CreateVLAN(ctx, *request)
		},
		getId: func(_ context.Context, _ *pfsenseapi.Client, response *pfsenseapi.VLAN) (string, error) {
			return response.Vlanif, nil
		},
		properties: map[string]*resourceProperty[pfsenseapi.VLANRequest, pfsenseapi.VLAN]{
			"if": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Parent interface to add the new VLAN to.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.VLANRequest) error {
					req.If = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.VLAN) (interface{}, error) {
					return req.If, nil
				},
			},
			"tag": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Required:     true,
					Description:  "VLAN tag to add to the parent interface",
					ValidateFunc: validation.IntBetween(1, 4094),
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.VLANRequest) error {
					req.Tag = d.Get(name).(int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.VLAN) (interface{}, error) {
					return req.Tag, nil
				},
			},
			"pcp": {
				schema: &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validation.IntBetween(0, 7),
					Description:  "802.1q VLAN priority.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.VLANRequest) error {
					req.Pcp = d.Get(name).(*int)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.VLAN) (interface{}, error) {
					return req.Pcp, nil
				},
			},
			"description": {
				schema: &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Description of the VLAN interface.",
				},
				updateRequest: func(d *schema.ResourceData, name string, req *pfsenseapi.VLANRequest) error {
					req.Descr = d.Get(name).(string)
					return nil
				},
				getFromResponse: func(req *pfsenseapi.VLAN) (interface{}, error) {
					return req.Descr, nil
				},
			},
		},
	}
}
