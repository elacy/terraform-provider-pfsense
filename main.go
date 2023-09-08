package main

import (
	"github.com/elacy/terraform-pfsense-provider/pfsense"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: pfsense.Provider,
	})
}
