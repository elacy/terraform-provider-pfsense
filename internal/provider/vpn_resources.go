package provider

import (
	"context"
	"fmt"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_ipsec_phase1
// Key: descr. The `encryption` collection is managed by
// pfsense_ipsec_phase1_encryption.
// ---------------------------------------------------------------------------

type ipsecPhase1Resource struct{ client *client.Client }

var _ resource.Resource = (*ipsecPhase1Resource)(nil)
var _ resource.ResourceWithConfigure = (*ipsecPhase1Resource)(nil)
var _ resource.ResourceWithImportState = (*ipsecPhase1Resource)(nil)

const (
	ipsecPhase1Plural   = "/api/v2/vpn/ipsec/phase1s"
	ipsecPhase1Singular = "/api/v2/vpn/ipsec/phase1"
)

type ipsecPhase1Model struct {
	ID                   types.String `tfsdk:"id"`
	IKEID                types.Int64  `tfsdk:"ikeid"`
	Descr                types.String `tfsdk:"descr"`
	Disabled             types.Bool   `tfsdk:"disabled"`
	IKEType              types.String `tfsdk:"iketype"`
	Mode                 types.String `tfsdk:"mode"`
	Protocol             types.String `tfsdk:"protocol"`
	Interface            types.String `tfsdk:"interface"`
	RemoteGateway        types.String `tfsdk:"remote_gateway"`
	AuthenticationMethod types.String `tfsdk:"authentication_method"`
	MyIDType             types.String `tfsdk:"myid_type"`
	MyIDData             types.String `tfsdk:"myid_data"`
	PeerIDType           types.String `tfsdk:"peerid_type"`
	PeerIDData           types.String `tfsdk:"peerid_data"`
	PreSharedKey         types.String `tfsdk:"pre_shared_key"`
	Certref              types.String `tfsdk:"certref"`
	Caref                types.String `tfsdk:"caref"`
	Lifetime             types.Int64  `tfsdk:"lifetime"`
	RekeyTime            types.Int64  `tfsdk:"rekey_time"`
	ReauthTime           types.Int64  `tfsdk:"reauth_time"`
	RandTime             types.Int64  `tfsdk:"rand_time"`
	StartAction          types.String `tfsdk:"startaction"`
	CloseAction          types.String `tfsdk:"closeaction"`
	NATTraversal         types.String `tfsdk:"nat_traversal"`
	Mobike               types.Bool   `tfsdk:"mobike"`
	GwDuplicates         types.Bool   `tfsdk:"gw_duplicates"`
	SplitConn            types.Bool   `tfsdk:"splitconn"`
	PRFSelectEnable      types.Bool   `tfsdk:"prfselect_enable"`
	IKEPort              types.String `tfsdk:"ikeport"`
	NATTPort             types.String `tfsdk:"nattport"`
	DPDDelay             types.Int64  `tfsdk:"dpd_delay"`
	DPDMaxFail           types.Int64  `tfsdk:"dpd_maxfail"`
}

func NewIPsecPhase1Resource() resource.Resource { return &ipsecPhase1Resource{} }

func (r *ipsecPhase1Resource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_ipsec_phase1"
}
func (r *ipsecPhase1Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *ipsecPhase1Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IPsec phase 1 (IKE) entry. Identified by its `descr` description. The phase 1 encryption proposals are managed by `pfsense_ipsec_phase1_encryption`.",
		Attributes: map[string]schema.Attribute{
			"id":                    computedIDAttribute(),
			"ikeid":                 computedIntAttribute("The unique IKE ID for this phase 1 entry, assigned by the system."),
			"descr":                 keyAttribute("A description for this phase 1 entry. Unique; immutable after creation."),
			"disabled":              optionalBoolAttribute("Disable this phase 1 entry."),
			"iketype":               requiredEnumAttribute("The IKE protocol version this phase 1 entry will use.", "ikev1", "ikev2", "auto"),
			"mode":                  enumAttribute("The IKEv1 negotiation mode this phase 1 entry will use (`iketype` must be `ikev1`).", "main", "aggressive"),
			"protocol":              requiredEnumAttribute("The IP version this phase 1 entry will use.", "inet", "inet6", "both"),
			"interface":             requiredStringAttribute("The interface for the local endpoint of this phase 1 entry."),
			"remote_gateway":        requiredStringAttribute("The IP address or hostname of the remote gateway.", isIPAddressOrHostname()),
			"authentication_method": requiredEnumAttribute("The IPsec authentication method this tunnel will use.", "pre_shared_key", "cert"),
			"myid_type":             requiredEnumAttribute("The identifier type used by the local end of the tunnel.", "myaddress", "address", "fqdn", "user_fqdn", "asn1dn", "keyid tag", "dyn_dns", "auto"),
			"myid_data":             optionalStringAttribute("The identifier value used by the local end of the tunnel. Required unless `myid_type` is `myaddress`."),
			"peerid_type":           requiredEnumAttribute("The identifier type used by the remote end of the tunnel.", "any", "peeraddress", "address", "fqdn", "user_fqdn", "asn1dn", "keyid tag", "dyn_dns", "auto"),
			"peerid_data":           optionalStringAttribute("The identifier value used by the remote end of the tunnel. Required unless `peerid_type` is `any` or `peeraddress`."),
			"pre_shared_key":        sensitiveStringAttribute("The Pre-Shared Key (PSK) value. Required when `authentication_method` is `pre_shared_key`."),
			"certref":               optionalStringAttribute("The `refid` of the certificate which identifies this system. Required when `authentication_method` is `cert`."),
			"caref":                 optionalStringAttribute("The `refid` of the certificate authority used to validate the peer certificate. Required when `authentication_method` is `cert`."),
			"lifetime":              optionalIntAttribute("The hard child SA lifetime (in seconds) after which the child SA will be expired."),
			"rekey_time":            optionalIntAttribute("The amount of time (in seconds) before a child SA establishes new keys."),
			"reauth_time":           optionalIntAttribute("The amount of time (in seconds) before a child SA is torn down and recreated from scratch."),
			"rand_time":             optionalIntAttribute("A random value up to this amount is subtracted from `rekey_time` to avoid simultaneous renegotiation."),
			"startaction":           enumAttribute("Forces specific initiation/responder behaviour for child SA (P2) entries.", "", "none", "start", "trap"),
			"closeaction":           enumAttribute("Controls the behaviour when the remote peer unexpectedly closes a child SA (P2).", "", "none", "start", "trap"),
			"nat_traversal":         enumAttribute("Controls the use of NAT-T (encapsulation of ESP in UDP packets).", "on", "force"),
			"mobike":                optionalBoolAttribute("Enable the use of MOBIKE for this tunnel."),
			"gw_duplicates":         optionalBoolAttribute("Allow multiple phase 1 configurations with the same remote gateway endpoint."),
			"splitconn":             optionalBoolAttribute("Use split connection entries with multiple phase 2 configurations."),
			"prfselect_enable":      optionalBoolAttribute("Enable manual Pseudo-Random Function (PRF) selection."),
			"ikeport":               optionalStringAttribute("The UDP port for IKE on the remote gateway.", isPort()),
			"nattport":              optionalStringAttribute("The UDP port for NAT-T on the remote gateway.", isPort()),
			"dpd_delay":             optionalIntAttribute("The delay (in seconds) between sending peer acknowledgement messages."),
			"dpd_maxfail":           optionalIntAttribute("The number of consecutive failures allowed before disconnecting."),
		},
	}
}

func (r *ipsecPhase1Resource) payload(m ipsecPhase1Model) map[string]any {
	p := map[string]any{}
	setString(p, "descr", m.Descr)
	setBool(p, "disabled", m.Disabled)
	setString(p, "iketype", m.IKEType)
	setString(p, "mode", m.Mode)
	setString(p, "protocol", m.Protocol)
	setString(p, "interface", m.Interface)
	setString(p, "remote_gateway", m.RemoteGateway)
	setString(p, "authentication_method", m.AuthenticationMethod)
	setString(p, "myid_type", m.MyIDType)
	setString(p, "myid_data", m.MyIDData)
	setString(p, "peerid_type", m.PeerIDType)
	setString(p, "peerid_data", m.PeerIDData)
	setString(p, "pre_shared_key", m.PreSharedKey)
	setString(p, "certref", m.Certref)
	setString(p, "caref", m.Caref)
	setInt(p, "lifetime", m.Lifetime)
	setInt(p, "rekey_time", m.RekeyTime)
	setInt(p, "reauth_time", m.ReauthTime)
	setInt(p, "rand_time", m.RandTime)
	setString(p, "startaction", m.StartAction)
	setString(p, "closeaction", m.CloseAction)
	setString(p, "nat_traversal", m.NATTraversal)
	setBool(p, "mobike", m.Mobike)
	setBool(p, "gw_duplicates", m.GwDuplicates)
	setBool(p, "splitconn", m.SplitConn)
	setBool(p, "prfselect_enable", m.PRFSelectEnable)
	setString(p, "ikeport", m.IKEPort)
	setString(p, "nattport", m.NATTPort)
	setInt(p, "dpd_delay", m.DPDDelay)
	setInt(p, "dpd_maxfail", m.DPDMaxFail)
	return p
}

func (r *ipsecPhase1Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipsecPhase1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, ipsecPhase1Singular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 1 entry", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 1 entry", err.Error())
		return
	}
	plan.IKEID = intValue(getInt(obj, "ikeid"))
	plan.ID = types.StringValue(plan.Descr.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase1Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipsecPhase1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, ipsecPhase1Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read IPsec phase 1 entry", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.IKEID = intValue(getInt(obj, "ikeid"))
	state.Descr = types.StringValue(descr)
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.IKEType = strValue(getString(obj, "iketype"))
	state.Mode = strValue(getString(obj, "mode"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.Interface = strValue(getString(obj, "interface"))
	state.RemoteGateway = strValue(getString(obj, "remote_gateway"))
	state.AuthenticationMethod = strValue(getString(obj, "authentication_method"))
	state.MyIDType = strValue(getString(obj, "myid_type"))
	state.MyIDData = strValue(getString(obj, "myid_data"))
	state.PeerIDType = strValue(getString(obj, "peerid_type"))
	state.PeerIDData = strValue(getString(obj, "peerid_data"))
	state.PreSharedKey = strValue(getString(obj, "pre_shared_key"))
	state.Certref = strValue(getString(obj, "certref"))
	state.Caref = strValue(getString(obj, "caref"))
	state.Lifetime = intValue(getInt(obj, "lifetime"))
	state.RekeyTime = intValue(getInt(obj, "rekey_time"))
	state.ReauthTime = intValue(getInt(obj, "reauth_time"))
	state.RandTime = intValue(getInt(obj, "rand_time"))
	state.StartAction = strValue(getString(obj, "startaction"))
	state.CloseAction = strValue(getString(obj, "closeaction"))
	state.NATTraversal = strValue(getString(obj, "nat_traversal"))
	state.Mobike = boolValue(getBool(obj, "mobike"))
	state.GwDuplicates = boolValue(getBool(obj, "gw_duplicates"))
	state.SplitConn = boolValue(getBool(obj, "splitconn"))
	state.PRFSelectEnable = boolValue(getBool(obj, "prfselect_enable"))
	state.IKEPort = strValue(getString(obj, "ikeport"))
	state.NATTPort = strValue(getString(obj, "nattport"))
	state.DPDDelay = intValue(getInt(obj, "dpd_delay"))
	state.DPDMaxFail = intValue(getInt(obj, "dpd_maxfail"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipsecPhase1Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipsecPhase1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ipsecPhase1Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 1 entry", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("IPsec phase 1 entry not found", "phase 1 entry "+descr+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, ipsecPhase1Singular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 1 entry", err.Error())
		return
	}
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase1Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipsecPhase1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ipsecPhase1Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 1 entry", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, ipsecPhase1Singular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 1 entry", err.Error())
	}
}

