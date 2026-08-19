package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDNSResolverHostOverrideUpgradeStateV0To1 exercises the full state-upgrade
// path for pfsense_unbound_host_override →
// pfsense_services_dns_resolver_host_override, including the nested aliases
// list.
func TestDNSResolverHostOverrideUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	rawJSON, err := json.Marshal(map[string]any{
		"id":           "www.example.com",
		"dns":          "www.example.com",
		"ip_addresses": []string{"192.168.1.10", "192.168.1.11"},
		"description":  "web server",
		"aliases": []map[string]any{
			{"host_name": "mail", "domain_name": "example.com", "description": "mail alias"},
			{"host_name": "*", "domain_name": "example.net", "description": "wildcard"},
		},
	})
	if err != nil {
		t.Fatalf("marshal prior raw state: %v", err)
	}
	raw := &tfprotov6.RawState{JSON: rawJSON}

	// priorResourceID must surface the implicit id not present in PriorSchema.
	if got, want := priorResourceID(raw), "www.example.com"; got != want {
		t.Errorf("priorResourceID() = %q, want %q", got, want)
	}
	if got, want := splitDNSHostDomain("www.example.com"); got != "www" || want != "example.com" {
		t.Errorf("splitDNSHostDomain() = (%q, %q), want (\"www\", \"example.com\")", got, want)
	}

	priorValue, err := raw.UnmarshalWithOpts(
		dnsResolverHostOverridePriorSchemaV0.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal prior raw state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: raw,
		State:    &tfsdk.State{Raw: priorValue, Schema: dnsResolverHostOverridePriorSchemaV0},
	}
	var resp resource.UpgradeStateResponse

	var r dnsResolverHostOverrideResource
	var sreq resource.SchemaRequest
	var sresp resource.SchemaResponse
	r.Schema(ctx, sreq, &sresp)
	resp.State.Schema = sresp.Schema

	r.upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %s", resp.Diagnostics)
	}

	var got dnsResolverHostOverrideModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode upgraded state: %s", resp.Diagnostics)
	}

	if got, want := got.ID.ValueString(), "www|example.com"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := got.Host.ValueString(), "www"; got != want {
		t.Errorf("host = %q, want %q", got, want)
	}
	if got, want := got.Domain.ValueString(), "example.com"; got != want {
		t.Errorf("domain = %q, want %q", got, want)
	}
	if got, want := got.Descr.ValueString(), "web server"; got != want {
		t.Errorf("descr = %q, want %q", got, want)
	}
	if got, want := listStringValues(t, got.IP), []string{"192.168.1.10", "192.168.1.11"}; !equalStrings(got, want) {
		t.Errorf("ip = %v, want %v", got, want)
	}

	if got.Aliases.IsNull() || got.Aliases.IsUnknown() {
		t.Fatalf("aliases = %v, want a list of 2 elements", got.Aliases)
	}
	elements := got.Aliases.Elements()
	if len(elements) != 2 {
		t.Fatalf("len(aliases) = %d, want 2", len(elements))
	}
	wantAliases := []map[string]string{
		{"host": "mail", "domain": "example.com", "descr": "mail alias"},
		{"host": "*", "domain": "example.net", "descr": "wildcard"},
	}
	for i, element := range elements {
		obj, ok := element.(types.Object)
		if !ok {
			t.Fatalf("aliases[%d] = %T, want types.Object", i, element)
		}
		attrs := obj.Attributes()
		for k, want := range wantAliases[i] {
			v, ok := attrs[k].(types.String)
			if !ok {
				t.Fatalf("aliases[%d].%s = %T, want types.String", i, k, attrs[k])
			}
			if got := v.ValueString(); got != want {
				t.Errorf("aliases[%d].%s = %q, want %q", i, k, got, want)
			}
		}
	}
}

