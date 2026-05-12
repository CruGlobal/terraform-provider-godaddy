package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/CruGlobal/terraform-provider-godaddy/internal/api"
)

var (
	_ resource.Resource                = &domainPurchaseResource{}
	_ resource.ResourceWithConfigure   = &domainPurchaseResource{}
	_ resource.ResourceWithImportState = &domainPurchaseResource{}
)

func NewDomainPurchaseResource() resource.Resource {
	return &domainPurchaseResource{}
}

type domainPurchaseResource struct {
	client *api.Client
}

type domainPurchaseResourceModel struct {
	ID            types.String  `tfsdk:"id"`
	Domain        types.String  `tfsdk:"domain"`
	Customer      types.String  `tfsdk:"customer"`
	YearsLeased   types.Int64   `tfsdk:"years_leased"`
	EnablePrivacy types.Bool    `tfsdk:"enable_privacy"`
	AutoRenew     types.Bool    `tfsdk:"auto_renew"`
	Nameservers   types.List    `tfsdk:"nameservers"`
	Admin         *contactModel `tfsdk:"admin"`
	Billing       *contactModel `tfsdk:"billing"`
	Registrant    *contactModel `tfsdk:"registrant"`
	Tech          *contactModel `tfsdk:"tech"`
}

type contactModel struct {
	Address      *addressModel `tfsdk:"address"`
	Email        types.String  `tfsdk:"email"`
	Fax          types.String  `tfsdk:"fax"`
	JobTitle     types.String  `tfsdk:"job_title"`
	FirstName    types.String  `tfsdk:"first_name"`
	LastName     types.String  `tfsdk:"last_name"`
	MiddleName   types.String  `tfsdk:"middle_name"`
	Organization types.String  `tfsdk:"organization"`
	Phone        types.String  `tfsdk:"phone"`
}

type addressModel struct {
	Line1      types.String `tfsdk:"line_1"`
	Line2      types.String `tfsdk:"line_2"`
	City       types.String `tfsdk:"city"`
	Country    types.String `tfsdk:"country"`
	PostalCode types.String `tfsdk:"postal_code"`
	State      types.String `tfsdk:"state"`
}

func (r *domainPurchaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_purchase"
}

func (r *domainPurchaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	addrAttrs := map[string]schema.Attribute{
		"line_1":      schema.StringAttribute{Optional: true},
		"line_2":      schema.StringAttribute{Optional: true},
		"city":        schema.StringAttribute{Optional: true},
		"country":     schema.StringAttribute{Optional: true},
		"postal_code": schema.StringAttribute{Optional: true},
		"state":       schema.StringAttribute{Optional: true},
	}

	contactAttrs := map[string]schema.Attribute{
		"address": schema.SingleNestedAttribute{
			Description: "Mailing address for the contact.",
			Optional:    true,
			Attributes:  addrAttrs,
		},
		"email":        schema.StringAttribute{Optional: true},
		"fax":          schema.StringAttribute{Optional: true},
		"job_title":    schema.StringAttribute{Optional: true},
		"first_name":   schema.StringAttribute{Optional: true},
		"last_name":    schema.StringAttribute{Optional: true},
		"middle_name":  schema.StringAttribute{Optional: true},
		"organization": schema.StringAttribute{Optional: true},
		"phone":        schema.StringAttribute{Optional: true},
	}

	resp.Schema = schema.Schema{
		Description: "`godaddy_domain_purchase` registers a new domain through GoDaddy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric GoDaddy domain ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Domain name to register.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"customer": schema.StringAttribute{
				Description: "Optional GoDaddy customer (shopper) ID.",
				Optional:    true,
			},
			"years_leased": schema.Int64Attribute{
				Description: "Lease length in years.",
				Required:    true,
			},
			"enable_privacy": schema.BoolAttribute{
				Description: "Enable WHOIS privacy.",
				Optional:    true,
				Computed:    true,
			},
			"auto_renew": schema.BoolAttribute{
				Description: "Auto-renew on expiry.",
				Optional:    true,
				Computed:    true,
			},
			"nameservers": schema.ListAttribute{
				Description: "Custom nameservers for the domain.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"admin":      schema.SingleNestedAttribute{Optional: true, Attributes: contactAttrs},
			"billing":    schema.SingleNestedAttribute{Optional: true, Attributes: contactAttrs},
			"registrant": schema.SingleNestedAttribute{Optional: true, Attributes: contactAttrs},
			"tech":       schema.SingleNestedAttribute{Optional: true, Attributes: contactAttrs},
		},
	}
}

