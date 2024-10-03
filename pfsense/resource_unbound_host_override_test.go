package pfsense

import "github.com/sjafferali/pfsense-api-goclient/pfsenseapi"

func resourceUnboundHostOverrideTest() resourceTest {
	return &tfResourceTest[pfsenseapi.UnboundHostOverride, pfsenseapi.UnboundHostOverride, string]{
		resource: resourceUnboundHostOverride(),
	}
}
