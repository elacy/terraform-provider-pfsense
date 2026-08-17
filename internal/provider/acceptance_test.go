package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live acceptance tests talk to a real pfSense instance and (for the resource
// tests) mutate its configuration. They are therefore opt-in: nothing here runs
// unless TF_ACC=1 is exported, so `go test ./...` stays hermetic and only
// exercises the mock-backed tests in firewall_alias_resource_test.go.
//
// Connection details come from the environment only — credentials must never be
// committed to this repository:
//
//	PFSENSE_URL       base URL of the box, e.g. https://192.168.1.1
//	PFSENSE_USERNAME  local-database username
//	PFSENSE_PASSWORD  local-database password
//	PFSENSE_API_KEY   optional; when set it replaces username/password auth
const (
	envURL      = "PFSENSE_URL"
	envUsername = "PFSENSE_USERNAME"
	envPassword = "PFSENSE_PASSWORD"
	envAPIKey   = "PFSENSE_API_KEY"
)

// testAccPreCheck gates every live test. It skips (rather than fails) when
// TF_ACC is not opted in, but fails loudly on a half-configured environment: a
// missing credential there is a setup mistake, not a reason to silently pass.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") != "1" {
		t.Skip("TF_ACC=1 is not set; skipping live acceptance test")
	}

	if os.Getenv(envURL) == "" {
		t.Fatalf("%s must be set for live acceptance tests", envURL)
	}

	// Either auth mode is fine, but a half-configured one is a setup mistake:
	// an API key alone, or a complete username/password pair, but not a
	// username without its password.
	hasAPIKey := os.Getenv(envAPIKey) != ""
	hasUserPass := os.Getenv(envUsername) != "" && os.Getenv(envPassword) != ""
	if !hasAPIKey && !hasUserPass {
		t.Fatalf("live acceptance tests need either %s, or both %s and %s", envAPIKey, envUsername, envPassword)
	}
}

// testAccProviderConfig renders the provider block used by every live test step
// from the environment. TLS verification is skipped because pfSense boxes serve
// the API under a self-signed certificate by default.
func testAccProviderConfig() string {
	if apiKey := os.Getenv(envAPIKey); apiKey != "" {
		return fmt.Sprintf(`
provider "pfsense" {
  url             = %q
  api_key         = %q
  skip_tls_verify = true
}
`, os.Getenv(envURL), apiKey)
	}

	return fmt.Sprintf(`
provider "pfsense" {
  url             = %q
  username        = %q
  password        = %q
  skip_tls_verify = true
}
`, os.Getenv(envURL), os.Getenv(envUsername), os.Getenv(envPassword))
}

// testAccClient builds an API client from the same environment the live
// provider block uses. Checks that must observe the box directly rather than
// through Terraform state (CheckDestroy) go through this.
func testAccClient() (*client.Client, error) {
	return client.New(client.Config{
		URL:           os.Getenv(envURL),
		Username:      os.Getenv(envUsername),
		Password:      os.Getenv(envPassword),
		APIKey:        os.Getenv(envAPIKey),
		SkipTLSVerify: true,
	})
}

// testAccCheckListAttrMinLen asserts that a computed list attribute is present
// in state and holds at least minLen entries. Live inventories differ per box,
// so read-only tests assert a lower bound instead of an exact count.
func testAccCheckListAttrMinLen(resourceName, attribute string, minLen int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		countKey := attribute + ".#"
		raw, ok := rs.Primary.Attributes[countKey]
		if !ok {
			return fmt.Errorf("%s: attribute %s is not set", resourceName, countKey)
		}
		count, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("%s: attribute %s is not a number: %q", resourceName, countKey, raw)
		}
		if count < minLen {
			return fmt.Errorf("%s: expected at least %d %s entries, got %d", resourceName, minLen, attribute, count)
		}
		return nil
	}
}

// testAccFirewallAliasExists reports whether an alias is present on the live
// box, resolving it by its natural key the same way the resource does.
func testAccFirewallAliasExists(name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, firewallAliasPlural, "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up firewall alias %q: %w", name, err)
	}
	return found, nil
}

// testAccCheckFirewallAliasAbsent verifies an alias is gone from the live box.
func testAccCheckFirewallAliasAbsent(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccFirewallAliasExists(name)
		if err != nil {
			return fmt.Errorf("checking firewall alias %q was destroyed: %w", name, err)
		}
		if found {
			return fmt.Errorf("firewall alias %q still exists after destroy", name)
		}
		return nil
	}
}
