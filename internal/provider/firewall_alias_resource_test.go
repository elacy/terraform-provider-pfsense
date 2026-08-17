package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccProtoV6ProviderFactories runs the provider in-process against a mock
// server, exercising the full CRUD lifecycle without a real pfSense instance.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"pfsense": providerserver.NewProtocol6WithError(New("test")()),
}

// aliasMock is an in-memory firewall-alias backend that emulates the v2 REST
// API response envelope and array-index ID semantics.
type aliasMock struct {
	mu      sync.Mutex
	aliases []map[string]any
}

func (m *aliasMock) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case r.URL.Path == "/api/v2/firewall/aliases" && r.Method == http.MethodGet:
			filter := r.URL.Query().Get("name__exact")
			out := []map[string]any{}
			for i, a := range m.aliases {
				if filter != "" && a["name"] != filter {
					continue
				}
				cp := map[string]any{"id": i}
				for k, v := range a {
					cp[k] = v
				}
				out = append(out, cp)
			}
			writeEnvelope(w, 200, out)

		case r.URL.Path == "/api/v2/firewall/alias" && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeEnvelope(w, 400, nil)
				return
			}
			delete(body, "apply")
			if body["name"] == nil || body["name"] == "" {
				writeEnvelope(w, 422, nil)
				return
			}
			m.aliases = append(m.aliases, body)
			idx := len(m.aliases) - 1
			resp := map[string]any{"id": idx}
			for k, v := range body {
				resp[k] = v
			}
			writeEnvelope(w, 200, resp)

		case r.URL.Path == "/api/v2/firewall/alias" && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeEnvelope(w, 400, nil)
				return
			}
			idx := int(body["id"].(float64))
			if idx < 0 || idx >= len(m.aliases) {
				writeEnvelope(w, 404, nil)
				return
			}
			delete(body, "apply")
			delete(body, "id")
			for k, v := range body {
				m.aliases[idx][k] = v
			}
			resp := map[string]any{"id": idx}
			for k, v := range m.aliases[idx] {
				resp[k] = v
			}
			writeEnvelope(w, 200, resp)

		case r.URL.Path == "/api/v2/firewall/alias" && r.Method == http.MethodDelete:
			idx := int(mustFloat(r.URL.Query().Get("id")))
			if idx < 0 || idx >= len(m.aliases) {
				writeEnvelope(w, 404, nil)
				return
			}
			m.aliases = append(m.aliases[:idx], m.aliases[idx+1:]...)
			writeEnvelope(w, 200, nil)

		default:
			writeEnvelope(w, 404, nil)
		}
	})
}

func writeEnvelope(w http.ResponseWriter, code int, data any) {
	w.WriteHeader(code)
	payload := map[string]any{
		"code":        code,
		"status":      "ok",
		"response_id": "SUCCESS",
		"message":     "",
	}
	if data != nil {
		payload["data"] = data
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func mustFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func TestAccFirewallAliasResource(t *testing.T) {
	mock := &aliasMock{}
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "pfsense" {
  url     = %q
  api_key = "test-key"
}

resource "pfsense_firewall_alias" "webservers" {
  name    = "webservers"
  type    = "host"
  descr   = "Web servers"
  address = ["10.0.0.10", "10.0.0.11"]
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "name", "webservers"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "type", "host"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "address.#", "2"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "id", "webservers"),
				),
			},
			{
				Config: fmt.Sprintf(`
provider "pfsense" {
  url     = %q
  api_key = "test-key"
}

resource "pfsense_firewall_alias" "webservers" {
  name    = "webservers"
  type    = "host"
  descr   = "Web servers (updated)"
  address = ["10.0.0.10"]
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "descr", "Web servers (updated)"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.webservers", "address.#", "1"),
				),
			},
		},
	})
}

func TestAccFirewallAliasesDataSource(t *testing.T) {
	mock := &aliasMock{}
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "pfsense" {
  url     = %q
  api_key = "test-key"
}

resource "pfsense_firewall_alias" "a" {
  name = "one"
  type = "host"
}

data "pfsense_firewall_aliases" "all" {
  depends_on = [pfsense_firewall_alias.a]
}
`, srv.URL),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pfsense_firewall_aliases.all", "aliases.#", "1"),
				),
			},
		},
	})
}
