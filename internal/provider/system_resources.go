package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_system_user
// ---------------------------------------------------------------------------

type systemUserResource struct{ client *client.Client }

var _ resource.Resource = (*systemUserResource)(nil)
var _ resource.ResourceWithConfigure = (*systemUserResource)(nil)
var _ resource.ResourceWithImportState = (*systemUserResource)(nil)

const (
	systemUserPlural   = "/api/v2/users"
	systemUserSingular = "/api/v2/user"
)

type systemUserModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Password       types.String `tfsdk:"password"`
	UID            types.Int64  `tfsdk:"uid"`
	Priv           types.List   `tfsdk:"priv"`
	Disabled       types.Bool   `tfsdk:"disabled"`
	Descr          types.String `tfsdk:"descr"`
	Expires        types.String `tfsdk:"expires"`
	Cert           types.List   `tfsdk:"cert"`
	AuthorizedKeys types.String `tfsdk:"authorizedkeys"`
	IPsecPSK       types.String `tfsdk:"ipsecpsk"`
}

func NewSystemUserResource() resource.Resource { return &systemUserResource{} }

func (r *systemUserResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_user"
}
func (r *systemUserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a local system user.",
		Attributes: map[string]schema.Attribute{
			"id":             computedIDAttribute(),
			"name":           requiredNameAttribute(),
			"password":       schema.StringAttribute{Optional: true, Sensitive: true, Description: "The user's password (plaintext; hashed by the API)."},
			"uid":            schema.Int64Attribute{Computed: true, Description: "The system UID assigned to the user."},
			"priv":           schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "Privilege names assigned to the user."},
			"disabled":       optionalBoolAttribute("Disable this user account."),
			"descr":          optionalStringAttribute("Description for the user."),
			"expires":        optionalStringAttribute("Account expiry date (YYYY-MM-DD)."),
			"cert":           schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "Certificate refids associated with the user."},
			"authorizedkeys": optionalStringAttribute("Authorized SSH public key."),
			"ipsecpsk":       schema.StringAttribute{Optional: true, Sensitive: true, Description: "IPsec pre-shared key."},
		},
	}
}

func (r *systemUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setString(payload, "password", plan.Password)
	setStringList(payload, "priv", plan.Priv)
	setBool(payload, "disabled", plan.Disabled)
	setString(payload, "descr", plan.Descr)
	setString(payload, "expires", plan.Expires)
	setStringList(payload, "cert", plan.Cert)
	setString(payload, "authorizedkeys", plan.AuthorizedKeys)
	setString(payload, "ipsecpsk", plan.IPsecPSK)
	if _, err := r.client.Create(ctx, systemUserSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create user", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, systemUserPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read user", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	// Password is write-only in the API; preserve the prior state value.
	state.UID = intValue(getInt(obj, "uid"))
	state.Priv = strListValue(ctx, getStringSlice(obj, "priv"))
	state.Disabled = boolValue(getBool(obj, "disabled"))
	state.Descr = strValue(getString(obj, "descr"))
	state.Expires = strValue(getString(obj, "expires"))
	state.Cert = strListValue(ctx, getStringSlice(obj, "cert"))
	state.AuthorizedKeys = strValue(getString(obj, "authorizedkeys"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemUserPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update user", err.Error())
		} else {
			resp.Diagnostics.AddError("user not found", "user "+name+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "password", plan.Password)
	setStringList(payload, "priv", plan.Priv)
	setBool(payload, "disabled", plan.Disabled)
	setString(payload, "descr", plan.Descr)
	setString(payload, "expires", plan.Expires)
	setStringList(payload, "cert", plan.Cert)
	setString(payload, "authorizedkeys", plan.AuthorizedKeys)
	setString(payload, "ipsecpsk", plan.IPsecPSK)
	if _, err := r.client.Update(ctx, systemUserSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update user", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemUserPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete user", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, systemUserSingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete user", err.Error())
	}
}

func (r *systemUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_system_group
// ---------------------------------------------------------------------------

type systemGroupResource struct{ client *client.Client }

var _ resource.Resource = (*systemGroupResource)(nil)
var _ resource.ResourceWithConfigure = (*systemGroupResource)(nil)
var _ resource.ResourceWithImportState = (*systemGroupResource)(nil)

const (
	systemGroupPlural   = "/api/v2/user/groups"
	systemGroupSingular = "/api/v2/user/group"
)

type systemGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GID         types.Int64  `tfsdk:"gid"`
	Scope       types.String `tfsdk:"scope"`
	Description types.String `tfsdk:"description"`
	Member      types.List   `tfsdk:"member"`
	Priv        types.List   `tfsdk:"priv"`
}

func NewSystemGroupResource() resource.Resource { return &systemGroupResource{} }

func (r *systemGroupResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_group"
}
func (r *systemGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a local system group.",
		Attributes: map[string]schema.Attribute{
			"id":          computedIDAttribute(),
			"name":        requiredNameAttribute(),
			"gid":         schema.Int64Attribute{Computed: true, Description: "The system GID assigned to the group."},
			"scope":       optionalStringAttribute("The group scope (default `local`)."),
			"description": optionalStringAttribute("Description for the group."),
			"member":      schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "Usernames that are members of the group."},
			"priv":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "Privileges assigned to the group."},
		},
	}
}

