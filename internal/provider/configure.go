package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// resourceClient extracts the shared *client.Client from resource Configure data.
func resourceClient(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected provider data",
			"Expected *client.Client but got a different type. This is a provider bug.",
		)
		return nil
	}
	return c
}

// dataSourceClient extracts the shared *client.Client from data source Configure data.
func dataSourceClient(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"unexpected provider data",
			"Expected *client.Client but got a different type. This is a provider bug.",
		)
		return nil
	}
	return c
}