func (r *ipsecPhase1Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_ipsec_phase1_encryption
// Parent: IPsec phase 1 (natural key = descr). Child key:
// encryption_algorithm_name|hash_algorithm|dhgroup|encryption_algorithm_keylen.
// Resource ID = <phase1 descr>|<algorithm>|<hash>|<dhgroup>|<keylen>.
//
// The key length is part of the key because a phase 1 commonly carries the same
// algorithm/hash/DH group at several key lengths (AES-128 alongside AES-256);
// without it those rows would collide on a single Terraform ID.
// ---------------------------------------------------------------------------

type ipsecPhase1EncryptionResource struct{ client *client.Client }

var _ resource.Resource = (*ipsecPhase1EncryptionResource)(nil)
var _ resource.ResourceWithConfigure = (*ipsecPhase1EncryptionResource)(nil)
var _ resource.ResourceWithImportState = (*ipsecPhase1EncryptionResource)(nil)

const (
	ipsecPhase1EncryptionPlural   = "/api/v2/vpn/ipsec/phase1/encryptions"
	ipsecPhase1EncryptionSingular = "/api/v2/vpn/ipsec/phase1/encryption"
)

type ipsecPhase1EncryptionModel struct {
	ID                        types.String `tfsdk:"id"`
	ParentID                  types.String `tfsdk:"parent_id"`
	EncryptionAlgorithmName   types.String `tfsdk:"encryption_algorithm_name"`
	EncryptionAlgorithmKeylen types.Int64  `tfsdk:"encryption_algorithm_keylen"`
	HashAlgorithm             types.String `tfsdk:"hash_algorithm"`
	DHGroup                   types.Int64  `tfsdk:"dhgroup"`
	PRFAlgorithm              types.String `tfsdk:"prf_algorithm"`
}

func NewIPsecPhase1EncryptionResource() resource.Resource { return &ipsecPhase1EncryptionResource{} }

func (r *ipsecPhase1EncryptionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_ipsec_phase1_encryption"
}
func (r *ipsecPhase1EncryptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *ipsecPhase1EncryptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an encryption proposal on an IPsec phase 1 entry. Identified by the parent phase 1 description and the algorithm, key length, hash and DH group combination.",
		Attributes: map[string]schema.Attribute{
			"id":        computedIDAttribute(),
			"parent_id": parentIDAttribute("The `descr` of the parent IPsec phase 1 entry."),
			"encryption_algorithm_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the encryption algorithm to use for this P1 encryption item.",
				Validators: []validator.String{
					stringvalidator.OneOf("aes", "aes128gcm", "aes192gcm", "aes256gcm", "chacha20poly1305"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"encryption_algorithm_keylen": schema.Int64Attribute{
				Optional:    true,
				Description: "The key length for the encryption algorithm. Required for variable key length algorithms. Part of this item's identity, so changing it replaces the resource.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"hash_algorithm": schema.StringAttribute{
				Required:    true,
				Description: "The hash algorithm to use for this P1 encryption item.",
				Validators: []validator.String{
					stringvalidator.OneOf("sha1", "sha256", "sha384", "sha512", "aesxcbc"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dhgroup": schema.Int64Attribute{
				Required:    true,
				Description: "The Diffie-Hellman (DH) group to use for this P1 encryption item.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"prf_algorithm": enumAttribute("The PRF algorithm to use for this P1 encryption item. Has no effect unless the phase 1 entry has PRF enabled.", "sha1", "sha256", "sha384", "sha512", "aesxcbc"),
		},
	}
}

func (r *ipsecPhase1EncryptionResource) key(parent, algorithm, hash string, dhgroup, keylen int64) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d", parent, algorithm, hash, dhgroup, keylen)
}

// identity returns the parent descr and the composite child key, preferring the
// state/plan attributes and falling back to the resource ID (set on import).
func (r *ipsecPhase1EncryptionResource) identity(m ipsecPhase1EncryptionModel) (parent, algorithm, hash string, dhgroup, keylen int64) {
	parent = m.ParentID.ValueString()
	algorithm = m.EncryptionAlgorithmName.ValueString()
	hash = m.HashAlgorithm.ValueString()
	dhgroup = m.DHGroup.ValueInt64()
	keylen = m.EncryptionAlgorithmKeylen.ValueInt64()
	if parent == "" {
		parts := splitKeyN(m.ID.ValueString(), 5)
		parent, algorithm, hash = parts[0], parts[1], parts[2]
		dhgroup, keylen = atoi64(parts[3]), atoi64(parts[4])
	}
	return parent, algorithm, hash, dhgroup, keylen
}

// find locates the proposal row within its parent phase 1. keylen only narrows
// the match when it is set: fixed-key-length algorithms (the GCM variants,
// ChaCha20-Poly1305) carry no key length at all, and for those the remaining
// fields already identify the row uniquely.
func (r *ipsecPhase1EncryptionResource) find(ctx context.Context, pid any, algorithm, hash string, dhgroup, keylen int64) (any, map[string]any, bool, error) {
	filters := map[string]string{
		"encryption_algorithm_name": algorithm,
		"hash_algorithm":            hash,
		"dhgroup":                   formatID(dhgroup),
	}
	if keylen != 0 {
		filters["encryption_algorithm_keylen"] = formatID(keylen)
	}
	return findByKeysInParent(ctx, r.client, ipsecPhase1EncryptionPlural, formatID(pid), filters)
}

// keylenValue renders an optional key length back into state. A zero means the
// algorithm has a fixed key length and carries none, which must stay null so it
// does not read back as drift against an unset config value — and, since key
// length is part of the natural key, as a replace loop.
func keylenValue(keylen int64) types.Int64 {
	if keylen == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(keylen)
}

func (r *ipsecPhase1EncryptionResource) payload(m ipsecPhase1EncryptionModel) map[string]any {
	p := map[string]any{}
	setString(p, "encryption_algorithm_name", m.EncryptionAlgorithmName)
	setInt(p, "encryption_algorithm_keylen", m.EncryptionAlgorithmKeylen)
	setString(p, "hash_algorithm", m.HashAlgorithm)
	setInt(p, "dhgroup", m.DHGroup)
	setString(p, "prf_algorithm", m.PRFAlgorithm)
	return p
}

func (r *ipsecPhase1EncryptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipsecPhase1EncryptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, algorithm, hash, dhgroup, keylen := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase1Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 1 entry", err.Error())
		return
	}
	payload := r.payload(plan)
	payload["parent_id"] = pid
	if _, err := r.client.Create(ctx, ipsecPhase1EncryptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 1 encryption item", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, algorithm, hash, dhgroup, keylen))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase1EncryptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipsecPhase1EncryptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, algorithm, hash, dhgroup, keylen := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase1Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 1 entry", err.Error())
		return
	}
	_, obj, found, err := r.find(ctx, pid, algorithm, hash, dhgroup, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to read IPsec phase 1 encryption item", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parent, algorithm, hash, dhgroup, keylen))
	state.ParentID = types.StringValue(parent)
	state.EncryptionAlgorithmName = strValue(getString(obj, "encryption_algorithm_name"))
	state.EncryptionAlgorithmKeylen = keylenValue(keylen)
	state.HashAlgorithm = strValue(getString(obj, "hash_algorithm"))
	state.DHGroup = intValue(getInt(obj, "dhgroup"))
	state.PRFAlgorithm = strValue(getString(obj, "prf_algorithm"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipsecPhase1EncryptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipsecPhase1EncryptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, algorithm, hash, dhgroup, keylen := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase1Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 1 entry", err.Error())
		return
	}
	id, _, found, err := r.find(ctx, pid, algorithm, hash, dhgroup, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 1 encryption item", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("IPsec phase 1 encryption item not found", "encryption item "+r.key(parent, algorithm, hash, dhgroup, keylen)+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	payload["parent_id"] = pid
	if _, err := r.client.Update(ctx, ipsecPhase1EncryptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 1 encryption item", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, algorithm, hash, dhgroup, keylen))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase1EncryptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipsecPhase1EncryptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, algorithm, hash, dhgroup, keylen := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase1Plural, "descr", parent)
	if err != nil {
		// Parent phase 1 entry removed out-of-band: the child is already gone.
		return
	}
	id, _, found, err := r.find(ctx, pid, algorithm, hash, dhgroup, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 1 encryption item", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, ipsecPhase1EncryptionSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 1 encryption item", err.Error())
	}
}

func (r *ipsecPhase1EncryptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_ipsec_phase2
// Key: descr. The `encryption_algorithm_option` collection is managed by
// pfsense_ipsec_phase2_encryption.
// ---------------------------------------------------------------------------

type ipsecPhase2Resource struct{ client *client.Client }

var _ resource.Resource = (*ipsecPhase2Resource)(nil)
var _ resource.ResourceWithConfigure = (*ipsecPhase2Resource)(nil)
var _ resource.ResourceWithImportState = (*ipsecPhase2Resource)(nil)

const (
	ipsecPhase2Plural   = "/api/v2/vpn/ipsec/phase2s"
	ipsecPhase2Singular = "/api/v2/vpn/ipsec/phase2"
)

type ipsecPhase2Model struct {
	ID                  types.String `tfsdk:"id"`
	Uniqid              types.String `tfsdk:"uniqid"`
	Reqid               types.Int64  `tfsdk:"reqid"`
	IKEID               types.Int64  `tfsdk:"ikeid"`
	Descr               types.String `tfsdk:"descr"`
	Disabled            types.Bool   `tfsdk:"disabled"`
	Mode                types.String `tfsdk:"mode"`
	LocalIDType         types.String `tfsdk:"localid_type"`
	LocalIDAddress      types.String `tfsdk:"localid_address"`
	LocalIDNetbits      types.Int64  `tfsdk:"localid_netbits"`
	NATLocalIDType      types.String `tfsdk:"natlocalid_type"`
	NATLocalIDAddress   types.String `tfsdk:"natlocalid_address"`
	NATLocalIDNetbits   types.Int64  `tfsdk:"natlocalid_netbits"`
	RemoteIDType        types.String `tfsdk:"remoteid_type"`
	RemoteIDAddress     types.String `tfsdk:"remoteid_address"`
	RemoteIDNetbits     types.Int64  `tfsdk:"remoteid_netbits"`
	Protocol            types.String `tfsdk:"protocol"`
	HashAlgorithmOption types.List   `tfsdk:"hash_algorithm_option"`
	PFSGroup            types.Int64  `tfsdk:"pfsgroup"`
	Lifetime            types.Int64  `tfsdk:"lifetime"`
	RekeyTime           types.Int64  `tfsdk:"rekey_time"`
	RandTime            types.Int64  `tfsdk:"rand_time"`
	PingHost            types.String `tfsdk:"pinghost"`
	Keepalive           types.Bool   `tfsdk:"keepalive"`
}

func NewIPsecPhase2Resource() resource.Resource { return &ipsecPhase2Resource{} }

func (r *ipsecPhase2Resource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_ipsec_phase2"
}
func (r *ipsecPhase2Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *ipsecPhase2Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an IPsec phase 2 (child SA) entry. Identified by its `descr` description. The phase 2 encryption proposals are managed by `pfsense_ipsec_phase2_encryption`.",
		Attributes: map[string]schema.Attribute{
			"id":                    computedIDAttribute(),
			"uniqid":                computedStringAttribute("The unique ID used to identify this phase 2 entry internally, assigned by the system."),
			"reqid":                 computedIntAttribute("The unique request ID used to identify this phase 2 entry internally, assigned by the system."),
			"ikeid":                 requiredIntAttribute("The `ikeid` of the parent IPsec phase 1 entry this phase 2 entry belongs to."),
			"descr":                 keyAttribute("A description for this phase 2 entry. Unique; immutable after creation."),
			"disabled":              optionalBoolAttribute("Disable this phase 2 entry."),
			"mode":                  requiredEnumAttribute("The IPsec phase 2 mode this entry will use.", "tunnel", "tunnel6", "transport", "vti"),
			"localid_type":          optionalStringAttribute("The local ID type: an existing interface, `address` or `network`."),
			"localid_address":       optionalStringAttribute("The local network IP component of this IPsec security association."),
			"localid_netbits":       optionalIntAttribute("The subnet bits of the `localid_address` network.", isSubnetBits(128)),
			"natlocalid_type":       optionalStringAttribute("The NAT/BINAT translation type for this phase 2 entry. Leave unset if NAT/BINAT is not needed."),
			"natlocalid_address":    optionalStringAttribute("The NAT/BINAT local network IP component of this IPsec security association."),
			"natlocalid_netbits":    optionalIntAttribute("The subnet bits of the `natlocalid_address` network.", isSubnetBits(128)),
			"remoteid_type":         enumAttribute("The remote ID type: `address` or `network`.", "address", "network"),
			"remoteid_address":      optionalStringAttribute("The remote network IP component of this IPsec security association."),
			"remoteid_netbits":      optionalIntAttribute("The subnet bits of the `remoteid_address` network.", isSubnetBits(128)),
			"protocol":              enumAttribute("The phase 2 proposal protocol: `esp` (encryption and authentication) or `ah` (authentication only).", "esp", "ah"),
			"hash_algorithm_option": requiredStringListAttribute("The hashing algorithms used by this phase 2 entry. Ignored with GCM algorithms."),
			"pfsgroup":              optionalIntAttribute("The PFS key group this phase 2 entry should use. Use 0 to disable PFS."),
			"lifetime":              optionalIntAttribute("The hard IKE SA lifetime (in seconds) after which the IKE SA will be expired."),
			"rekey_time":            optionalIntAttribute("The amount of time (in seconds) before an IKE SA establishes new keys."),
			"rand_time":             optionalIntAttribute("A random value up to this amount is subtracted from `rekey_time` to avoid simultaneous renegotiation."),
			"pinghost":              optionalStringAttribute("The IP address to send an ICMP echo request to inside the tunnel."),
			"keepalive":             optionalBoolAttribute("Check this phase 2 entry and initiate it if disconnected."),
		},
	}
}

func (r *ipsecPhase2Resource) payload(m ipsecPhase2Model) map[string]any {
	p := map[string]any{}
	setInt(p, "ikeid", m.IKEID)
	setString(p, "descr", m.Descr)
	setBool(p, "disabled", m.Disabled)
	setString(p, "mode", m.Mode)
	setString(p, "localid_type", m.LocalIDType)
	setString(p, "localid_address", m.LocalIDAddress)
	setInt(p, "localid_netbits", m.LocalIDNetbits)
	setString(p, "natlocalid_type", m.NATLocalIDType)
	setString(p, "natlocalid_address", m.NATLocalIDAddress)
	setInt(p, "natlocalid_netbits", m.NATLocalIDNetbits)
	setString(p, "remoteid_type", m.RemoteIDType)
	setString(p, "remoteid_address", m.RemoteIDAddress)
	setInt(p, "remoteid_netbits", m.RemoteIDNetbits)
	setString(p, "protocol", m.Protocol)
	setStringList(p, "hash_algorithm_option", m.HashAlgorithmOption)
	setInt(p, "pfsgroup", m.PFSGroup)
	setInt(p, "lifetime", m.Lifetime)
	setInt(p, "rekey_time", m.RekeyTime)
	setInt(p, "rand_time", m.RandTime)
	setString(p, "pinghost", m.PingHost)
	setBool(p, "keepalive", m.Keepalive)
	return p
}

func (r *ipsecPhase2Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipsecPhase2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, ipsecPhase2Singular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 2 entry", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 2 entry", err.Error())
		return
	}
	plan.Uniqid = strValue(getString(obj, "uniqid"))
	plan.Reqid = intValue(getInt(obj, "reqid"))
	plan.ID = types.StringValue(plan.Descr.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase2Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipsecPhase2Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, ipsecPhase2Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read IPsec phase 2 entry", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.Uniqid = strValue(getString(obj, "uniqid"))
	state.Reqid = intValue(getInt(obj, "reqid"))
	state.IKEID = intValue(getInt(obj, "ikeid"))
	state.Descr = types.StringValue(descr)
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.Mode = strValue(getString(obj, "mode"))
	state.LocalIDType = strValue(getString(obj, "localid_type"))
	state.LocalIDAddress = strValue(getString(obj, "localid_address"))
	state.LocalIDNetbits = intValue(getInt(obj, "localid_netbits"))
	state.NATLocalIDType = strValue(getString(obj, "natlocalid_type"))
	state.NATLocalIDAddress = strValue(getString(obj, "natlocalid_address"))
	state.NATLocalIDNetbits = intValue(getInt(obj, "natlocalid_netbits"))
	state.RemoteIDType = strValue(getString(obj, "remoteid_type"))
	state.RemoteIDAddress = strValue(getString(obj, "remoteid_address"))
	state.RemoteIDNetbits = intValue(getInt(obj, "remoteid_netbits"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.HashAlgorithmOption = strListValue(ctx, getStringSlice(obj, "hash_algorithm_option"))
	state.PFSGroup = intValue(getInt(obj, "pfsgroup"))
	state.Lifetime = intValue(getInt(obj, "lifetime"))
	state.RekeyTime = intValue(getInt(obj, "rekey_time"))
	state.RandTime = intValue(getInt(obj, "rand_time"))
	state.PingHost = strValue(getString(obj, "pinghost"))
	state.Keepalive = boolValue(getBool(obj, "keepalive"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipsecPhase2Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipsecPhase2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ipsecPhase2Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 2 entry", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("IPsec phase 2 entry not found", "phase 2 entry "+descr+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, ipsecPhase2Singular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 2 entry", err.Error())
		return
	}
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase2Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipsecPhase2Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, ipsecPhase2Plural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 2 entry", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, ipsecPhase2Singular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 2 entry", err.Error())
	}
}

func (r *ipsecPhase2Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_ipsec_phase2_encryption
// Parent: IPsec phase 2 (natural key = descr). Child key: name|keylen.
// Resource ID = <phase2 descr>|<name>|<keylen>.
//
// A phase 2 encryption row is exactly {name, keylen}, so the key length is the
// only thing separating the AES-128 and AES-256 proposals that are routinely
// configured side by side; keying on name alone would collide them.
// ---------------------------------------------------------------------------

type ipsecPhase2EncryptionResource struct{ client *client.Client }

var _ resource.Resource = (*ipsecPhase2EncryptionResource)(nil)
var _ resource.ResourceWithConfigure = (*ipsecPhase2EncryptionResource)(nil)
var _ resource.ResourceWithImportState = (*ipsecPhase2EncryptionResource)(nil)

const (
	ipsecPhase2EncryptionPlural   = "/api/v2/vpn/ipsec/phase2/encryptions"
	ipsecPhase2EncryptionSingular = "/api/v2/vpn/ipsec/phase2/encryption"
)

type ipsecPhase2EncryptionModel struct {
	ID       types.String `tfsdk:"id"`
	ParentID types.String `tfsdk:"parent_id"`
	Name     types.String `tfsdk:"name"`
	Keylen   types.Int64  `tfsdk:"keylen"`
}

func NewIPsecPhase2EncryptionResource() resource.Resource { return &ipsecPhase2EncryptionResource{} }

func (r *ipsecPhase2EncryptionResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_ipsec_phase2_encryption"
}
func (r *ipsecPhase2EncryptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *ipsecPhase2EncryptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an encryption proposal on an IPsec phase 2 entry. Identified by the parent phase 2 description and the algorithm name and key length.",
		Attributes: map[string]schema.Attribute{
			"id":        computedIDAttribute(),
			"parent_id": parentIDAttribute("The `descr` of the parent IPsec phase 2 entry."),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the encryption algorithm to use for this P2 encryption item.",
				Validators: []validator.String{
					stringvalidator.OneOf("aes", "aes128gcm", "aes192gcm", "aes256gcm", "chacha20poly1305"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keylen": schema.Int64Attribute{
				Optional:    true,
				Description: "The key length for the encryption algorithm. Required for variable key length algorithms. Part of this item's identity, so changing it replaces the resource.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ipsecPhase2EncryptionResource) key(parent, name string, keylen int64) string {
	return fmt.Sprintf("%s|%s|%d", parent, name, keylen)
}

// identity returns the parent descr and the composite child key, preferring the
// state/plan attributes and falling back to the resource ID (set on import).
func (r *ipsecPhase2EncryptionResource) identity(m ipsecPhase2EncryptionModel) (parent, name string, keylen int64) {
	parent = m.ParentID.ValueString()
	name = m.Name.ValueString()
	keylen = m.Keylen.ValueInt64()
	if parent == "" {
		parts := splitKeyN(m.ID.ValueString(), 3)
		parent, name, keylen = parts[0], parts[1], atoi64(parts[2])
	}
	return parent, name, keylen
}

// find locates the proposal row within its parent phase 2. As on phase 1,
// keylen only narrows the match when it is set, since fixed-key-length
// algorithms carry no key length.
func (r *ipsecPhase2EncryptionResource) find(ctx context.Context, pid any, name string, keylen int64) (any, map[string]any, bool, error) {
	filters := map[string]string{"name": name}
	if keylen != 0 {
		filters["keylen"] = formatID(keylen)
	}
	return findByKeysInParent(ctx, r.client, ipsecPhase2EncryptionPlural, formatID(pid), filters)
}

func (r *ipsecPhase2EncryptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipsecPhase2EncryptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, name, keylen := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase2Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 2 entry", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "name", plan.Name)
	setInt(payload, "keylen", plan.Keylen)
	if _, err := r.client.Create(ctx, ipsecPhase2EncryptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create IPsec phase 2 encryption item", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, name, keylen))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase2EncryptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipsecPhase2EncryptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, name, keylen := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase2Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 2 entry", err.Error())
		return
	}
	_, _, found, err := r.find(ctx, pid, name, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to read IPsec phase 2 encryption item", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parent, name, keylen))
	state.ParentID = types.StringValue(parent)
	state.Name = types.StringValue(name)
	state.Keylen = keylenValue(keylen)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipsecPhase2EncryptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipsecPhase2EncryptionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, name, keylen := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase2Plural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent IPsec phase 2 entry", err.Error())
		return
	}
	id, _, found, err := r.find(ctx, pid, name, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 2 encryption item", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("IPsec phase 2 encryption item not found", "encryption item "+r.key(parent, name, keylen)+" no longer exists")
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "name", plan.Name)
	setInt(payload, "keylen", plan.Keylen)
	if _, err := r.client.Update(ctx, ipsecPhase2EncryptionSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update IPsec phase 2 encryption item", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, name, keylen))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipsecPhase2EncryptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipsecPhase2EncryptionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, name, keylen := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, ipsecPhase2Plural, "descr", parent)
	if err != nil {
		// Parent phase 2 entry removed out-of-band: the child is already gone.
		return
	}
	id, _, found, err := r.find(ctx, pid, name, keylen)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 2 encryption item", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, ipsecPhase2EncryptionSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete IPsec phase 2 encryption item", err.Error())
	}
}

func (r *ipsecPhase2EncryptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_openvpn_client
// Key: description (the OpenVPN models use `description`, not `descr`).
// ---------------------------------------------------------------------------

type openVPNClientResource struct{ client *client.Client }

var _ resource.Resource = (*openVPNClientResource)(nil)
var _ resource.ResourceWithConfigure = (*openVPNClientResource)(nil)
var _ resource.ResourceWithImportState = (*openVPNClientResource)(nil)

const (
	openVPNClientPlural   = "/api/v2/vpn/openvpn/clients"
	openVPNClientSingular = "/api/v2/vpn/openvpn/client"
)

type openVPNClientModel struct {
	ID                  types.String `tfsdk:"id"`
	VPNID               types.Int64  `tfsdk:"vpnid"`
	VPNIf               types.String `tfsdk:"vpnif"`
	Description         types.String `tfsdk:"description"`
	Disable             types.Bool   `tfsdk:"disable"`
	Mode                types.String `tfsdk:"mode"`
	DevMode             types.String `tfsdk:"dev_mode"`
	Protocol            types.String `tfsdk:"protocol"`
	Interface           types.String `tfsdk:"interface"`
	ServerAddr          types.String `tfsdk:"server_addr"`
	ServerPort          types.String `tfsdk:"server_port"`
	LocalPort           types.String `tfsdk:"local_port"`
	ProxyAddr           types.String `tfsdk:"proxy_addr"`
	ProxyPort           types.String `tfsdk:"proxy_port"`
	ProxyAuthtype       types.String `tfsdk:"proxy_authtype"`
	ProxyUser           types.String `tfsdk:"proxy_user"`
	ProxyPasswd         types.String `tfsdk:"proxy_passwd"`
	AuthUser            types.String `tfsdk:"auth_user"`
	AuthPass            types.String `tfsdk:"auth_pass"`
	AuthRetryNone       types.Bool   `tfsdk:"auth_retry_none"`
	TLS                 types.String `tfsdk:"tls"`
	TLSType             types.String `tfsdk:"tls_type"`
	TLSAuthKeydir       types.String `tfsdk:"tlsauth_keydir"`
	Caref               types.String `tfsdk:"caref"`
	Certref             types.String `tfsdk:"certref"`
	DataCiphers         types.List   `tfsdk:"data_ciphers"`
	DataCiphersFallback types.String `tfsdk:"data_ciphers_fallback"`
	Digest              types.String `tfsdk:"digest"`
	RemoteCertTLS       types.Bool   `tfsdk:"remote_cert_tls"`
	TunnelNetwork       types.String `tfsdk:"tunnel_network"`
	TunnelNetworkV6     types.String `tfsdk:"tunnel_networkv6"`
	RemoteNetwork       types.List   `tfsdk:"remote_network"`
	RemoteNetworkV6     types.List   `tfsdk:"remote_networkv6"`
	UseShaper           types.Int64  `tfsdk:"use_shaper"`
	AllowCompression    types.String `tfsdk:"allow_compression"`
	Passtos             types.Bool   `tfsdk:"passtos"`
	RouteNoPull         types.Bool   `tfsdk:"route_no_pull"`
	RouteNoExec         types.Bool   `tfsdk:"route_no_exec"`
	DNSAdd              types.Bool   `tfsdk:"dns_add"`
	Topology            types.String `tfsdk:"topology"`
	InactiveSeconds     types.Int64  `tfsdk:"inactive_seconds"`
	PingMethod          types.String `tfsdk:"ping_method"`
	KeepaliveInterval   types.Int64  `tfsdk:"keepalive_interval"`
	KeepaliveTimeout    types.Int64  `tfsdk:"keepalive_timeout"`
	PingSeconds         types.Int64  `tfsdk:"ping_seconds"`
	PingAction          types.String `tfsdk:"ping_action"`
	PingActionSeconds   types.Int64  `tfsdk:"ping_action_seconds"`
	CustomOptions       types.List   `tfsdk:"custom_options"`
	UDPFastIO           types.Bool   `tfsdk:"udp_fast_io"`
	ExitNotify          types.String `tfsdk:"exit_notify"`
	Sndrcvbuf           types.Int64  `tfsdk:"sndrcvbuf"`
	CreateGw            types.String `tfsdk:"create_gw"`
	VerbosityLevel      types.Int64  `tfsdk:"verbosity_level"`
}

func NewOpenVPNClientResource() resource.Resource { return &openVPNClientResource{} }

func (r *openVPNClientResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_openvpn_client"
}
func (r *openVPNClientResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *openVPNClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenVPN client instance. Identified by its `description`.",
		Attributes: map[string]schema.Attribute{
			"id":                    computedIDAttribute(),
			"vpnid":                 computedIntAttribute("The unique ID for this OpenVPN client, assigned by the system."),
			"vpnif":                 computedStringAttribute("The VPN interface name for this OpenVPN client, assigned by the system."),
			"description":           keyAttribute("The description for this OpenVPN client. Unique; immutable after creation."),
			"disable":               optionalBoolAttribute("Disable this OpenVPN client."),
			"mode":                  enumAttribute("The OpenVPN client mode. Defaults to p2p_tls when unset.", "p2p_tls"),
			"dev_mode":              requiredEnumAttribute("The carrier mode for this OpenVPN client: `tun` (layer 3) or `tap` (layer 2).", "tun", "tap"),
			"protocol":              requiredEnumAttribute("The protocol used by this OpenVPN client.", "UDP4", "UDP6", "UDP", "TCP4", "TCP6", "TCP"),
			"interface":             optionalStringAttribute("The interface used by the firewall to originate this OpenVPN client connection."),
			"server_addr":           requiredStringAttribute("The IP address or hostname of the OpenVPN server this client will connect to.", isIPAddressOrHostname()),
			"server_port":           requiredStringAttribute("The port used by the server to receive client connections.", isPort()),
			"local_port":            optionalStringAttribute("The port binding used by OpenVPN for client connections.", isPort()),
			"proxy_addr":            optionalStringAttribute("The address of an HTTP proxy this client can use to connect to the remote server."),
			"proxy_port":            optionalStringAttribute("The port used by the HTTP proxy."),
			"proxy_authtype":        enumAttribute("The type of authentication used by the proxy server.", "none", "basic", "ntlm"),
			"proxy_user":            optionalStringAttribute("The username to use for authentication to the remote proxy."),
			"proxy_passwd":          sensitiveStringAttribute("The password to use for authentication to the remote proxy."),
			"auth_user":             optionalStringAttribute("The username used to authenticate with the OpenVPN server."),
			"auth_pass":             sensitiveStringAttribute("The password used to authenticate with the OpenVPN server."),
			"auth_retry_none":       optionalBoolAttribute("Do not retry authentication when the server returns an authentication failure."),
			"tls":                   sensitiveStringAttribute("The TLS key used to sign control channel packets with an HMAC signature."),
			"tls_type":              enumAttribute("The TLS key usage type: `auth` (HMAC authentication) or `crypt` (authentication and encryption).", "auth", "crypt"),
			"tlsauth_keydir":        enumAttribute("The TLS key direction. Must be set to complementary values on the client and server.", "default", "0", "1", "2"),
			"caref":                 requiredStringAttribute("The `refid` of the CA object to assume as the peer CA."),
			"certref":               optionalStringAttribute("The `refid` of the certificate object to assume as the OpenVPN client certificate."),
			"data_ciphers":          requiredStringListAttribute("The encryption algorithms/ciphers allowed by this OpenVPN client."),
			"data_ciphers_fallback": requiredStringAttribute("The fallback encryption algorithm/cipher used for data channel packets."),
			"digest":                requiredStringAttribute("The algorithm used to authenticate data channel packets."),
			"remote_cert_tls":       optionalBoolAttribute("Require the server to have a server certificate to connect."),
			"tunnel_network":        optionalStringAttribute("The IPv4 virtual network used for private communications between this client and the server.", isCIDR()),
			"tunnel_networkv6":      optionalStringAttribute("The IPv6 virtual network used for private communications between this client and the server.", isCIDR()),
			"remote_network":        optionalStringListAttribute("IPv4 networks that will be routed through the tunnel.", listvalidator.ValueStringsAre(isCIDR())),
			"remote_networkv6":      optionalStringListAttribute("IPv6 networks that will be routed through the tunnel.", listvalidator.ValueStringsAre(isCIDR())),
			"use_shaper":            optionalIntAttribute("Maximum outgoing bandwidth (in bytes per second) for this tunnel."),
			"allow_compression":     enumAttribute("The compression mode allowed by this OpenVPN client.", "no", "yes", "asym"),
			"passtos":               optionalBoolAttribute("Set the TOS IP header value of tunnel packets to match the encapsulated packet value."),
			"route_no_pull":         optionalBoolAttribute("Do not accept routes pushed by the server."),
			"route_no_exec":         optionalBoolAttribute("Do not add or remove routes automatically."),
			"dns_add":               optionalBoolAttribute("Use the DNS server(s) provided by the OpenVPN server."),
			"topology":              enumAttribute("The method used to supply a virtual adapter IP address to clients when using TUN mode on IPv4.", "subnet", "net30"),
			"inactive_seconds":      optionalIntAttribute("The amount of time (in seconds) until a connection is closed for inactivity."),
			"ping_method":           enumAttribute("The method used to define ping configuration.", "keepalive", "ping"),
			"keepalive_interval":    optionalIntAttribute("The keepalive interval parameter."),
			"keepalive_timeout":     optionalIntAttribute("The keepalive timeout parameter."),
			"ping_seconds":          optionalIntAttribute("The number of seconds to accept no packets before sending a ping to the remote peer."),
			"ping_action":           enumAttribute("The action to take after a ping to the remote peer times out.", "ping_restart", "ping_exit"),
			"ping_action_seconds":   optionalIntAttribute("The number of seconds that must elapse before the ping is considered a timeout."),
			"custom_options":        optionalStringListAttribute("Additional options to add to the OpenVPN client configuration."),
			"udp_fast_io":           optionalBoolAttribute("Use fast I/O operations with UDP writes to tun/tap (experimental)."),
			"exit_notify":           enumAttribute("The number of times this client will attempt to send exit notifications.", "1", "2", "3", "4", "5", "none"),
			"sndrcvbuf":             optionalIntAttribute("The send and receive buffer size for OpenVPN. Leave unset to use the system default."),
			"create_gw":             enumAttribute("The gateway type(s) created when a virtual interface is assigned to this OpenVPN client.", "both", "v4only", "v6only"),
			"verbosity_level":       optionalIntAttribute("The OpenVPN logging verbosity level (0-11)."),
		},
	}
}

func (r *openVPNClientResource) payload(m openVPNClientModel) map[string]any {
	p := map[string]any{}
	setString(p, "description", m.Description)
	setBool(p, "disable", m.Disable)
	setString(p, "mode", m.Mode)
	setString(p, "dev_mode", m.DevMode)
	setString(p, "protocol", m.Protocol)
	setString(p, "interface", m.Interface)
	setString(p, "server_addr", m.ServerAddr)
	setString(p, "server_port", m.ServerPort)
	setString(p, "local_port", m.LocalPort)
	setString(p, "proxy_addr", m.ProxyAddr)
	setString(p, "proxy_port", m.ProxyPort)
	setString(p, "proxy_authtype", m.ProxyAuthtype)
	setString(p, "proxy_user", m.ProxyUser)
	setString(p, "proxy_passwd", m.ProxyPasswd)
	setString(p, "auth_user", m.AuthUser)
	setString(p, "auth_pass", m.AuthPass)
	setBool(p, "auth_retry_none", m.AuthRetryNone)
	setString(p, "tls", m.TLS)
	setString(p, "tls_type", m.TLSType)
	setString(p, "tlsauth_keydir", m.TLSAuthKeydir)
	setString(p, "caref", m.Caref)
	setString(p, "certref", m.Certref)
	setStringList(p, "data_ciphers", m.DataCiphers)
	setString(p, "data_ciphers_fallback", m.DataCiphersFallback)
	setString(p, "digest", m.Digest)
	setBool(p, "remote_cert_tls", m.RemoteCertTLS)
	setString(p, "tunnel_network", m.TunnelNetwork)
	setString(p, "tunnel_networkv6", m.TunnelNetworkV6)
	setStringList(p, "remote_network", m.RemoteNetwork)
	setStringList(p, "remote_networkv6", m.RemoteNetworkV6)
	setInt(p, "use_shaper", m.UseShaper)
	setString(p, "allow_compression", m.AllowCompression)
	setBool(p, "passtos", m.Passtos)
	setBool(p, "route_no_pull", m.RouteNoPull)
	setBool(p, "route_no_exec", m.RouteNoExec)
	setBool(p, "dns_add", m.DNSAdd)
	setString(p, "topology", m.Topology)
	setInt(p, "inactive_seconds", m.InactiveSeconds)
	setString(p, "ping_method", m.PingMethod)
	setInt(p, "keepalive_interval", m.KeepaliveInterval)
	setInt(p, "keepalive_timeout", m.KeepaliveTimeout)
	setInt(p, "ping_seconds", m.PingSeconds)
	setString(p, "ping_action", m.PingAction)
	setInt(p, "ping_action_seconds", m.PingActionSeconds)
	setStringList(p, "custom_options", m.CustomOptions)
	setBool(p, "udp_fast_io", m.UDPFastIO)
	setString(p, "exit_notify", m.ExitNotify)
	setInt(p, "sndrcvbuf", m.Sndrcvbuf)
	setString(p, "create_gw", m.CreateGw)
	setInt(p, "verbosity_level", m.VerbosityLevel)
	return p
}

func (r *openVPNClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan openVPNClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, openVPNClientSingular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create OpenVPN client", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create OpenVPN client", err.Error())
		return
	}
	plan.VPNID = intValue(getInt(obj, "vpnid"))
	plan.VPNIf = strValue(getString(obj, "vpnif"))
	plan.ID = types.StringValue(plan.Description.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state openVPNClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := state.Description.ValueString()
	if description == "" {
		description = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, openVPNClientPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to read OpenVPN client", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(description)
	state.VPNID = intValue(getInt(obj, "vpnid"))
	state.VPNIf = strValue(getString(obj, "vpnif"))
	state.Description = types.StringValue(description)
	state.Disable = boolValue(getBool(obj, "disable"))
	state.Mode = strValue(getString(obj, "mode"))
	state.DevMode = strValue(getString(obj, "dev_mode"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.Interface = strValue(getString(obj, "interface"))
	state.ServerAddr = strValue(getString(obj, "server_addr"))
	state.ServerPort = strValue(getString(obj, "server_port"))
	state.LocalPort = strValue(getString(obj, "local_port"))
	state.ProxyAddr = strValue(getString(obj, "proxy_addr"))
	state.ProxyPort = strValue(getString(obj, "proxy_port"))
	state.ProxyAuthtype = strValue(getString(obj, "proxy_authtype"))
	state.ProxyUser = strValue(getString(obj, "proxy_user"))
	state.ProxyPasswd = strValue(getString(obj, "proxy_passwd"))
	state.AuthUser = strValue(getString(obj, "auth_user"))
	state.AuthPass = strValue(getString(obj, "auth_pass"))
	state.AuthRetryNone = boolValue(getBool(obj, "auth_retry_none"))
	state.TLS = strValue(getString(obj, "tls"))
	state.TLSType = strValue(getString(obj, "tls_type"))
	state.TLSAuthKeydir = strValue(getString(obj, "tlsauth_keydir"))
	state.Caref = strValue(getString(obj, "caref"))
	state.Certref = strValue(getString(obj, "certref"))
	state.DataCiphers = strListValue(ctx, getStringSlice(obj, "data_ciphers"))
	state.DataCiphersFallback = strValue(getString(obj, "data_ciphers_fallback"))
	state.Digest = strValue(getString(obj, "digest"))
	state.RemoteCertTLS = boolValue(getBool(obj, "remote_cert_tls"))
	state.TunnelNetwork = strValue(getString(obj, "tunnel_network"))
	state.TunnelNetworkV6 = strValue(getString(obj, "tunnel_networkv6"))
	state.RemoteNetwork = strListValue(ctx, getStringSlice(obj, "remote_network"))
	state.RemoteNetworkV6 = strListValue(ctx, getStringSlice(obj, "remote_networkv6"))
	state.UseShaper = intValue(getInt(obj, "use_shaper"))
	state.AllowCompression = strValue(getString(obj, "allow_compression"))
	state.Passtos = boolValue(getBool(obj, "passtos"))
	state.RouteNoPull = boolValue(getBool(obj, "route_no_pull"))
	state.RouteNoExec = boolValue(getBool(obj, "route_no_exec"))
	state.DNSAdd = boolValue(getBool(obj, "dns_add"))
	state.Topology = strValue(getString(obj, "topology"))
	state.InactiveSeconds = intValue(getInt(obj, "inactive_seconds"))
	state.PingMethod = strValue(getString(obj, "ping_method"))
	state.KeepaliveInterval = intValue(getInt(obj, "keepalive_interval"))
	state.KeepaliveTimeout = intValue(getInt(obj, "keepalive_timeout"))
	state.PingSeconds = intValue(getInt(obj, "ping_seconds"))
	state.PingAction = strValue(getString(obj, "ping_action"))
	state.PingActionSeconds = intValue(getInt(obj, "ping_action_seconds"))
	state.CustomOptions = strListValue(ctx, getStringSlice(obj, "custom_options"))
	state.UDPFastIO = boolValue(getBool(obj, "udp_fast_io"))
	state.ExitNotify = strValue(getString(obj, "exit_notify"))
	state.Sndrcvbuf = intValue(getInt(obj, "sndrcvbuf"))
	state.CreateGw = strValue(getString(obj, "create_gw"))
	state.VerbosityLevel = intValue(getInt(obj, "verbosity_level"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openVPNClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan openVPNClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := plan.Description.ValueString()
	if description == "" {
		description = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNClientPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN client", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("OpenVPN client not found", "client "+description+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, openVPNClientSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN client", err.Error())
		return
	}
	plan.ID = types.StringValue(description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state openVPNClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := state.Description.ValueString()
	if description == "" {
		description = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNClientPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN client", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, openVPNClientSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN client", err.Error())
	}
}

func (r *openVPNClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_openvpn_server
// Key: description.
// ---------------------------------------------------------------------------

type openVPNServerResource struct{ client *client.Client }

var _ resource.Resource = (*openVPNServerResource)(nil)
var _ resource.ResourceWithConfigure = (*openVPNServerResource)(nil)
var _ resource.ResourceWithImportState = (*openVPNServerResource)(nil)

const (
	openVPNServerPlural   = "/api/v2/vpn/openvpn/servers"
	openVPNServerSingular = "/api/v2/vpn/openvpn/server"
)

type openVPNServerModel struct {
	ID                    types.String `tfsdk:"id"`
	VPNID                 types.Int64  `tfsdk:"vpnid"`
	VPNIf                 types.String `tfsdk:"vpnif"`
	Description           types.String `tfsdk:"description"`
	Disable               types.Bool   `tfsdk:"disable"`
	Mode                  types.String `tfsdk:"mode"`
	Authmode              types.List   `tfsdk:"authmode"`
	DevMode               types.String `tfsdk:"dev_mode"`
	Protocol              types.String `tfsdk:"protocol"`
	Interface             types.String `tfsdk:"interface"`
	LocalPort             types.String `tfsdk:"local_port"`
	UseTLS                types.Bool   `tfsdk:"use_tls"`
	TLS                   types.String `tfsdk:"tls"`
	TLSType               types.String `tfsdk:"tls_type"`
	TLSAuthKeydir         types.String `tfsdk:"tlsauth_keydir"`
	Caref                 types.String `tfsdk:"caref"`
	Certref               types.String `tfsdk:"certref"`
	CertDepth             types.Int64  `tfsdk:"cert_depth"`
	DHLength              types.String `tfsdk:"dh_length"`
	ECDHCurve             types.String `tfsdk:"ecdh_curve"`
	DataCiphers           types.List   `tfsdk:"data_ciphers"`
	DataCiphersFallback   types.String `tfsdk:"data_ciphers_fallback"`
	Digest                types.String `tfsdk:"digest"`
	StrictUserCN          types.Bool   `tfsdk:"strictusercn"`
	RemoteCertTLS         types.Bool   `tfsdk:"remote_cert_tls"`
	TunnelNetwork         types.String `tfsdk:"tunnel_network"`
	TunnelNetworkV6       types.String `tfsdk:"tunnel_networkv6"`
	ServerbridgeDHCP      types.Bool   `tfsdk:"serverbridge_dhcp"`
	ServerbridgeInterface types.String `tfsdk:"serverbridge_interface"`
	ServerbridgeRouteGw   types.Bool   `tfsdk:"serverbridge_routegateway"`
	ServerbridgeDHCPStart types.String `tfsdk:"serverbridge_dhcp_start"`
	ServerbridgeDHCPEnd   types.String `tfsdk:"serverbridge_dhcp_end"`
	Gwredir               types.Bool   `tfsdk:"gwredir"`
	Gwredir6              types.Bool   `tfsdk:"gwredir6"`
	LocalNetwork          types.List   `tfsdk:"local_network"`
	LocalNetworkV6        types.List   `tfsdk:"local_networkv6"`
	RemoteNetwork         types.List   `tfsdk:"remote_network"`
	RemoteNetworkV6       types.List   `tfsdk:"remote_networkv6"`
	Maxclients            types.Int64  `tfsdk:"maxclients"`
	AllowCompression      types.String `tfsdk:"allow_compression"`
	Passtos               types.Bool   `tfsdk:"passtos"`
	Client2Client         types.Bool   `tfsdk:"client2client"`
	DuplicateCN           types.Bool   `tfsdk:"duplicate_cn"`
	Connlimit             types.Int64  `tfsdk:"connlimit"`
	DynamicIP             types.Bool   `tfsdk:"dynamic_ip"`
	Topology              types.String `tfsdk:"topology"`
	InactiveSeconds       types.Int64  `tfsdk:"inactive_seconds"`
	PingMethod            types.String `tfsdk:"ping_method"`
	KeepaliveInterval     types.Int64  `tfsdk:"keepalive_interval"`
	KeepaliveTimeout      types.Int64  `tfsdk:"keepalive_timeout"`
	PingSeconds           types.Int64  `tfsdk:"ping_seconds"`
	PingPush              types.Bool   `tfsdk:"ping_push"`
	PingAction            types.String `tfsdk:"ping_action"`
	PingActionSeconds     types.Int64  `tfsdk:"ping_action_seconds"`
	PingActionPush        types.Bool   `tfsdk:"ping_action_push"`
	DNSDomain             types.String `tfsdk:"dns_domain"`
	DNSServer1            types.String `tfsdk:"dns_server1"`
	DNSServer2            types.String `tfsdk:"dns_server2"`
	DNSServer3            types.String `tfsdk:"dns_server3"`
	DNSServer4            types.String `tfsdk:"dns_server4"`
	PushBlockOutsideDNS   types.Bool   `tfsdk:"push_blockoutsidedns"`
	PushRegisterDNS       types.Bool   `tfsdk:"push_register_dns"`
	NTPServer1            types.String `tfsdk:"ntp_server1"`
	NTPServer2            types.String `tfsdk:"ntp_server2"`
	NetbiosEnable         types.Bool   `tfsdk:"netbios_enable"`
	NetbiosNtype          types.Int64  `tfsdk:"netbios_ntype"`
	NetbiosScope          types.String `tfsdk:"netbios_scope"`
	WINSServer1           types.String `tfsdk:"wins_server1"`
	WINSServer2           types.String `tfsdk:"wins_server2"`
	CustomOptions         types.List   `tfsdk:"custom_options"`
	UsernameAsCommonName  types.Bool   `tfsdk:"username_as_common_name"`
	Sndrcvbuf             types.Int64  `tfsdk:"sndrcvbuf"`
	CreateGw              types.String `tfsdk:"create_gw"`
	VerbosityLevel        types.Int64  `tfsdk:"verbosity_level"`
}

func NewOpenVPNServerResource() resource.Resource { return &openVPNServerResource{} }

func (r *openVPNServerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_openvpn_server"
}
func (r *openVPNServerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *openVPNServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenVPN server instance. Identified by its `description`.",
		Attributes: map[string]schema.Attribute{
			"id":                        computedIDAttribute(),
			"vpnid":                     computedIntAttribute("The unique ID for this OpenVPN server, assigned by the system."),
			"vpnif":                     computedStringAttribute("The VPN interface name for this OpenVPN server, assigned by the system."),
			"description":               keyAttribute("The description for this OpenVPN server. Unique; immutable after creation."),
			"disable":                   optionalBoolAttribute("Disable this OpenVPN server."),
			"mode":                      requiredEnumAttribute("The OpenVPN server mode.", "p2p_tls", "server_tls", "server_user", "server_tls_user"),
			"authmode":                  optionalStringListAttribute("The authentication servers used as the authentication backend for this OpenVPN server."),
			"dev_mode":                  requiredEnumAttribute("The carrier mode for this OpenVPN server: `tun` (layer 3) or `tap` (layer 2).", "tun", "tap"),
			"protocol":                  requiredEnumAttribute("The protocol used by this OpenVPN server.", "UDP4", "UDP6", "UDP", "TCP4", "TCP6", "TCP"),
			"interface":                 optionalStringAttribute("The interface or virtual IP address where OpenVPN will receive client connections."),
			"local_port":                optionalStringAttribute("The port used by OpenVPN to receive client connections.", isPort()),
			"use_tls":                   optionalBoolAttribute("Use a TLS key for this OpenVPN server."),
			"tls":                       sensitiveStringAttribute("The TLS key used to sign control channel packets with an HMAC signature."),
			"tls_type":                  enumAttribute("The TLS key usage type: `auth` (HMAC authentication) or `crypt` (authentication and encryption).", "auth", "crypt"),
			"tlsauth_keydir":            enumAttribute("The TLS key direction. Must be set to complementary values on the client and server.", "default", "0", "1", "2"),
			"caref":                     requiredStringAttribute("The `refid` of the CA object to assume as the peer CA."),
			"certref":                   requiredStringAttribute("The `refid` of the certificate object to assume as the OpenVPN server certificate."),
			"cert_depth":                optionalIntAttribute("The depth of the certificate chain to check when a certificate based client signs in."),
			"dh_length":                 requiredStringAttribute("The Diffie-Hellman (DH) parameter set used for key exchange."),
			"ecdh_curve":                requiredStringAttribute("The elliptic curve to use for key exchange."),
			"data_ciphers":              requiredStringListAttribute("The encryption algorithms/ciphers allowed by this OpenVPN server."),
			"data_ciphers_fallback":     requiredStringAttribute("The fallback encryption algorithm/cipher used for data channel packets."),
			"digest":                    requiredStringAttribute("The algorithm used to authenticate data channel packets."),
			"strictusercn":              optionalBoolAttribute("Require a match between the common name of the client certificate and the username."),
			"remote_cert_tls":           optionalBoolAttribute("Require hosts to have a client certificate to connect."),
			"tunnel_network":            optionalStringAttribute("The IPv4 virtual network used for private communications between this server and client hosts.", isCIDR()),
			"tunnel_networkv6":          optionalStringAttribute("The IPv6 virtual network used for private communications between this server and client hosts.", isCIDR()),
			"serverbridge_dhcp":         optionalBoolAttribute("Allow clients on the bridge to obtain DHCP."),
			"serverbridge_interface":    optionalStringAttribute("The interface this TAP instance will be bridged to."),
			"serverbridge_routegateway": optionalBoolAttribute("Push the bridge interface's IPv4 address to connecting clients as a route gateway."),
			"serverbridge_dhcp_start":   optionalStringAttribute("The bridge DHCP range's start address."),
			"serverbridge_dhcp_end":     optionalStringAttribute("The bridge DHCP range's end address."),
			"gwredir":                   optionalBoolAttribute("Force all client-generated IPv4 traffic through the tunnel."),
			"gwredir6":                  optionalBoolAttribute("Force all client-generated IPv6 traffic through the tunnel."),
			"local_network":             optionalStringListAttribute("The IPv4 networks that will be accessible from the remote endpoint.", listvalidator.ValueStringsAre(isCIDR())),
			"local_networkv6":           optionalStringListAttribute("The IPv6 networks that will be accessible from the remote endpoint.", listvalidator.ValueStringsAre(isCIDR())),
			"remote_network":            optionalStringListAttribute("IPv4 networks that will be routed through the tunnel.", listvalidator.ValueStringsAre(isCIDR())),
			"remote_networkv6":          optionalStringListAttribute("IPv6 networks that will be routed through the tunnel.", listvalidator.ValueStringsAre(isCIDR())),
			"maxclients":                optionalIntAttribute("The maximum number of clients allowed to concurrently connect to this server."),
			"allow_compression":         enumAttribute("The compression mode allowed by this OpenVPN server.", "no", "yes", "asym"),
			"passtos":                   optionalBoolAttribute("Set the TOS IP header value of tunnel packets to match the encapsulated packet value."),
			"client2client":             optionalBoolAttribute("Allow communication between clients connected to this server."),
			"duplicate_cn":              optionalBoolAttribute("Allow the same user to connect multiple times."),
			"connlimit":                 optionalIntAttribute("The number of concurrent connections a single user can have."),
			"dynamic_ip":                optionalBoolAttribute("Allow connected clients to retain their connections if their IP address changes."),
			"topology":                  enumAttribute("The method used to supply a virtual adapter IP address to clients when using TUN mode on IPv4.", "subnet", "net30"),
			"inactive_seconds":          optionalIntAttribute("The amount of time (in seconds) until a client connection is closed for inactivity."),
			"ping_method":               enumAttribute("The method used to define ping configuration.", "keepalive", "ping"),
			"keepalive_interval":        optionalIntAttribute("The keepalive interval parameter."),
			"keepalive_timeout":         optionalIntAttribute("The keepalive timeout parameter."),
			"ping_seconds":              optionalIntAttribute("The number of seconds to accept no packets before sending a ping to the remote peer."),
			"ping_push":                 optionalBoolAttribute("Push the ping configuration to the VPN client."),
			"ping_action":               enumAttribute("The action to take after a ping to the remote peer times out.", "ping_restart", "ping_exit"),
			"ping_action_seconds":       optionalIntAttribute("The number of seconds that must elapse before the ping is considered a timeout."),
			"ping_action_push":          optionalBoolAttribute("Push the ping action to the VPN client."),
			"dns_domain":                optionalStringAttribute("The default domain to provide to clients."),
			"dns_server1":               optionalStringAttribute("The primary DNS server to provide to clients."),
			"dns_server2":               optionalStringAttribute("The secondary DNS server to provide to clients."),
			"dns_server3":               optionalStringAttribute("The tertiary DNS server to provide to clients."),
			"dns_server4":               optionalStringAttribute("The quaternary DNS server to provide to clients."),
			"push_blockoutsidedns":      optionalBoolAttribute("Block Windows 10 clients' access to DNS servers except across the OpenVPN tunnel."),
			"push_register_dns":         optionalBoolAttribute("Run `net stop dnscache`, `net start dnscache` and `ipconfig` DNS commands on connecting clients."),
			"ntp_server1":               optionalStringAttribute("The primary NTP server to provide to clients."),
			"ntp_server2":               optionalStringAttribute("The secondary NTP server to provide to clients."),
			"netbios_enable":            optionalBoolAttribute("Enable NetBIOS over TCP/IP."),
			"netbios_ntype":             optionalIntAttribute("The NetBIOS node type (0, 1, 2, 4 or 8)."),
			"netbios_scope":             optionalStringAttribute("The NetBIOS scope ID."),
			"wins_server1":              optionalStringAttribute("The primary WINS server to provide to clients."),
			"wins_server2":              optionalStringAttribute("The secondary WINS server to provide to clients."),
			"custom_options":            optionalStringListAttribute("Additional options to add to the OpenVPN server configuration."),
			"username_as_common_name":   optionalBoolAttribute("Use the username of the client in place of the certificate common name."),
			"sndrcvbuf":                 optionalIntAttribute("The send and receive buffer size for OpenVPN. Leave unset to use the system default."),
			"create_gw":                 enumAttribute("The gateway type(s) created when a virtual interface is assigned to this OpenVPN server.", "both", "v4only", "v6only"),
			"verbosity_level":           optionalIntAttribute("The OpenVPN logging verbosity level (0-11)."),
		},
	}
}

func (r *openVPNServerResource) payload(m openVPNServerModel) map[string]any {
	p := map[string]any{}
	setString(p, "description", m.Description)
	setBool(p, "disable", m.Disable)
	setString(p, "mode", m.Mode)
	setStringList(p, "authmode", m.Authmode)
	setString(p, "dev_mode", m.DevMode)
	setString(p, "protocol", m.Protocol)
	setString(p, "interface", m.Interface)
	setString(p, "local_port", m.LocalPort)
	setBool(p, "use_tls", m.UseTLS)
	setString(p, "tls", m.TLS)
	setString(p, "tls_type", m.TLSType)
	setString(p, "tlsauth_keydir", m.TLSAuthKeydir)
	setString(p, "caref", m.Caref)
	setString(p, "certref", m.Certref)
	setInt(p, "cert_depth", m.CertDepth)
	setString(p, "dh_length", m.DHLength)
	setString(p, "ecdh_curve", m.ECDHCurve)
	setStringList(p, "data_ciphers", m.DataCiphers)
	setString(p, "data_ciphers_fallback", m.DataCiphersFallback)
	setString(p, "digest", m.Digest)
	setBool(p, "strictusercn", m.StrictUserCN)
	setBool(p, "remote_cert_tls", m.RemoteCertTLS)
	setString(p, "tunnel_network", m.TunnelNetwork)
	setString(p, "tunnel_networkv6", m.TunnelNetworkV6)
	setBool(p, "serverbridge_dhcp", m.ServerbridgeDHCP)
	setString(p, "serverbridge_interface", m.ServerbridgeInterface)
	setBool(p, "serverbridge_routegateway", m.ServerbridgeRouteGw)
	setString(p, "serverbridge_dhcp_start", m.ServerbridgeDHCPStart)
	setString(p, "serverbridge_dhcp_end", m.ServerbridgeDHCPEnd)
	setBool(p, "gwredir", m.Gwredir)
	setBool(p, "gwredir6", m.Gwredir6)
	setStringList(p, "local_network", m.LocalNetwork)
	setStringList(p, "local_networkv6", m.LocalNetworkV6)
	setStringList(p, "remote_network", m.RemoteNetwork)
	setStringList(p, "remote_networkv6", m.RemoteNetworkV6)
	setInt(p, "maxclients", m.Maxclients)
	setString(p, "allow_compression", m.AllowCompression)
	setBool(p, "passtos", m.Passtos)
	setBool(p, "client2client", m.Client2Client)
	setBool(p, "duplicate_cn", m.DuplicateCN)
	setInt(p, "connlimit", m.Connlimit)
	setBool(p, "dynamic_ip", m.DynamicIP)
	setString(p, "topology", m.Topology)
	setInt(p, "inactive_seconds", m.InactiveSeconds)
	setString(p, "ping_method", m.PingMethod)
	setInt(p, "keepalive_interval", m.KeepaliveInterval)
	setInt(p, "keepalive_timeout", m.KeepaliveTimeout)
	setInt(p, "ping_seconds", m.PingSeconds)
	setBool(p, "ping_push", m.PingPush)
	setString(p, "ping_action", m.PingAction)
	setInt(p, "ping_action_seconds", m.PingActionSeconds)
	setBool(p, "ping_action_push", m.PingActionPush)
	setString(p, "dns_domain", m.DNSDomain)
	setString(p, "dns_server1", m.DNSServer1)
	setString(p, "dns_server2", m.DNSServer2)
	setString(p, "dns_server3", m.DNSServer3)
	setString(p, "dns_server4", m.DNSServer4)
	setBool(p, "push_blockoutsidedns", m.PushBlockOutsideDNS)
	setBool(p, "push_register_dns", m.PushRegisterDNS)
	setString(p, "ntp_server1", m.NTPServer1)
	setString(p, "ntp_server2", m.NTPServer2)
	setBool(p, "netbios_enable", m.NetbiosEnable)
	setInt(p, "netbios_ntype", m.NetbiosNtype)
	setString(p, "netbios_scope", m.NetbiosScope)
	setString(p, "wins_server1", m.WINSServer1)
	setString(p, "wins_server2", m.WINSServer2)
	setStringList(p, "custom_options", m.CustomOptions)
	setBool(p, "username_as_common_name", m.UsernameAsCommonName)
	setInt(p, "sndrcvbuf", m.Sndrcvbuf)
	setString(p, "create_gw", m.CreateGw)
	setInt(p, "verbosity_level", m.VerbosityLevel)
	return p
}

func (r *openVPNServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan openVPNServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, openVPNServerSingular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create OpenVPN server", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create OpenVPN server", err.Error())
		return
	}
	plan.VPNID = intValue(getInt(obj, "vpnid"))
	plan.VPNIf = strValue(getString(obj, "vpnif"))
	plan.ID = types.StringValue(plan.Description.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state openVPNServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := state.Description.ValueString()
	if description == "" {
		description = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, openVPNServerPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to read OpenVPN server", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(description)
	state.VPNID = intValue(getInt(obj, "vpnid"))
	state.VPNIf = strValue(getString(obj, "vpnif"))
	state.Description = types.StringValue(description)
	state.Disable = boolValue(getBool(obj, "disable"))
	state.Mode = strValue(getString(obj, "mode"))
	state.Authmode = strListValue(ctx, getStringSlice(obj, "authmode"))
	state.DevMode = strValue(getString(obj, "dev_mode"))
	state.Protocol = strValue(getString(obj, "protocol"))
	state.Interface = strValue(getString(obj, "interface"))
	state.LocalPort = strValue(getString(obj, "local_port"))
	state.UseTLS = boolValue(getBool(obj, "use_tls"))
	state.TLS = strValue(getString(obj, "tls"))
	state.TLSType = strValue(getString(obj, "tls_type"))
	state.TLSAuthKeydir = strValue(getString(obj, "tlsauth_keydir"))
	state.Caref = strValue(getString(obj, "caref"))
	state.Certref = strValue(getString(obj, "certref"))
	state.CertDepth = intValue(getInt(obj, "cert_depth"))
	state.DHLength = strValue(getString(obj, "dh_length"))
	state.ECDHCurve = strValue(getString(obj, "ecdh_curve"))
	state.DataCiphers = strListValue(ctx, getStringSlice(obj, "data_ciphers"))
	state.DataCiphersFallback = strValue(getString(obj, "data_ciphers_fallback"))
	state.Digest = strValue(getString(obj, "digest"))
	state.StrictUserCN = boolValue(getBool(obj, "strictusercn"))
	state.RemoteCertTLS = boolValue(getBool(obj, "remote_cert_tls"))
	state.TunnelNetwork = strValue(getString(obj, "tunnel_network"))
	state.TunnelNetworkV6 = strValue(getString(obj, "tunnel_networkv6"))
	state.ServerbridgeDHCP = boolValue(getBool(obj, "serverbridge_dhcp"))
	state.ServerbridgeInterface = strValue(getString(obj, "serverbridge_interface"))
	state.ServerbridgeRouteGw = boolValue(getBool(obj, "serverbridge_routegateway"))
	state.ServerbridgeDHCPStart = strValue(getString(obj, "serverbridge_dhcp_start"))
	state.ServerbridgeDHCPEnd = strValue(getString(obj, "serverbridge_dhcp_end"))
	state.Gwredir = boolValue(getBool(obj, "gwredir"))
	state.Gwredir6 = boolValue(getBool(obj, "gwredir6"))
	state.LocalNetwork = strListValue(ctx, getStringSlice(obj, "local_network"))
	state.LocalNetworkV6 = strListValue(ctx, getStringSlice(obj, "local_networkv6"))
	state.RemoteNetwork = strListValue(ctx, getStringSlice(obj, "remote_network"))
	state.RemoteNetworkV6 = strListValue(ctx, getStringSlice(obj, "remote_networkv6"))
	state.Maxclients = intValue(getInt(obj, "maxclients"))
	state.AllowCompression = strValue(getString(obj, "allow_compression"))
	state.Passtos = boolValue(getBool(obj, "passtos"))
	state.Client2Client = boolValue(getBool(obj, "client2client"))
	state.DuplicateCN = boolValue(getBool(obj, "duplicate_cn"))
	state.Connlimit = intValue(getInt(obj, "connlimit"))
	state.DynamicIP = boolValue(getBool(obj, "dynamic_ip"))
	state.Topology = strValue(getString(obj, "topology"))
	state.InactiveSeconds = intValue(getInt(obj, "inactive_seconds"))
	state.PingMethod = strValue(getString(obj, "ping_method"))
	state.KeepaliveInterval = intValue(getInt(obj, "keepalive_interval"))
	state.KeepaliveTimeout = intValue(getInt(obj, "keepalive_timeout"))
	state.PingSeconds = intValue(getInt(obj, "ping_seconds"))
	state.PingPush = boolValue(getBool(obj, "ping_push"))
	state.PingAction = strValue(getString(obj, "ping_action"))
	state.PingActionSeconds = intValue(getInt(obj, "ping_action_seconds"))
	state.PingActionPush = boolValue(getBool(obj, "ping_action_push"))
	state.DNSDomain = strValue(getString(obj, "dns_domain"))
	state.DNSServer1 = strValue(getString(obj, "dns_server1"))
	state.DNSServer2 = strValue(getString(obj, "dns_server2"))
	state.DNSServer3 = strValue(getString(obj, "dns_server3"))
	state.DNSServer4 = strValue(getString(obj, "dns_server4"))
	state.PushBlockOutsideDNS = boolValue(getBool(obj, "push_blockoutsidedns"))
	state.PushRegisterDNS = boolValue(getBool(obj, "push_register_dns"))
	state.NTPServer1 = strValue(getString(obj, "ntp_server1"))
	state.NTPServer2 = strValue(getString(obj, "ntp_server2"))
	state.NetbiosEnable = boolValue(getBool(obj, "netbios_enable"))
	state.NetbiosNtype = intValue(getInt(obj, "netbios_ntype"))
	state.NetbiosScope = strValue(getString(obj, "netbios_scope"))
	state.WINSServer1 = strValue(getString(obj, "wins_server1"))
	state.WINSServer2 = strValue(getString(obj, "wins_server2"))
	state.CustomOptions = strListValue(ctx, getStringSlice(obj, "custom_options"))
	state.UsernameAsCommonName = boolValue(getBool(obj, "username_as_common_name"))
	state.Sndrcvbuf = intValue(getInt(obj, "sndrcvbuf"))
	state.CreateGw = strValue(getString(obj, "create_gw"))
	state.VerbosityLevel = intValue(getInt(obj, "verbosity_level"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openVPNServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan openVPNServerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := plan.Description.ValueString()
	if description == "" {
		description = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNServerPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN server", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("OpenVPN server not found", "server "+description+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, openVPNServerSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN server", err.Error())
		return
	}
	plan.ID = types.StringValue(description)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state openVPNServerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	description := state.Description.ValueString()
	if description == "" {
		description = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNServerPlural, "description", description)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN server", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, openVPNServerSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN server", err.Error())
	}
}

func (r *openVPNServerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_openvpn_cso
// OpenVPN client specific override. Key: common_name.
// ---------------------------------------------------------------------------

type openVPNCSOResource struct{ client *client.Client }

var _ resource.Resource = (*openVPNCSOResource)(nil)
var _ resource.ResourceWithConfigure = (*openVPNCSOResource)(nil)
var _ resource.ResourceWithImportState = (*openVPNCSOResource)(nil)

const (
	openVPNCSOPlural   = "/api/v2/vpn/openvpn/csos"
	openVPNCSOSingular = "/api/v2/vpn/openvpn/cso"
)

type openVPNCSOModel struct {
	ID              types.String `tfsdk:"id"`
	CommonName      types.String `tfsdk:"common_name"`
	Disable         types.Bool   `tfsdk:"disable"`
	Block           types.Bool   `tfsdk:"block"`
	Description     types.String `tfsdk:"description"`
	ServerList      types.List   `tfsdk:"server_list"`
	TunnelNetwork   types.String `tfsdk:"tunnel_network"`
	TunnelNetworkV6 types.String `tfsdk:"tunnel_networkv6"`
	LocalNetwork    types.List   `tfsdk:"local_network"`
	LocalNetworkV6  types.List   `tfsdk:"local_networkv6"`
	RemoteNetwork   types.List   `tfsdk:"remote_network"`
	RemoteNetworkV6 types.List   `tfsdk:"remote_networkv6"`
	Gwredir         types.Bool   `tfsdk:"gwredir"`
	PushReset       types.Bool   `tfsdk:"push_reset"`
	RemoveOptions   types.List   `tfsdk:"remove_options"`
	DNSDomain       types.String `tfsdk:"dns_domain"`
	DNSServer1      types.String `tfsdk:"dns_server1"`
	DNSServer2      types.String `tfsdk:"dns_server2"`
	DNSServer3      types.String `tfsdk:"dns_server3"`
	DNSServer4      types.String `tfsdk:"dns_server4"`
	NTPServer1      types.String `tfsdk:"ntp_server1"`
	NTPServer2      types.String `tfsdk:"ntp_server2"`
	NetbiosEnable   types.Bool   `tfsdk:"netbios_enable"`
	NetbiosNtype    types.Int64  `tfsdk:"netbios_ntype"`
	NetbiosScope    types.String `tfsdk:"netbios_scope"`
	WINSServer1     types.String `tfsdk:"wins_server1"`
	WINSServer2     types.String `tfsdk:"wins_server2"`
	CustomOptions   types.List   `tfsdk:"custom_options"`
}

func NewOpenVPNCSOResource() resource.Resource { return &openVPNCSOResource{} }

func (r *openVPNCSOResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_openvpn_cso"
}
func (r *openVPNCSOResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *openVPNCSOResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenVPN client specific override. Identified by its `common_name`.",
		Attributes: map[string]schema.Attribute{
			"id":               computedIDAttribute(),
			"common_name":      keyAttribute("The X.509 common name of the client certificate, or the username for VPNs using password authentication. Unique; immutable after creation."),
			"disable":          optionalBoolAttribute("Disable this client specific override."),
			"block":            optionalBoolAttribute("Block this client from connecting to the server."),
			"description":      optionalStringAttribute("The description for this client specific override."),
			"server_list":      optionalStringListAttribute("The OpenVPN servers that will use this override. Leave unset to apply to all servers."),
			"tunnel_network":   optionalStringAttribute("The IPv4 virtual network used for private communications between the server and this client.", isCIDR()),
			"tunnel_networkv6": optionalStringAttribute("The IPv6 virtual network used for private communications between the server and this client.", isCIDR()),
			"local_network":    optionalStringListAttribute("The IPv4 server-side networks that will be accessible from this client.", listvalidator.ValueStringsAre(isCIDR())),
			"local_networkv6":  optionalStringListAttribute("The IPv6 server-side networks that will be accessible from this client.", listvalidator.ValueStringsAre(isCIDR())),
			"remote_network":   optionalStringListAttribute("The IPv4 client-side networks routed to this client using iroute.", listvalidator.ValueStringsAre(isCIDR())),
			"remote_networkv6": optionalStringListAttribute("The IPv6 client-side networks routed to this client using iroute.", listvalidator.ValueStringsAre(isCIDR())),
			"gwredir":          optionalBoolAttribute("Force all client-generated traffic through the tunnel."),
			"push_reset":       optionalBoolAttribute("Prevent this client from receiving any server-defined client settings."),
			"remove_options":   optionalStringListAttribute("The push-remove options to apply to this client."),
			"dns_domain":       optionalStringAttribute("The default domain to provide to the client."),
			"dns_server1":      optionalStringAttribute("The primary DNS server to provide to the client."),
			"dns_server2":      optionalStringAttribute("The secondary DNS server to provide to the client."),
			"dns_server3":      optionalStringAttribute("The tertiary DNS server to provide to the client."),
			"dns_server4":      optionalStringAttribute("The quaternary DNS server to provide to the client."),
			"ntp_server1":      optionalStringAttribute("The primary NTP server to provide to the client."),
			"ntp_server2":      optionalStringAttribute("The secondary NTP server to provide to the client."),
			"netbios_enable":   optionalBoolAttribute("Enable NetBIOS over TCP/IP."),
			"netbios_ntype":    optionalIntAttribute("The NetBIOS node type (0, 1, 2, 4 or 8)."),
			"netbios_scope":    optionalStringAttribute("The NetBIOS scope ID."),
			"wins_server1":     optionalStringAttribute("The primary WINS server to provide to the client."),
			"wins_server2":     optionalStringAttribute("The secondary WINS server to provide to the client."),
			"custom_options":   optionalStringListAttribute("Additional OpenVPN options to add for this client."),
		},
	}
}

func (r *openVPNCSOResource) payload(m openVPNCSOModel) map[string]any {
	p := map[string]any{}
	setString(p, "common_name", m.CommonName)
	setBool(p, "disable", m.Disable)
	setBool(p, "block", m.Block)
	setString(p, "description", m.Description)
	setStringList(p, "server_list", m.ServerList)
	setString(p, "tunnel_network", m.TunnelNetwork)
	setString(p, "tunnel_networkv6", m.TunnelNetworkV6)
	setStringList(p, "local_network", m.LocalNetwork)
	setStringList(p, "local_networkv6", m.LocalNetworkV6)
	setStringList(p, "remote_network", m.RemoteNetwork)
	setStringList(p, "remote_networkv6", m.RemoteNetworkV6)
	setBool(p, "gwredir", m.Gwredir)
	setBool(p, "push_reset", m.PushReset)
	setStringList(p, "remove_options", m.RemoveOptions)
	setString(p, "dns_domain", m.DNSDomain)
	setString(p, "dns_server1", m.DNSServer1)
	setString(p, "dns_server2", m.DNSServer2)
	setString(p, "dns_server3", m.DNSServer3)
	setString(p, "dns_server4", m.DNSServer4)
	setString(p, "ntp_server1", m.NTPServer1)
	setString(p, "ntp_server2", m.NTPServer2)
	setBool(p, "netbios_enable", m.NetbiosEnable)
	setInt(p, "netbios_ntype", m.NetbiosNtype)
	setString(p, "netbios_scope", m.NetbiosScope)
	setString(p, "wins_server1", m.WINSServer1)
	setString(p, "wins_server2", m.WINSServer2)
	setStringList(p, "custom_options", m.CustomOptions)
	return p
}

func (r *openVPNCSOResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan openVPNCSOModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, openVPNCSOSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create OpenVPN client specific override", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.CommonName.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNCSOResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state openVPNCSOModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	commonName := state.CommonName.ValueString()
	if commonName == "" {
		commonName = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, openVPNCSOPlural, "common_name", commonName)
	if err != nil {
		resp.Diagnostics.AddError("failed to read OpenVPN client specific override", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(commonName)
	state.CommonName = types.StringValue(commonName)
	state.Disable = boolValue(getBool(obj, "disable"))
	state.Block = boolValue(getBool(obj, "block"))
	state.Description = strValue(getString(obj, "description"))
	state.ServerList = strListValue(ctx, getStringSlice(obj, "server_list"))
	state.TunnelNetwork = strValue(getString(obj, "tunnel_network"))
	state.TunnelNetworkV6 = strValue(getString(obj, "tunnel_networkv6"))
	state.LocalNetwork = strListValue(ctx, getStringSlice(obj, "local_network"))
	state.LocalNetworkV6 = strListValue(ctx, getStringSlice(obj, "local_networkv6"))
	state.RemoteNetwork = strListValue(ctx, getStringSlice(obj, "remote_network"))
	state.RemoteNetworkV6 = strListValue(ctx, getStringSlice(obj, "remote_networkv6"))
	state.Gwredir = boolValue(getBool(obj, "gwredir"))
	state.PushReset = boolValue(getBool(obj, "push_reset"))
	state.RemoveOptions = strListValue(ctx, getStringSlice(obj, "remove_options"))
	state.DNSDomain = strValue(getString(obj, "dns_domain"))
	state.DNSServer1 = strValue(getString(obj, "dns_server1"))
	state.DNSServer2 = strValue(getString(obj, "dns_server2"))
	state.DNSServer3 = strValue(getString(obj, "dns_server3"))
	state.DNSServer4 = strValue(getString(obj, "dns_server4"))
	state.NTPServer1 = strValue(getString(obj, "ntp_server1"))
	state.NTPServer2 = strValue(getString(obj, "ntp_server2"))
	state.NetbiosEnable = boolValue(getBool(obj, "netbios_enable"))
	state.NetbiosNtype = intValue(getInt(obj, "netbios_ntype"))
	state.NetbiosScope = strValue(getString(obj, "netbios_scope"))
	state.WINSServer1 = strValue(getString(obj, "wins_server1"))
	state.WINSServer2 = strValue(getString(obj, "wins_server2"))
	state.CustomOptions = strListValue(ctx, getStringSlice(obj, "custom_options"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *openVPNCSOResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan openVPNCSOModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	commonName := plan.CommonName.ValueString()
	if commonName == "" {
		commonName = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNCSOPlural, "common_name", commonName)
	if err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN client specific override", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("OpenVPN client specific override not found", "override for "+commonName+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, openVPNCSOSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update OpenVPN client specific override", err.Error())
		return
	}
	plan.ID = types.StringValue(commonName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *openVPNCSOResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state openVPNCSOModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	commonName := state.CommonName.ValueString()
	if commonName == "" {
		commonName = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, openVPNCSOPlural, "common_name", commonName)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN client specific override", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, openVPNCSOSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete OpenVPN client specific override", err.Error())
	}
}

func (r *openVPNCSOResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_wireguard_tunnel
// Key: name. The tunnel name (wgN) is assigned by the system, so it is exposed
// as a computed attribute and captured from the create response. The
// `addresses` collection is managed by pfsense_wireguard_tunnel_address.
// ---------------------------------------------------------------------------

type wireGuardTunnelResource struct{ client *client.Client }

var _ resource.Resource = (*wireGuardTunnelResource)(nil)
var _ resource.ResourceWithConfigure = (*wireGuardTunnelResource)(nil)
var _ resource.ResourceWithImportState = (*wireGuardTunnelResource)(nil)

const (
	wireGuardTunnelPlural   = "/api/v2/vpn/wireguard/tunnels"
	wireGuardTunnelSingular = "/api/v2/vpn/wireguard/tunnel"
)

type wireGuardTunnelModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	Descr      types.String `tfsdk:"descr"`
	Listenport types.String `tfsdk:"listenport"`
	Publickey  types.String `tfsdk:"publickey"`
	Privatekey types.String `tfsdk:"privatekey"`
	MTU        types.Int64  `tfsdk:"mtu"`
}

func NewWireGuardTunnelResource() resource.Resource { return &wireGuardTunnelResource{} }

func (r *wireGuardTunnelResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_wireguard_tunnel"
}
func (r *wireGuardTunnelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *wireGuardTunnelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WireGuard tunnel. The interface `name` (e.g. `tun_wg0`) is assigned by the system and identifies the tunnel. The tunnel addresses are managed by `pfsense_wireguard_tunnel_address`.",
		Attributes: map[string]schema.Attribute{
			"id": computedIDAttribute(),
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the WireGuard interface, assigned by the system.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled":    optionalBoolAttribute("Enable this tunnel and any associated peers."),
			"descr":      optionalStringAttribute("A description for this WireGuard tunnel."),
			"listenport": optionalStringAttribute("The port WireGuard will listen on for this tunnel."),
			"publickey":  computedStringAttribute("The public key for this tunnel, derived from `privatekey` by the system."),
			"privatekey": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The private key for this tunnel.",
			},
			"mtu": optionalIntAttribute("The MTU for this WireGuard tunnel interface."),
		},
	}
}

func (r *wireGuardTunnelResource) payload(m wireGuardTunnelModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enabled", m.Enabled)
	setString(p, "descr", m.Descr)
	setString(p, "listenport", m.Listenport)
	setString(p, "privatekey", m.Privatekey)
	setInt(p, "mtu", m.MTU)
	return p
}

func (r *wireGuardTunnelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wireGuardTunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Create(ctx, wireGuardTunnelSingular, applyNow(r.payload(plan)))
	if err != nil {
		resp.Diagnostics.AddError("failed to create WireGuard tunnel", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to create WireGuard tunnel", err.Error())
		return
	}
	name := ""
	if s := getString(obj, "name"); s != nil {
		name = *s
	}
	if name == "" {
		resp.Diagnostics.AddError("failed to create WireGuard tunnel", "the API did not return the generated tunnel name")
		return
	}
	plan.Name = types.StringValue(name)
	plan.Publickey = strValue(getString(obj, "publickey"))
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardTunnelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wireGuardTunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, wireGuardTunnelPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read WireGuard tunnel", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Listenport = strValue(getString(obj, "listenport"))
	state.Publickey = strValue(getString(obj, "publickey"))
	state.Privatekey = strValue(getString(obj, "privatekey"))
	state.MTU = intValue(getInt(obj, "mtu"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wireGuardTunnelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wireGuardTunnelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, wireGuardTunnelPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard tunnel", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("WireGuard tunnel not found", "tunnel "+name+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	raw, err := r.client.Update(ctx, wireGuardTunnelSingular, applyNow(payload))
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard tunnel", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard tunnel", err.Error())
		return
	}
	plan.Name = types.StringValue(name)
	plan.Publickey = strValue(getString(obj, "publickey"))
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardTunnelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wireGuardTunnelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, wireGuardTunnelPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard tunnel", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, wireGuardTunnelSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard tunnel", err.Error())
	}
}

func (r *wireGuardTunnelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_wireguard_tunnel_address
// Parent: WireGuard tunnel (natural key = name). Child key: address.
// Resource ID = <tunnel name>|<address>.
// ---------------------------------------------------------------------------

type wireGuardTunnelAddressResource struct{ client *client.Client }

var _ resource.Resource = (*wireGuardTunnelAddressResource)(nil)
var _ resource.ResourceWithConfigure = (*wireGuardTunnelAddressResource)(nil)
var _ resource.ResourceWithImportState = (*wireGuardTunnelAddressResource)(nil)

const (
	wireGuardTunnelAddressPlural   = "/api/v2/vpn/wireguard/tunnel/addresses"
	wireGuardTunnelAddressSingular = "/api/v2/vpn/wireguard/tunnel/address"
)

type wireGuardTunnelAddressModel struct {
	ID       types.String `tfsdk:"id"`
	ParentID types.String `tfsdk:"parent_id"`
	Address  types.String `tfsdk:"address"`
	Mask     types.Int64  `tfsdk:"mask"`
	Descr    types.String `tfsdk:"descr"`
}

func NewWireGuardTunnelAddressResource() resource.Resource { return &wireGuardTunnelAddressResource{} }

func (r *wireGuardTunnelAddressResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_wireguard_tunnel_address"
}
func (r *wireGuardTunnelAddressResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *wireGuardTunnelAddressResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an address assigned to a WireGuard tunnel interface. Identified by the parent tunnel name and the address.",
		Attributes: map[string]schema.Attribute{
			"id":        computedIDAttribute(),
			"parent_id": parentIDAttribute("The `name` of the parent WireGuard tunnel."),
			"address":   keyAttribute("The IPv4 or IPv6 address for this WireGuard tunnel. Unique within the tunnel; immutable after creation."),
			"mask":      requiredIntAttribute("The subnet mask bits for this WireGuard tunnel address."),
			"descr":     optionalStringAttribute("A description for this WireGuard tunnel address entry."),
		},
	}
}

func (r *wireGuardTunnelAddressResource) key(parent, address string) string {
	return parent + "|" + address
}

func (r *wireGuardTunnelAddressResource) identity(m wireGuardTunnelAddressModel) (parent, address string) {
	parent = m.ParentID.ValueString()
	address = m.Address.ValueString()
	if parent == "" {
		parent, address = splitRouteKey(m.ID.ValueString())
	}
	return parent, address
}

func (r *wireGuardTunnelAddressResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wireGuardTunnelAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, wireGuardTunnelPlural, "name", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard tunnel", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "address", plan.Address)
	setInt(payload, "mask", plan.Mask)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, wireGuardTunnelAddressSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create WireGuard tunnel address", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, address))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardTunnelAddressResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wireGuardTunnelAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, wireGuardTunnelPlural, "name", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard tunnel", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, wireGuardTunnelAddressPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to read WireGuard tunnel address", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parent, address))
	state.ParentID = types.StringValue(parent)
	state.Address = types.StringValue(address)
	state.Mask = intValue(getInt(obj, "mask"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wireGuardTunnelAddressResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wireGuardTunnelAddressModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, wireGuardTunnelPlural, "name", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard tunnel", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, wireGuardTunnelAddressPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard tunnel address", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("WireGuard tunnel address not found", "address "+address+" no longer exists")
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "address", plan.Address)
	setInt(payload, "mask", plan.Mask)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, wireGuardTunnelAddressSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard tunnel address", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, address))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardTunnelAddressResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wireGuardTunnelAddressModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, wireGuardTunnelPlural, "name", parent)
	if err != nil {
		// Parent WireGuard tunnel removed out-of-band: the child is already gone.
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, wireGuardTunnelAddressPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard tunnel address", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, wireGuardTunnelAddressSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard tunnel address", err.Error())
	}
}

func (r *wireGuardTunnelAddressResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_wireguard_peer
// Key: descr. The `allowedips` collection is managed by
// pfsense_wireguard_peer_allowed_ip.
// ---------------------------------------------------------------------------

type wireGuardPeerResource struct{ client *client.Client }

var _ resource.Resource = (*wireGuardPeerResource)(nil)
var _ resource.ResourceWithConfigure = (*wireGuardPeerResource)(nil)
var _ resource.ResourceWithImportState = (*wireGuardPeerResource)(nil)

const (
	wireGuardPeerPlural   = "/api/v2/vpn/wireguard/peers"
	wireGuardPeerSingular = "/api/v2/vpn/wireguard/peer"
)

type wireGuardPeerModel struct {
	ID                  types.String `tfsdk:"id"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Tun                 types.String `tfsdk:"tun"`
	Endpoint            types.String `tfsdk:"endpoint"`
	Port                types.String `tfsdk:"port"`
	Descr               types.String `tfsdk:"descr"`
	PersistentKeepalive types.Int64  `tfsdk:"persistentkeepalive"`
	Publickey           types.String `tfsdk:"publickey"`
	Presharedkey        types.String `tfsdk:"presharedkey"`
}

func NewWireGuardPeerResource() resource.Resource { return &wireGuardPeerResource{} }

func (r *wireGuardPeerResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_wireguard_peer"
}
func (r *wireGuardPeerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *wireGuardPeerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WireGuard peer. Identified by its `descr` description. The peer's allowed IPs are managed by `pfsense_wireguard_peer_allowed_ip`.",
		Attributes: map[string]schema.Attribute{
			"id":                  computedIDAttribute(),
			"enabled":             optionalBoolAttribute("Enable this WireGuard peer."),
			"tun":                 optionalStringAttribute("The name of the WireGuard tunnel this peer belongs to. Defaults to `unassigned`."),
			"endpoint":            optionalStringAttribute("The IP address or hostname of the remote peer. Leave unset for a dynamic endpoint."),
			"port":                optionalStringAttribute("The port used by the remote peer."),
			"descr":               keyAttribute("The description for this peer. Unique; immutable after creation."),
			"persistentkeepalive": optionalIntAttribute("The interval (in seconds) for keep alive packets sent to this peer."),
			"publickey":           requiredStringAttribute("The public key for this peer."),
			"presharedkey":        sensitiveStringAttribute("The pre-shared key for this peer."),
		},
	}
}

func (r *wireGuardPeerResource) payload(m wireGuardPeerModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enabled", m.Enabled)
	setString(p, "tun", m.Tun)
	setString(p, "endpoint", m.Endpoint)
	setString(p, "port", m.Port)
	setString(p, "descr", m.Descr)
	setInt(p, "persistentkeepalive", m.PersistentKeepalive)
	setString(p, "publickey", m.Publickey)
	setString(p, "presharedkey", m.Presharedkey)
	return p
}

func (r *wireGuardPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wireGuardPeerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Create(ctx, wireGuardPeerSingular, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to create WireGuard peer", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.Descr.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wireGuardPeerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, wireGuardPeerPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to read WireGuard peer", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(descr)
	state.Enabled = boolValue(getBool(obj, "enabled"))
	state.Tun = strValue(getString(obj, "tun"))
	state.Endpoint = strValue(getString(obj, "endpoint"))
	state.Port = strValue(getString(obj, "port"))
	state.Descr = types.StringValue(descr)
	state.PersistentKeepalive = intValue(getInt(obj, "persistentkeepalive"))
	state.Publickey = strValue(getString(obj, "publickey"))
	state.Presharedkey = strValue(getString(obj, "presharedkey"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wireGuardPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wireGuardPeerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := plan.Descr.ValueString()
	if descr == "" {
		descr = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, wireGuardPeerPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard peer", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("WireGuard peer not found", "peer "+descr+" no longer exists")
		return
	}
	payload := r.payload(plan)
	payload["id"] = id
	if _, err := r.client.Update(ctx, wireGuardPeerSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard peer", err.Error())
		return
	}
	plan.ID = types.StringValue(descr)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wireGuardPeerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	descr := state.Descr.ValueString()
	if descr == "" {
		descr = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, wireGuardPeerPlural, "descr", descr)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard peer", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, wireGuardPeerSingular, client.Query{}.Set("id", formatID(id)).Set("apply", "true")); err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard peer", err.Error())
	}
}

func (r *wireGuardPeerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_wireguard_peer_allowed_ip
// Parent: WireGuard peer (natural key = descr). Child key: address.
// Resource ID = <peer descr>|<address>.
// ---------------------------------------------------------------------------

type wireGuardPeerAllowedIPResource struct{ client *client.Client }

var _ resource.Resource = (*wireGuardPeerAllowedIPResource)(nil)
var _ resource.ResourceWithConfigure = (*wireGuardPeerAllowedIPResource)(nil)
var _ resource.ResourceWithImportState = (*wireGuardPeerAllowedIPResource)(nil)

const (
	wireGuardPeerAllowedIPPlural   = "/api/v2/vpn/wireguard/peer/allowed_ips"
	wireGuardPeerAllowedIPSingular = "/api/v2/vpn/wireguard/peer/allowed_ip"
)

type wireGuardPeerAllowedIPModel struct {
	ID       types.String `tfsdk:"id"`
	ParentID types.String `tfsdk:"parent_id"`
	Address  types.String `tfsdk:"address"`
	Mask     types.Int64  `tfsdk:"mask"`
	Descr    types.String `tfsdk:"descr"`
}

func NewWireGuardPeerAllowedIPResource() resource.Resource { return &wireGuardPeerAllowedIPResource{} }

func (r *wireGuardPeerAllowedIPResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_wireguard_peer_allowed_ip"
}
func (r *wireGuardPeerAllowedIPResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *wireGuardPeerAllowedIPResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an allowed IP entry on a WireGuard peer. Identified by the parent peer description and the address.",
		Attributes: map[string]schema.Attribute{
			"id":        computedIDAttribute(),
			"parent_id": parentIDAttribute("The `descr` of the parent WireGuard peer."),
			"address":   keyAttribute("The IPv4 or IPv6 address for this peer IP. Unique within the peer; immutable after creation."),
			"mask":      requiredIntAttribute("The subnet mask bits for this peer IP."),
			"descr":     optionalStringAttribute("A description for this allowed peer IP."),
		},
	}
}

func (r *wireGuardPeerAllowedIPResource) key(parent, address string) string {
	return parent + "|" + address
}

func (r *wireGuardPeerAllowedIPResource) identity(m wireGuardPeerAllowedIPModel) (parent, address string) {
	parent = m.ParentID.ValueString()
	address = m.Address.ValueString()
	if parent == "" {
		parent, address = splitRouteKey(m.ID.ValueString())
	}
	return parent, address
}

func (r *wireGuardPeerAllowedIPResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wireGuardPeerAllowedIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, wireGuardPeerPlural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard peer", err.Error())
		return
	}
	payload := map[string]any{"parent_id": pid}
	setString(payload, "address", plan.Address)
	setInt(payload, "mask", plan.Mask)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, wireGuardPeerAllowedIPSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to create WireGuard peer allowed IP", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, address))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardPeerAllowedIPResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wireGuardPeerAllowedIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, wireGuardPeerPlural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard peer", err.Error())
		return
	}
	_, obj, found, err := findByKeyInParent(ctx, r.client, wireGuardPeerAllowedIPPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to read WireGuard peer allowed IP", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(r.key(parent, address))
	state.ParentID = types.StringValue(parent)
	state.Address = types.StringValue(address)
	state.Mask = intValue(getInt(obj, "mask"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *wireGuardPeerAllowedIPResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wireGuardPeerAllowedIPModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(plan)
	pid, err := resolveParentID(ctx, r.client, wireGuardPeerPlural, "descr", parent)
	if err != nil {
		resp.Diagnostics.AddError("failed to resolve parent WireGuard peer", err.Error())
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, wireGuardPeerAllowedIPPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard peer allowed IP", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("WireGuard peer allowed IP not found", "allowed IP "+address+" no longer exists")
		return
	}
	payload := map[string]any{"id": id, "parent_id": pid}
	setString(payload, "address", plan.Address)
	setInt(payload, "mask", plan.Mask)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, wireGuardPeerAllowedIPSingular, applyNow(payload)); err != nil {
		resp.Diagnostics.AddError("failed to update WireGuard peer allowed IP", err.Error())
		return
	}
	plan.ID = types.StringValue(r.key(parent, address))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *wireGuardPeerAllowedIPResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wireGuardPeerAllowedIPModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	parent, address := r.identity(state)
	pid, err := resolveParentID(ctx, r.client, wireGuardPeerPlural, "descr", parent)
	if err != nil {
		// Parent WireGuard peer removed out-of-band: the child is already gone.
		return
	}
	id, _, found, err := findByKeyInParent(ctx, r.client, wireGuardPeerAllowedIPPlural, formatID(pid), "address", address)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard peer allowed IP", err.Error())
		return
	}
	if !found {
		return
	}
	q := client.Query{}.Set("parent_id", formatID(pid)).Set("id", formatID(id)).Set("apply", "true")
	if err := r.client.Delete(ctx, wireGuardPeerAllowedIPSingular, q); err != nil {
		resp.Diagnostics.AddError("failed to delete WireGuard peer allowed IP", err.Error())
	}
}

func (r *wireGuardPeerAllowedIPResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
