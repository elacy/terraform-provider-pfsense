package provider

import (
	"context"
	"encoding/json"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_system_ca
//
// Certificate authorities are addressed by array index, but they expose a
// persistent `refid` (UID) that we use as the Terraform identifier.
// ---------------------------------------------------------------------------

type systemCAResource struct{ client *client.Client }

var _ resource.Resource = (*systemCAResource)(nil)
var _ resource.ResourceWithConfigure = (*systemCAResource)(nil)
var _ resource.ResourceWithImportState = (*systemCAResource)(nil)

const (
	systemCAPlural   = "/api/v2/system/certificate_authorities"
	systemCASingular = "/api/v2/system/certificate_authority"
)

type systemCAModel struct {
	ID           types.String `tfsdk:"id"`
	Descr        types.String `tfsdk:"descr"`
	RefID        types.String `tfsdk:"refid"`
	Trust        types.Bool   `tfsdk:"trust"`
	RandomSerial types.Bool   `tfsdk:"randomserial"`
	Serial       types.Int64  `tfsdk:"serial"`
	Crt          types.String `tfsdk:"crt"`
	Prv          types.String `tfsdk:"prv"`
}

func NewSystemCAResource() resource.Resource { return &systemCAResource{} }

func (r *systemCAResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_ca"
}
func (r *systemCAResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemCAResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a certificate authority (CA).",
		Attributes: map[string]schema.Attribute{
			"id":           computedIDAttribute(),
			"descr":        requiredStringAttribute("Name/description of the CA."),
			"refid":        schema.StringAttribute{Computed: true, Description: "Persistent unique ID assigned to the CA."},
			"trust":        optionalBoolAttribute("Add this CA to the trust store."),
			"randomserial": optionalBoolAttribute("Use a random serial number for certificates."),
			"serial":       optionalIntAttribute("Next certificate serial number."),
			"crt":          schema.StringAttribute{Required: true, Sensitive: true, Description: "The CA certificate (base64)."},
			"prv":          schema.StringAttribute{Optional: true, Sensitive: true, Description: "The CA private key (base64)."},
		},
	}
}

