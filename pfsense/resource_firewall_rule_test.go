package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceFirewallRuleTest() resourceTest {
	return &tfResourceTest[pfsenseapi.FirewallRuleRequest, pfsenseapi.FirewallRule, int]{
		resource: resourceFirewallRule(),
	}
}
