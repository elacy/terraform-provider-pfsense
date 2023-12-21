package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceDhcpStaticMappingTest() resourceTest {
	return &tfResourceTest[pfsenseapi.DHCPStaticMappingRequest, pfsenseapi.DHCPStaticMapping, string]{
		resource: resourceDHCPStaticMapping(),
	}
}