func (r *domainPurchaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*api.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected *api.Client, got %T. Please report this to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *domainPurchaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainPurchaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purchase, d := planToPurchase(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "purchasing domain", map[string]any{"domain": plan.Domain.ValueString()})
	if _, err := r.client.PurchaseDomain(plan.Customer.ValueString(), purchase); err != nil {
		resp.Diagnostics.AddError("Failed to purchase domain", err.Error())
		return
	}

	if d := r.fetchAndPopulate(ctx, &plan); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainPurchaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainPurchaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d := r.fetchAndPopulate(ctx, &state); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainPurchaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainPurchaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	purchase, d := planToPurchase(ctx, &plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "updating domain", map[string]any{"domain": plan.Domain.ValueString()})
	if err := r.client.UpdateDomain(plan.Customer.ValueString(), plan.Domain.ValueString(), purchase); err != nil {
		resp.Diagnostics.AddError("Failed to update domain", err.Error())
		return
	}

	if d := r.fetchAndPopulate(ctx, &plan); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainPurchaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainPurchaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "canceling domain", map[string]any{"domain": state.Domain.ValueString()})
	if err := r.client.CancelDomain(state.Customer.ValueString(), state.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to cancel domain", err.Error())
	}
}

func (r *domainPurchaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainPurchaseResource) fetchAndPopulate(ctx context.Context, state *domainPurchaseResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	d, err := lookupDomain(r.client, state.Customer.ValueString(), state.Domain.ValueString())
	if err != nil {
		diags.AddError("Couldn't find domain", err.Error())
		return diags
	}
	state.ID = types.StringValue(strconv.FormatInt(d.ID, 10))
	state.AutoRenew = types.BoolValue(d.AutoRenew)
	state.EnablePrivacy = types.BoolValue(d.EnablePrivacy)
	if d.YearsLeased > 0 {
		state.YearsLeased = types.Int64Value(int64(d.YearsLeased))
	}

	nsList, nd := types.ListValueFrom(ctx, types.StringType, d.NameServers)
	diags.Append(nd...)
	if !diags.HasError() {
		state.Nameservers = nsList
	}
	return diags
}

func planToPurchase(_ context.Context, plan *domainPurchaseResourceModel) (*api.DomainPurchase, diag.Diagnostics) {
	var diags diag.Diagnostics
	purchase := &api.DomainPurchase{
		Domain:        plan.Domain.ValueString(),
		YearsLeased:   int(plan.YearsLeased.ValueInt64()),
		EnablePrivacy: plan.EnablePrivacy.ValueBool(),
		AutoRenew:     plan.AutoRenew.ValueBool(),
		Consent: &api.Consent{
			AgreementKeys: []string{"DNRA"},
			AgreedAt:      time.Now().UTC().Format(time.RFC3339),
			AgreedBy:      "127.0.0.1",
		},
	}

	if !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown() {
		var ns []string
		diags.Append(plan.Nameservers.ElementsAs(context.Background(), &ns, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, n := range ns {
			if err := api.ValidateData(api.NSType, n); err != nil {
				diags.AddError("Invalid nameserver", err.Error())
				return nil, diags
			}
		}
		purchase.NameServers = ns
	} else {
		purchase.NameServers = []string{}
	}

	purchase.AdminContact = contactToAPI(plan.Admin)
	purchase.BillingContact = contactToAPI(plan.Billing)
	purchase.RegistrantContact = contactToAPI(plan.Registrant)
	purchase.TechContact = contactToAPI(plan.Tech)

	return purchase, diags
}

func contactToAPI(c *contactModel) *api.Contact {
	if c == nil {
		return nil
	}
	contact := &api.Contact{
		Email:        c.Email.ValueString(),
		Fax:          c.Fax.ValueString(),
		JobTitle:     c.JobTitle.ValueString(),
		FirstName:    c.FirstName.ValueString(),
		LastName:     c.LastName.ValueString(),
		MiddleName:   c.MiddleName.ValueString(),
		Organization: c.Organization.ValueString(),
		Phone:        c.Phone.ValueString(),
	}
	if c.Address != nil {
		contact.Address = &api.Address{
			Line1:      c.Address.Line1.ValueString(),
			Line2:      c.Address.Line2.ValueString(),
			City:       c.Address.City.ValueString(),
			Country:    c.Address.Country.ValueString(),
			PostalCode: c.Address.PostalCode.ValueString(),
			State:      c.Address.State.ValueString(),
		}
	}
	return contact
}
