package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_firewall_rule (v0, SDKv2 provider v1) -> pfsense_firewall_rule (v1)
// ---------------------------------------------------------------------------

// firewallRuleTCPFlagV0 is the version-0 state shape of one element of the old
// "tcp_flag" list (SDKv2 TypeList of Resource with flag/present), from the
// provider v1 properties map of pfsense_firewall_rule.
type firewallRuleTCPFlagV0 struct {
	Flag    types.String `tfsdk:"flag"`
	Present types.Bool   `tfsdk:"present"`
}

// firewallRuleModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_firewall_rule. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// (the numeric tracker) is intentionally absent: the new provider id is the
// rule's description, not the tracker, so it is derived by the upgrader.
type firewallRuleModelV0 struct {
	AckQueue        types.String            `tfsdk:"ack_queue"`
	DefaultQueue    types.String            `tfsdk:"default_queue"`
	Description     types.String            `tfsdk:"description"`
	Direction       types.String            `tfsdk:"direction"`
	Disabled        types.Bool              `tfsdk:"disabled"`
	DnPipe          types.String            `tfsdk:"dn_pipe"`
	Destination     types.String            `tfsdk:"destination"`
	DestinationPort types.String            `tfsdk:"destination_port"`
	Floating        types.Bool              `tfsdk:"floating"`
	Gateway         types.String            `tfsdk:"gateway"`
	ICMPType        types.List              `tfsdk:"icmp_type"`
	Interface       types.List              `tfsdk:"interface"`
	IPProtocol      types.String            `tfsdk:"ip_protocol"`
	Log             types.Bool              `tfsdk:"log"`
	PdPipe          types.String            `tfsdk:"pdn_pipe"`
	Protocol        types.String            `tfsdk:"protocol"`
	Quick           types.Bool              `tfsdk:"quick"`
	Schedule        types.String            `tfsdk:"schedule"`
	Source          types.String            `tfsdk:"source"`
	SourcePort      types.String            `tfsdk:"source_port"`
	StateType       types.String            `tfsdk:"state_type"`
	TCPFlag         []firewallRuleTCPFlagV0 `tfsdk:"tcp_flag"`
	Type            types.String            `tfsdk:"type"`
}

var _ resource.ResourceWithUpgradeState = (*firewallRuleResource)(nil)

// firewallRulePriorSchemaV0 is the SDKv2 pfsense_firewall_rule schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules (TypeList of String becomes a
// ListAttribute of String, the nested `tcp_flag` TypeList of Resource becomes
// a ListAttribute of Object{flag String, present Bool}). The implicit SDKv2
// `id` (numeric tracker) is deliberately excluded — the new id is the rule
// description, so no id survives from the prior state.
var firewallRulePriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"ack_queue":        schema.StringAttribute{Optional: true},
		"default_queue":    schema.StringAttribute{Optional: true},
		"description":      schema.StringAttribute{Optional: true},
		"direction":        schema.StringAttribute{Optional: true},
		"disabled":         schema.BoolAttribute{Optional: true},
		"dn_pipe":          schema.StringAttribute{Optional: true},
		"destination":      schema.StringAttribute{Optional: true},
		"destination_port": schema.StringAttribute{Optional: true},
		"floating":         schema.BoolAttribute{Optional: true},
		"gateway":          schema.StringAttribute{Optional: true},
		"icmp_type": schema.ListAttribute{
			Optional:    true,
			ElementType: types.StringType,
		},
		"interface": schema.ListAttribute{
			Required:    true,
			ElementType: types.StringType,
		},
		"ip_protocol": schema.StringAttribute{Optional: true},
		"log":         schema.BoolAttribute{Optional: true},
		"pdn_pipe":    schema.StringAttribute{Optional: true},
		"protocol":    schema.StringAttribute{Optional: true},
		"quick":       schema.BoolAttribute{Optional: true},
		"schedule":    schema.StringAttribute{Optional: true},
		"source":      schema.StringAttribute{Optional: true},
		"source_port": schema.StringAttribute{Optional: true},
		"state_type":  schema.StringAttribute{Optional: true},
		"tcp_flag": schema.ListAttribute{
			Optional: true,
			ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
				"flag":    types.StringType,
				"present": types.BoolType,
			}},
		},
		"type": schema.StringAttribute{Required: true},
	},
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_firewall_rule state (schema version 0) is migrated in-place to
// schema version 1 with no recreation.
func (r *firewallRuleResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &firewallRulePriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 decodes the v0 state, maps every user-configurable value
// to its new home (renames + the tcp_flag split), derives the new id from the
// description, and writes the v1 state.
func (r *firewallRuleResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior firewallRuleModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The v1 provider identifies a rule solely by its `descr`, and Read /
	// Update / Delete look the rule up with findByKey(..., "descr", <id>).
	// The v0 `description` was Optional, and the SDKv2 persisted an unset
	// optional string as "" — so migrating such a rule would produce an
	// empty id that matches the FIRST unrelated rule with an empty descr and
	// silently PATCH or DELETE it. Abort the upgrade instead of warning: a
	// warning does not stop the apply.
	descr := prior.Description.ValueString()
	if descr == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_firewall_rule",
			"this rule has no `description`, but the v2 provider identifies firewall rules by a unique `descr` "+
				"and would otherwise match an unrelated rule with an empty description. "+
				"Assign a unique `description` to this rule in the pfSense UI and in your configuration, "+
				"refresh the v1 state so the description is recorded, then re-run the upgrade.",
		)
		return
	}

	cur, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// toCurrent maps every v0 value to its v1 home:
