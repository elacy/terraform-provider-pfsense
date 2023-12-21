package pfsense

import (
	"github.com/sjafferali/pfsense-api-goclient/pfsenseapi"
)

func resourceInterfaceTest() resourceTest {
	return &tfResourceTest[pfsenseapi.InterfaceRequest, pfsenseapi.Interface, string]{
		resource: resourceInterface(),
	}
}
