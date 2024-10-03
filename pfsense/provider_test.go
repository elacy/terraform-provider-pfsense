package pfsense

import (
	"testing"
)

func Test_AtLeastOneRequiredProperty(t *testing.T) {
	p := Provider()

outer:
	for name, resource := range p.ResourcesMap {
		for _, schema := range resource.Schema {
			if schema.Required {
				continue outer
			}
		}
		t.Errorf("There isn't at least one required property in resource %s", name)
	}
}

func Test_Validate(t *testing.T) {
	p := Provider()

	err := p.InternalValidate()

	if err != nil {
		t.Errorf("Encountered error during validation %v", err)
	}
}

func Test_AllPropertiesAndResourcesAreDocumented(t *testing.T) {
	p := Provider()

	for name, resource := range p.ResourcesMap {
		if resource.Description == "" {
			t.Errorf("Resource %s has no documentation", name)
		}

		for property, schema := range resource.Schema {
			if schema.Description == "" {
				t.Errorf("Property %s on resource %s has no documentation", property, name)
			}
		}
	}
}

func Test_runResourceTests(t *testing.T) {
	p := Provider()

	resources := []resourceTest{
		resourceDhcpServerTest(),
		resourceDhcpStaticMappingTest(),
		resourceFirewallAliasTest(),
		resourceFirewallRuleTest(),
		resourceInterfaceTest(),
		resourceInterfaceVLANTest(),
		resourceUnboundHostOverrideTest(),
	}

	resourceMap := map[string]resourceTest{}

	for _, r := range resources {
		if _, exists := resourceMap[r.GetName()]; exists {
			t.Errorf("Duplicate Resource Test for %s", r.GetName())
		} else {
			resourceMap[r.GetName()] = r
		}
	}

	for resourceName := range p.ResourcesMap {
		r, exists := resourceMap[resourceName]

		if exists {
			r.RunTests(t)
			delete(resourceMap, resourceName)
		} else {
			t.Errorf("Unable to find tests for %s", resourceName)
		}
	}

	for resourceName := range resourceMap {
		t.Errorf("Test exists for %s resource but is not present in provider", resourceName)
	}
}