func (r *systemGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "name", plan.Name)
	setString(payload, "scope", plan.Scope)
	setString(payload, "description", plan.Description)
	setStringList(payload, "member", plan.Member)
	setStringList(payload, "priv", plan.Priv)
	if _, err := r.client.Create(ctx, systemGroupSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create group", err.Error())
		return
	}
	plan.ID = plan.Name
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, systemGroupPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to read group", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(name)
	state.Name = types.StringValue(name)
	state.GID = intValue(getInt(obj, "gid"))
	state.Scope = strValue(getString(obj, "scope"))
	state.Description = strValue(getString(obj, "description"))
	state.Member = strListValue(ctx, getStringSlice(obj, "member"))
	state.Priv = strListValue(ctx, getStringSlice(obj, "priv"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := plan.Name.ValueString()
	if name == "" {
		name = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemGroupPlural, "name", name)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update group", err.Error())
		} else {
			resp.Diagnostics.AddError("group not found", "group "+name+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "scope", plan.Scope)
	setString(payload, "description", plan.Description)
	setStringList(payload, "member", plan.Member)
	setStringList(payload, "priv", plan.Priv)
	if _, err := r.client.Update(ctx, systemGroupSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update group", err.Error())
		return
	}
	plan.ID = types.StringValue(name)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemGroupPlural, "name", name)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete group", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, systemGroupSingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete group", err.Error())
	}
}

func (r *systemGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_system_tunable
// ---------------------------------------------------------------------------

type systemTunableResource struct{ client *client.Client }

var _ resource.Resource = (*systemTunableResource)(nil)
var _ resource.ResourceWithConfigure = (*systemTunableResource)(nil)
var _ resource.ResourceWithImportState = (*systemTunableResource)(nil)

const (
	systemTunablePlural   = "/api/v2/system/tunables"
	systemTunableSingular = "/api/v2/system/tunable"
)

type systemTunableModel struct {
	ID      types.String `tfsdk:"id"`
	Tunable types.String `tfsdk:"tunable"`
	Value   types.String `tfsdk:"value"`
	Descr   types.String `tfsdk:"descr"`
}

func NewSystemTunableResource() resource.Resource { return &systemTunableResource{} }

func (r *systemTunableResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_tunable"
}
func (r *systemTunableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemTunableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a system tunable (sysctl).",
		Attributes: map[string]schema.Attribute{
			"id":      computedIDAttribute(),
			"tunable": requiredNameAttribute(),
			"value":   requiredStringAttribute("The tunable value."),
			"descr":   optionalStringAttribute("Description for the tunable."),
		},
	}
}

func (r *systemTunableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemTunableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "tunable", plan.Tunable)
	setString(payload, "value", plan.Value)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Create(ctx, systemTunableSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to create tunable", err.Error())
		return
	}
	plan.ID = plan.Tunable
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemTunableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemTunableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	tunable := state.Tunable.ValueString()
	if tunable == "" {
		tunable = state.ID.ValueString()
	}
	_, obj, found, err := findByKey(ctx, r.client, systemTunablePlural, "tunable", tunable)
	if err != nil {
		resp.Diagnostics.AddError("failed to read tunable", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(tunable)
	state.Tunable = types.StringValue(tunable)
	state.Value = strValue(getString(obj, "value"))
	state.Descr = strValue(getString(obj, "descr"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemTunableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemTunableModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	tunable := plan.Tunable.ValueString()
	if tunable == "" {
		tunable = plan.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemTunablePlural, "tunable", tunable)
	if err != nil || !found {
		if err != nil {
			resp.Diagnostics.AddError("failed to update tunable", err.Error())
		} else {
			resp.Diagnostics.AddError("tunable not found", tunable+" no longer exists")
		}
		return
	}
	payload := map[string]any{"id": id}
	setString(payload, "value", plan.Value)
	setString(payload, "descr", plan.Descr)
	if _, err := r.client.Update(ctx, systemTunableSingular, payload); err != nil {
		resp.Diagnostics.AddError("failed to update tunable", err.Error())
		return
	}
	plan.ID = types.StringValue(tunable)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemTunableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemTunableModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	tunable := state.Tunable.ValueString()
	if tunable == "" {
		tunable = state.ID.ValueString()
	}
	id, _, found, err := findByKey(ctx, r.client, systemTunablePlural, "tunable", tunable)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete tunable", err.Error())
		return
	}
	if !found {
		return
	}
	if err := r.client.Delete(ctx, systemTunableSingular, client.Query{}.Set("id", formatID(id))); err != nil {
		resp.Diagnostics.AddError("failed to delete tunable", err.Error())
	}
}

func (r *systemTunableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