//
//	ack_queue  -> ackqueue
//	default_queue -> defaultqueue
//	description -> descr AND the new resource id (the old numeric tracker is
//	              NOT carried; an empty description aborts the upgrade in
//	              upgradeStateV0To1, so it never reaches here)
//	dn_pipe    -> dnpipe
//	icmp_type  -> icmptype
//	ip_protocol -> ipprotocol (values pass through; anything other than
//	              'inet'/'inet6' — e.g. the v0-only 'inet46' — is surfaced as
//	              a warning, since the v1 validator rejects it)
//	pdn_pipe   -> pdnpipe
//	schedule   -> sched
//	state_type -> statetype
//	tcp_flag   -> tcp_flags_set + tcp_flags_out_of + tcp_flags_any (split)
//	floating   -> DROPPED (warning when true)
//
// Optional strings go through emptyToNull and the optional bools (disabled,
// log, quick) through falseToNull, so the SDKv2 "" / false zero values do not
// land in v1 state where the framework means null. The v1-only attributes
// (dscp, tag) are left null so Read repopulates them. Both v1 lists are built
// with explicit element types; a null prior list stays null.
func (m firewallRuleModelV0) toCurrent(ctx context.Context) (firewallRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	descr := m.Description.ValueString()

	cur := firewallRuleModel{
		ID:              types.StringValue(descr),
		Type:            m.Type,
		Interface:       m.Interface,
		IPProtocol:      emptyToNull(m.IPProtocol),
		Protocol:        emptyToNull(m.Protocol),
		ICMPType:        emptyListToNull(ctx, m.ICMPType),
		Source:          emptyToNull(m.Source),
		SourcePort:      emptyToNull(m.SourcePort),
		Destination:     emptyToNull(m.Destination),
		DestinationPort: emptyToNull(m.DestinationPort),
		Descr:           types.StringValue(descr),
		Disabled:        falseToNull(m.Disabled),
		Log:             falseToNull(m.Log),
		StateType:       emptyToNull(m.StateType),
		Gateway:         emptyToNull(m.Gateway),
		Sched:           emptyToNull(m.Schedule),
		DnPipe:          emptyToNull(m.DnPipe),
		PdPipe:          emptyToNull(m.PdPipe),
		DefaultQueue:    emptyToNull(m.DefaultQueue),
		AckQueue:        emptyToNull(m.AckQueue),
		Quick:           falseToNull(m.Quick),
		Direction:       emptyToNull(m.Direction),
	}

	// floating has no v1 equivalent: warn when the old rule was floating,
	// drop silently otherwise.
	if m.Floating.ValueBool() {
		diags.AddWarning(
			"Floating firewall rule not carried over",
			"The pfsense_firewall_rule resource in this provider version no longer supports floating rules. The rule has been migrated without its floating setting; review and reconfigure it as a non-floating rule.",
		)
	}

	// The v0 provider accepted ip_protocol = "inet46" (dual stack); the v1
	// resource validates ipprotocol with OneOf("inet", "inet6"), so anything
	// else will fail validation on the next plan. Warn at upgrade time rather
	// than letting the practitioner discover it as an opaque validation error.
	if ipproto := cur.IPProtocol.ValueString(); !cur.IPProtocol.IsNull() && ipproto != "inet" && ipproto != "inet6" {
		diags.AddWarning(
			"Unsupported ip_protocol value carried into upgraded state",
			"The v0 rule set `ip_protocol = \""+ipproto+"\"`, but the v2 `ipprotocol` attribute only accepts "+
				"\"inet\" or \"inet6\". The value was migrated as-is and the next plan will fail validation. "+
				"Set `ipprotocol` to \"inet\" or \"inet6\"; if the rule needs to cover both address families, "+
				"split it into two rules.",
		)
	}

	// tcp_flag split (mirrors the old updateRequest): coveredFlags = every
	// flag in the list (in order), setFlags = only present==true flags,
	// TCPFlagsAny is null when coveredFlags is empty (the unset default is
	// "any") and false when specific flags are selected.
	//
	// SDKv2 persists an unset optional TypeList as [], which req.State.Get
	// decodes into a non-nil empty slice, so a v1 rule that never used tcp_flag
	// arrives here with an empty (not nil) TCPFlag. Normalise both nil and empty
	// to null so the Optional tcp_flags_out_of/tcp_flags_set attributes plan
	// null instead of surfacing a spurious [] -> null diff on the first plan.
	if len(m.TCPFlag) == 0 {
		// "any" is the unset default for tcp_flags_any (Optional, not
		// Computed), so leaving it null avoids a spurious true->null diff on
		// the next plan.
		cur.TCPFlagsAny = types.BoolNull()
		cur.TCPFlagsOutOf = types.ListNull(types.StringType)
		cur.TCPFlagsSet = types.ListNull(types.StringType)
		return cur, diags
	}

	outOf := make([]attr.Value, 0, len(m.TCPFlag))
	set := make([]attr.Value, 0, len(m.TCPFlag))
	for _, f := range m.TCPFlag {
		name := f.Flag.ValueString()
		outOf = append(outOf, types.StringValue(name))
		if f.Present.ValueBool() {
			set = append(set, types.StringValue(name))
		}
	}

	cur.TCPFlagsOutOf = types.ListValueMust(types.StringType, outOf)
	cur.TCPFlagsSet = types.ListValueMust(types.StringType, set)
	if len(outOf) == 0 {
		cur.TCPFlagsAny = types.BoolNull()
	} else {
		cur.TCPFlagsAny = types.BoolValue(false)
	}

	return cur, diags
}
