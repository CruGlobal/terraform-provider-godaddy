package provider

import (
	"context"
	"fmt"
	"strconv"

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

// defaultNameservers is what GoDaddy assigns to a domain by default. We restore
// these on Delete so the domain remains resolvable.
var defaultNameservers = []string{
	"ns53.domaincontrol.com",
	"ns54.domaincontrol.com",
}

var (
	_ resource.Resource                = &domainNameserversResource{}
	_ resource.ResourceWithConfigure   = &domainNameserversResource{}
	_ resource.ResourceWithImportState = &domainNameserversResource{}
)

func NewDomainNameserversResource() resource.Resource {
	return &domainNameserversResource{}
}

type domainNameserversResource struct {
	client *api.Client
}

type domainNameserversResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Customer    types.String `tfsdk:"customer"`
	Nameservers types.List   `tfsdk:"nameservers"`
}

func (r *domainNameserversResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_nameservers"
}

func (r *domainNameserversResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "`godaddy_domain_nameservers` manages the nameservers for a domain registered with GoDaddy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric GoDaddy domain ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "Domain name to manage nameservers for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"customer": schema.StringAttribute{
				Description: "Optional GoDaddy customer (shopper) ID.",
				Optional:    true,
			},
			"nameservers": schema.ListAttribute{
				Description: "List of nameserver hostnames.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *domainNameserversResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainNameserversResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainNameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainNameserversResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainNameserversResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	d, err := r.client.GetDomain(state.Customer.ValueString(), state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Couldn't read domain", err.Error())
		return
	}
	state.ID = types.StringValue(strconv.FormatInt(d.ID, 10))
	nsList, nd := types.ListValueFrom(ctx, types.StringType, d.NameServers)
	resp.Diagnostics.Append(nd...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Nameservers = nsList
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainNameserversResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainNameserversResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.apply(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainNameserversResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainNameserversResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "resetting nameservers", map[string]any{"domain": state.Domain.ValueString()})
	if err := r.client.UpdateDomain(
		state.Customer.ValueString(),
		state.Domain.ValueString(),
		&api.DomainPurchase{NameServers: defaultNameservers},
	); err != nil {
		resp.Diagnostics.AddError("Failed to reset nameservers", err.Error())
	}
}

func (r *domainNameserversResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

func (r *domainNameserversResource) apply(ctx context.Context, plan *domainNameserversResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	customer := plan.Customer.ValueString()
	domain := plan.Domain.ValueString()

	d, err := r.client.GetDomain(customer, domain)
	if err != nil {
		diags.AddError("Couldn't find domain", err.Error())
		return diags
	}
	plan.ID = types.StringValue(strconv.FormatInt(d.ID, 10))

	var ns []string
	diags.Append(plan.Nameservers.ElementsAs(ctx, &ns, false)...)
	if diags.HasError() {
		return diags
	}
	for _, n := range ns {
		if err := api.ValidateData(api.NSType, n); err != nil {
			diags.AddError("Invalid nameserver", err.Error())
			return diags
		}
	}

	tflog.Info(ctx, "setting nameservers", map[string]any{"domain": domain})
	if err := r.client.UpdateDomain(customer, domain, &api.DomainPurchase{NameServers: ns}); err != nil {
		diags.AddError("Failed to set nameservers", err.Error())
	}
	return diags
}
