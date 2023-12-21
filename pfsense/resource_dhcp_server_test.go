package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceDhcpServerTest() resourceTest {
	return &tfResourceTest[pfsenseapi.DHCPServerConfigurationRequest, pfsenseapi.DHCPServerConfiguration, string]{
		resource: resourceDHCPServer(),
	}
}