func (r *systemCAResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemCAModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "descr", plan.Descr)
	setBool(payload, "trust", plan.Trust)
	setBool(payload, "randomserial", plan.RandomSerial)
	setInt(payload, "serial", plan.Serial)
	setString(payload, "crt", plan.Crt)
	setString(payload, "prv", plan.Prv)
	raw, err := r.client.Create(ctx, systemCASingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to create CA", err.Error())
		return
	}
	var created map[string]any
	if err := json.Unmarshal(raw, &created); err == nil {
		if refid := getString(created, "refid"); refid != nil {
			plan.RefID = types.StringValue(*refid)
			plan.ID = types.StringValue(*refid)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemCAResource) findRefID(ctx context.Context, refid string) (any, map[string]any, bool, error) {
	return findByKey(ctx, r.client, systemCAPlural, "refid", refid)
}

func (r *systemCAResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemCAModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := state.RefID.ValueString()
	if refid == "" {
		refid = state.ID.ValueString()
	}
	_, obj, found, err := r.findRefID(ctx, refid)
	if err != nil {
		resp.Diagnostics.AddError("failed to read CA", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(refid)
	state.RefID = types.StringValue(refid)
	state.Descr = strValue(getString(obj, "descr"))
	state.Trust = boolValue(getBool(obj, "trust"))
	state.RandomSerial = boolValue(getBool(obj, "randomserial"))
	state.Serial = intValue(getInt(obj, "serial"))
	state.Crt = strValue(getString(obj, "crt"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemCAResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemCAModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := plan.RefID.ValueString()
	if refid == "" {
		refid = plan.ID.ValueString()
	}
	id, _, found, err := r.findRefID(ctx, refid)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update CA", err.Error())
		} else {
			resp.Diagnostics.AddError("CA not found", "CA "+refid+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "descr", plan.Descr)
	setBool(payload, "trust", plan.Trust)
	setBool(payload, "randomserial", plan.RandomSerial)
	setInt(payload, "serial", plan.Serial)
	setString(payload, "crt", plan.Crt)
	setString(payload, "prv", plan.Prv)
	if _, err := r.client.Update(ctx, systemCASingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update CA", err.Error())
		return
	}
	plan.ID = types.StringValue(refid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemCAResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemCAModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := state.RefID.ValueString()
	if refid == "" {
		refid = state.ID.ValueString()
	}
	id, _, found, err := r.findRefID(ctx, refid)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete CA", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, systemCASingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete CA", err.Error())
	}
}

func (r *systemCAResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_system_certificate
// ---------------------------------------------------------------------------

type systemCertificateResource struct{ client *client.Client }

var _ resource.Resource = (*systemCertificateResource)(nil)
var _ resource.ResourceWithConfigure = (*systemCertificateResource)(nil)
var _ resource.ResourceWithImportState = (*systemCertificateResource)(nil)

const (
	systemCertificatePlural   = "/api/v2/system/certificates"
	systemCertificateSingular = "/api/v2/system/certificate"
)

type systemCertificateModel struct {
	ID         types.String `tfsdk:"id"`
	Descr      types.String `tfsdk:"descr"`
	RefID      types.String `tfsdk:"refid"`
	Type       types.String `tfsdk:"type"`
	CA         types.String `tfsdk:"caref"`
	Crt        types.String `tfsdk:"crt"`
	Prv        types.String `tfsdk:"prv"`
	CSR        types.String `tfsdk:"csr"`
	ValidFrom  types.String `tfsdk:"valid_from"`
	ValidUntil types.String `tfsdk:"valid_until"`
}

func NewSystemCertificateResource() resource.Resource { return &systemCertificateResource{} }

func (r *systemCertificateResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_certificate"
}
func (r *systemCertificateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a certificate.",
		Attributes: map[string]schema.Attribute{
			"id":          computedIDAttribute(),
			"descr":       requiredStringAttribute("Name/description of the certificate."),
			"refid":       schema.StringAttribute{Computed: true, Description: "Persistent unique ID assigned to the certificate."},
			"type":        optionalStringAttribute("Certificate type (default `server`)."),
			"caref":       schema.StringAttribute{Computed: true, Description: "The CA that signed this certificate."},
			"crt":         schema.StringAttribute{Required: true, Sensitive: true, Description: "The certificate (base64)."},
			"prv":         schema.StringAttribute{Required: true, Sensitive: true, Description: "The private key (base64)."},
			"csr":         schema.StringAttribute{Computed: true, Description: "The certificate signing request (base64)."},
			"valid_from":  schema.StringAttribute{Computed: true, Description: "Certificate validity start date."},
			"valid_until": schema.StringAttribute{Computed: true, Description: "Certificate validity end date."},
		},
	}
}

func (r *systemCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "descr", plan.Descr)
	setString(payload, "type", plan.Type)
	setString(payload, "crt", plan.Crt)
	setString(payload, "prv", plan.Prv)
	raw, err := r.client.Create(ctx, systemCertificateSingular, payload)
	if err != nil {
		resp.Diagnostics.AddError("failed to create certificate", err.Error())
		return
	}
	var created map[string]any
	if err := json.Unmarshal(raw, &created); err == nil {
		if refid := getString(created, "refid"); refid != nil {
			plan.RefID = types.StringValue(*refid)
			plan.ID = types.StringValue(*refid)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := state.RefID.ValueString()
	if refid == "" {
		refid = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, systemCertificatePlural, "refid", refid)
	if err != nil {
		resp.Diagnostics.AddError("failed to read certificate", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(refid)
	state.RefID = types.StringValue(refid)
	state.Descr = strValue(getString(obj, "descr"))
	state.Type = strValue(getString(obj, "type"))
	state.CA = strValue(getString(obj, "caref"))
	state.Crt = strValue(getString(obj, "crt"))
	state.CSR = strValue(getString(obj, "csr"))
	state.ValidFrom = strValue(getString(obj, "valid_from"))
	state.ValidUntil = strValue(getString(obj, "valid_until"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := plan.RefID.ValueString()
	if refid == "" {
		refid = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemCertificatePlural, "refid", refid)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update certificate", err.Error())
		} else {
			resp.Diagnostics.AddError("certificate not found", "certificate "+refid+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "descr", plan.Descr)
	setString(payload, "type", plan.Type)
	setString(payload, "crt", plan.Crt)
	setString(payload, "prv", plan.Prv)
	if _, err := r.client.Update(ctx, systemCertificateSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update certificate", err.Error())
		return
	}
	plan.ID = types.StringValue(refid)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	refid := state.RefID.ValueString()
	if refid == "" {
		refid = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemCertificatePlural, "refid", refid)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete certificate", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, systemCertificateSingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete certificate", err.Error())
	}
}

func (r *systemCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
