package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceInterfaceVLANTest() resourceTest {
	return &tfResourceTest[pfsenseapi.VLANRequest, pfsenseapi.VLAN, string]{
		resource: resourceInterfaceVLAN(),
	}
}
