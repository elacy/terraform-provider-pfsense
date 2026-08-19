package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// ipsecPhase1Mock is an in-memory IPsec phase 1 backend that emulates the v2
// REST API envelope, the descr-keyed lookup, and the system-assigned ikeid.
type ipsecPhase1Mock struct {
	mu    sync.Mutex
	ikeid int64
	items []map[string]any
}

func (m *ipsecPhase1Mock) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case r.URL.Path == "/api/v2/vpn/ipsec/phase1s" && r.Method == http.MethodGet:
			filter := r.URL.Query().Get("descr__exact")
			out := []map[string]any{}
			for i, a := range m.items {
				if filter != "" && a["descr"] != filter {
					continue
				}
				cp := map[string]any{"id": i, "ikeid": a["ikeid"]}
				for k, v := range a {
					cp[k] = v
				}
				out = append(out, cp)
			}
			writeEnvelope(w, 200, out)

		case r.URL.Path == "/api/v2/vpn/ipsec/phase1" && r.Method == http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeEnvelope(w, 400, nil)
				return
			}
			delete(body, "apply")
			m.ikeid++
			body["ikeid"] = m.ikeid
			m.items = append(m.items, body)
			idx := len(m.items) - 1
			resp := map[string]any{"id": idx, "ikeid": m.ikeid}
			for k, v := range body {
				resp[k] = v
			}
			writeEnvelope(w, 200, resp)

		case r.URL.Path == "/api/v2/vpn/ipsec/phase1" && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeEnvelope(w, 400, nil)
				return
			}
			idx := int(body["id"].(float64))
			if idx < 0 || idx >= len(m.items) {
				writeEnvelope(w, 404, nil)
				return
			}
			delete(body, "apply")
			delete(body, "id")
			// ikeid is system-assigned and Update does not repopulate it, so
			// the stored value must survive an in-place update unchanged.
			ikeid := m.items[idx]["ikeid"]
			for k, v := range body {
				m.items[idx][k] = v
			}
			m.items[idx]["ikeid"] = ikeid
			resp := map[string]any{"id": idx, "ikeid": ikeid}
			for k, v := range m.items[idx] {
				resp[k] = v
			}
			writeEnvelope(w, 200, resp)

		case r.URL.Path == "/api/v2/vpn/ipsec/phase1" && r.Method == http.MethodDelete:
			idx := int(mustFloat(r.URL.Query().Get("id")))
			if idx < 0 || idx >= len(m.items) {
				writeEnvelope(w, 404, nil)
				return
			}
			m.items = append(m.items[:idx], m.items[idx+1:]...)
			writeEnvelope(w, 200, nil)

		default:
			writeEnvelope(w, 404, nil)
		}
	})
}

// TestAccIPsecPhase1ConstantIkeid verifies the observable behaviour that
// constantComputed* exists to guarantee: a system-assigned ID (ikeid) survives
// an in-place update of a non-key attribute unchanged, and the follow-up plan
// is empty (no spurious "known after apply" diff). This exercises the real
// resource lifecycle through an in-process mock rather than the framework's
// UseStateForUnknown modifier in isolation.
func TestAccIPsecPhase1ConstantIkeid(t *testing.T) {
	mock := &ipsecPhase1Mock{}
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: ipsecPhase1Config(srv.URL, "10.0.0.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_ipsec_phase1.p1", "ikeid", "1"),
					resource.TestCheckResourceAttr("pfsense_ipsec_phase1.p1", "remote_gateway", "10.0.0.1"),
				),
			},
			{
				Config: ipsecPhase1Config(srv.URL, "10.0.0.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_ipsec_phase1.p1", "ikeid", "1"),
					resource.TestCheckResourceAttr("pfsense_ipsec_phase1.p1", "remote_gateway", "10.0.0.2"),
				),
			},
		},
	})
}

func ipsecPhase1Config(url, gateway string) string {
	return fmt.Sprintf(`
provider "pfsense" {
  url     = %q
  api_key = "test-key"
}

resource "pfsense_ipsec_phase1" "p1" {
  descr                 = "site-a"
  iketype               = "ikev2"
  protocol              = "inet"
  interface             = "wan"
  remote_gateway        = %q
  authentication_method = "pre_shared_key"
  pre_shared_key        = "secret"
  myid_type             = "myaddress"
  peerid_type           = "any"
}
`, url, gateway)
}
