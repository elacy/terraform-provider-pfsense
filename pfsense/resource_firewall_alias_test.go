package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceFirewallAliasTest() resourceTest {
	return &tfResourceTest[pfsenseapi.FirewallAliasRequest, pfsenseapi.FirewallAlias, string]{
		resource: resourceFirewallAlias(),
	}
}
