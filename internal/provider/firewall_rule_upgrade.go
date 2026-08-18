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

// ruleTCPFlagV0 is the version-0 state shape of one element of the old
// "tcp_flag" list (SDKv2 TypeList of Resource with flag/present), from the
// provider v1 properties map of pfsense_firewall_rule.
type ruleTCPFlagV0 struct {
	Flag    types.String `tfsdk:"flag"`
	Present types.Bool   `tfsdk:"present"`
}

// ruleModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_firewall_rule. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// (the numeric tracker) is intentionally absent: the new provider id is the
// rule's description, not the tracker, so it is derived in toCurrent.
type ruleModelV0 struct {
	AckQueue        types.String    `tfsdk:"ack_queue"`
	DefaultQueue    types.String    `tfsdk:"default_queue"`
	Description     types.String    `tfsdk:"description"`
	Direction       types.String    `tfsdk:"direction"`
	Disabled        types.Bool      `tfsdk:"disabled"`
	DnPipe          types.String    `tfsdk:"dn_pipe"`
	Destination     types.String    `tfsdk:"destination"`
	DestinationPort types.String    `tfsdk:"destination_port"`
	Floating        types.Bool      `tfsdk:"floating"`
	Gateway         types.String    `tfsdk:"gateway"`
	ICMPType        types.List      `tfsdk:"icmp_type"`
	Interface       types.List      `tfsdk:"interface"`
	IPProtocol      types.String    `tfsdk:"ip_protocol"`
	Log             types.Bool      `tfsdk:"log"`
	PdPipe          types.String    `tfsdk:"pdn_pipe"`
	Protocol        types.String    `tfsdk:"protocol"`
	Quick           types.Bool      `tfsdk:"quick"`
	Schedule        types.String    `tfsdk:"schedule"`
	Source          types.String    `tfsdk:"source"`
	SourcePort      types.String    `tfsdk:"source_port"`
	StateType       types.String    `tfsdk:"state_type"`
	TCPFlag         []ruleTCPFlagV0 `tfsdk:"tcp_flag"`
	Type            types.String    `tfsdk:"type"`
}

var _ resource.ResourceWithUpgradeState = (*firewallRuleResource)(nil)

// rulePriorSchema is the SDKv2 pfsense_firewall_rule schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules (TypeList of String becomes a
// ListAttribute of String, the nested `tcp_flag` TypeList of Resource becomes
// a ListAttribute of Object{flag String, present Bool}). The implicit SDKv2
// `id` (numeric tracker) is deliberately excluded — the new id is the rule
// description, so no id survives from the prior state.
var rulePriorSchema = schema.Schema{
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
			PriorSchema:   &rulePriorSchema,
			StateUpgrader: ruleUpgradeStateV0To1,
		},
	}
}

// ruleUpgradeStateV0To1 decodes the v0 state, maps every user-configurable
// value to its new home (renames + the tcp_flag split), derives the new id
// from the description, and writes the v1 state.
func ruleUpgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior ruleModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
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
//	              NOT carried; an empty description yields an empty id plus a
//	              warning, since the v1 provider needs a unique descr)
//	dn_pipe    -> dnpipe
//	icmp_type  -> icmptype
//	ip_protocol -> ipprotocol (values pass through; 'inet46' is surfaced by
//	              new validation)
//	pdn_pipe   -> pdnpipe
//	schedule   -> sched
//	state_type -> statetype
//	tcp_flag   -> tcp_flags_set + tcp_flags_out_of + tcp_flags_any (split)
//	floating   -> DROPPED (warning when true)
//
// Identity attributes pass through unchanged. The v1-only attributes (dscp,
// tag) are left null so Read repopulates them. Both v1 lists are built with
// explicit element types; a null prior list stays null.
func (m ruleModelV0) toCurrent(ctx context.Context) (firewallRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	descr := m.Description.ValueString()

	cur := firewallRuleModel{
		ID:              types.StringValue(descr),
		Type:            m.Type,
		Interface:       m.Interface,
		IPProtocol:      m.IPProtocol,
		Protocol:        m.Protocol,
		ICMPType:        m.ICMPType,
		Source:          m.Source,
		SourcePort:      m.SourcePort,
		Destination:     m.Destination,
		DestinationPort: m.DestinationPort,
		Descr:           types.StringValue(descr),
		Disabled:        m.Disabled,
		Log:             m.Log,
		StateType:       m.StateType,
		Gateway:         m.Gateway,
		Sched:           m.Schedule,
		DnPipe:          m.DnPipe,
		PdPipe:          m.PdPipe,
		DefaultQueue:    m.DefaultQueue,
		AckQueue:        m.AckQueue,
		Quick:           m.Quick,
		Direction:       m.Direction,
	}

	// floating has no v1 equivalent: warn when the old rule was floating,
	// drop silently otherwise.
	if m.Floating.ValueBool() {
		diags.AddWarning(
			"Floating firewall rule not carried over",
			"The pfsense_firewall_rule resource in this provider version no longer supports floating rules. The rule has been migrated without its floating setting; review and reconfigure it as a non-floating rule.",
		)
	}

	if descr == "" {
		diags.AddWarning(
			"Firewall rule has no description",
			"The v1 provider identifies firewall rules by a unique `descr` (description). This rule was migrated with an empty id/descr and needs a unique description assigned to be tracked correctly.",
		)
	}

	// tcp_flag split (mirrors the old updateRequest): coveredFlags = every
	// flag in the list (in order), setFlags = only present==true flags,
	// TCPFlagsAny = true iff coveredFlags empty.
	if m.TCPFlag == nil {
		cur.TCPFlagsAny = types.BoolValue(true)
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
	cur.TCPFlagsAny = types.BoolValue(len(outOf) == 0)

	return cur, diags
}