// TestDNSResolverHostOverrideUpgradeStateV0To1MissingID covers the fallback: when
// the prior raw state has no implicit "id", the "dns" attribute is used.
func TestDNSResolverHostOverrideUpgradeStateV0To1MissingID(t *testing.T) {
	ctx := context.Background()

	rawJSON, err := json.Marshal(map[string]any{
		"dns":          "mail.example.com",
		"ip_addresses": []string{"10.0.0.5"},
	})
	if err != nil {
		t.Fatalf("marshal prior raw state: %v", err)
	}
	raw := &tfprotov6.RawState{JSON: rawJSON}
	if got := priorResourceID(raw); got != "" {
		t.Fatalf("priorResourceID() = %q, want empty", got)
	}

	priorValue, err := raw.UnmarshalWithOpts(
		dnsResolverHostOverridePriorSchemaV0.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal prior raw state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: raw,
		State:    &tfsdk.State{Raw: priorValue, Schema: dnsResolverHostOverridePriorSchemaV0},
	}
	var resp resource.UpgradeStateResponse

	var r dnsResolverHostOverrideResource
	var sreq resource.SchemaRequest
	var sresp resource.SchemaResponse
	r.Schema(ctx, sreq, &sresp)
	resp.State.Schema = sresp.Schema

	r.upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %s", resp.Diagnostics)
	}

	var got dnsResolverHostOverrideModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode upgraded state: %s", resp.Diagnostics)
	}
	if got, want := got.ID.ValueString(), "mail|example.com"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := got.Host.ValueString(), "mail"; got != want {
		t.Errorf("host = %q, want %q", got, want)
	}
	if got, want := got.Domain.ValueString(), "example.com"; got != want {
		t.Errorf("domain = %q, want %q", got, want)
	}
}

// TestDNSResolverHostOverrideModelV0ToCurrent unit-tests the pure mapping on a
// representative version-0 model, including a null (unset) aliases list.
func TestDNSResolverHostOverrideModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	prior := dnsResolverHostOverrideModelV0{
		DNS:         types.StringValue("www.example.com"),
		IPAddresses: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.168.1.10")}),
		Description: types.StringValue("web server"),
		Aliases: []dnsResolverHostOverrideAliasV0{
			{
				HostName:    types.StringValue("mail"),
				DomainName:  types.StringValue("example.com"),
				Description: types.StringValue("mail alias"),
			},
		},
	}

	current, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("toCurrent returned diagnostics: %s", diags)
	}

	if !current.IP.Equal(prior.IPAddresses) {
		t.Errorf("ip = %v, want %v", current.IP, prior.IPAddresses)
	}
	if got, want := current.Descr.ValueString(), "web server"; got != want {
		t.Errorf("descr = %q, want %q", got, want)
	}
	// host/domain/id are derived from the resource id by the StateUpgrader,
	// not by toCurrent.
	if got, want := current.Host.ValueString(), ""; got != want {
		t.Errorf("host must not be set by toCurrent, got %q", got)
	}
	if got, want := current.Domain.ValueString(), ""; got != want {
		t.Errorf("domain must not be set by toCurrent, got %q", got)
	}

	elements := current.Aliases.Elements()
	if len(elements) != 1 {
		t.Fatalf("len(aliases) = %d, want 1", len(elements))
	}
	obj, ok := elements[0].(types.Object)
	if !ok {
		t.Fatalf("aliases[0] = %T, want types.Object", elements[0])
	}
	attrs := obj.Attributes()
	for k, want := range map[string]string{"host": "mail", "domain": "example.com", "descr": "mail alias"} {
		v, ok := attrs[k].(types.String)
		if !ok {
			t.Fatalf("aliases[0].%s = %T, want types.String", k, attrs[k])
		}
		if got := v.ValueString(); got != want {
			t.Errorf("aliases[0].%s = %q, want %q", k, got, want)
		}
	}

	// A null (unset) prior aliases list must stay null.
	priorNull := dnsResolverHostOverrideModelV0{
		DNS:         types.StringValue("www.example.com"),
		IPAddresses: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("192.168.1.10")}),
	}
	currentNull, diags := priorNull.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("toCurrent returned diagnostics: %s", diags)
	}
	if !currentNull.Aliases.IsNull() {
		t.Errorf("aliases must be null when prior aliases was null, got %v", currentNull.Aliases)
	}

	// splitDNSHostDomain edge cases.
	if got, want := splitDNSHostDomain("printer"); got != "printer" || want != "" {
		t.Errorf("splitDNSHostDomain(\"printer\") = (%q, %q), want (\"printer\", \"\")", got, want)
	}
	if got, want := splitDNSHostDomain(""); got != "" || want != "" {
		t.Errorf("splitDNSHostDomain(\"\") = (%q, %q), want (\"\", \"\")", got, want)
	}
}
