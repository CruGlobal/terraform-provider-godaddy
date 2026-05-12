package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/CruGlobal/terraform-provider-godaddy/internal/api"
)

var defaultRecords = []*api.DomainRecord{
	{Type: api.CNameType, Name: "www", Data: "@", TTL: api.DefaultTTL},
	{Type: api.CNameType, Name: "_domainconnect", Data: "_domainconnect.gd.domaincontrol.com", TTL: api.DefaultTTL},
}

var (
	_ resource.Resource                = &domainRecordResource{}
	_ resource.ResourceWithConfigure   = &domainRecordResource{}
	_ resource.ResourceWithImportState = &domainRecordResource{}
)

func NewDomainRecordResource() resource.Resource {
	return &domainRecordResource{}
}

type domainRecordResource struct {
	client *api.Client
}

type domainRecordResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Domain      types.String `tfsdk:"domain"`
	Customer    types.String `tfsdk:"customer"`
	Addresses   types.List   `tfsdk:"addresses"`
	Nameservers types.List   `tfsdk:"nameservers"`
	Record      types.Set    `tfsdk:"record"`
}

type recordModel struct {
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Data     types.String `tfsdk:"data"`
	TTL      types.Int64  `tfsdk:"ttl"`
	Priority types.Int64  `tfsdk:"priority"`
	Weight   types.Int64  `tfsdk:"weight"`
	Service  types.String `tfsdk:"service"`
	Protocol types.String `tfsdk:"protocol"`
	Port     types.Int64  `tfsdk:"port"`
}

func recordObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":     types.StringType,
			"type":     types.StringType,
			"data":     types.StringType,
			"ttl":      types.Int64Type,
			"priority": types.Int64Type,
			"weight":   types.Int64Type,
			"service":  types.StringType,
			"protocol": types.StringType,
			"port":     types.Int64Type,
		},
	}
}

func (r *domainRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_record"
}

func (r *domainRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "`godaddy_domain_record` manages DNS records for a domain registered with GoDaddy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Numeric GoDaddy domain ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Description: "The domain name to manage records for.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"customer": schema.StringAttribute{
				Description: "Optional GoDaddy customer (shopper) ID. Required when the API key does not belong to the customer owning the domain.",
				Optional:    true,
			},
			"addresses": schema.ListAttribute{
				Description: "A records pointing the root (`@`) of the domain at the given IP addresses.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"nameservers": schema.ListAttribute{
				Description: "NS records to override the default GoDaddy nameservers.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"record": schema.SetNestedAttribute{
				Description: "One or more DNS records to manage on the domain.",
				Optional:    true,
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Record name (subdomain). Use `@` for the root.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Record type. One of A, AAAA, CAA, CNAME, MX, NS, SOA, SRV, TXT.",
							Required:    true,
						},
						"data": schema.StringAttribute{
							Description: "Record data (value).",
							Required:    true,
						},
						"ttl": schema.Int64Attribute{
							Description: "Record TTL in seconds.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(int64(api.DefaultTTL)),
						},
						"priority": schema.Int64Attribute{
							Description: "Priority (MX records).",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(int64(api.DefaultPriority)),
						},
						"weight": schema.Int64Attribute{
							Description: "Weight (SRV records).",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(int64(api.DefaultWeight)),
						},
						"service": schema.StringAttribute{
							Description: "Service (SRV records). Must start with an underscore.",
							Optional:    true,
							Computed:    true,
						},
						"protocol": schema.StringAttribute{
							Description: "Protocol (SRV records). Must start with an underscore.",
							Optional:    true,
							Computed:    true,
						},
						"port": schema.Int64Attribute{
							Description: "Port (SRV records).",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(int64(api.DefaultPort)),
						},
					},
				},
			},
		},
	}
}

func (r *domainRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *domainRecordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan domainRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.applyPlan(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.refreshState(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainRecordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state domainRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refreshState(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *domainRecordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan domainRecordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.applyPlan(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.refreshState(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *domainRecordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state domainRecordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	customer := state.Customer.ValueString()
	domain := state.Domain.ValueString()

	tflog.Info(ctx, "restoring default DNS records", map[string]any{"domain": domain})
	if err := r.client.UpdateDomainRecords(customer, domain, defaultRecords); err != nil {
		resp.Diagnostics.AddError("Failed to restore default records", err.Error())
	}
}

func (r *domainRecordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), req.ID)...)
}

// applyPlan converts the plan into API records and pushes them to GoDaddy.
func (r *domainRecordResource) applyPlan(ctx context.Context, plan *domainRecordResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	customer := plan.Customer.ValueString()
	domain := plan.Domain.ValueString()

	domainInfo, err := lookupDomain(r.client, customer, domain)
	if err != nil {
		diags.AddError("Couldn't find domain", err.Error())
		return diags
	}
	plan.ID = types.StringValue(strconv.FormatInt(domainInfo.ID, 10))

	records, d := buildRecords(ctx, plan)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	tflog.Info(ctx, "updating domain records", map[string]any{"domain": domain})
	if err := r.client.UpdateDomainRecords(customer, domain, records); err != nil {
		diags.AddError("Failed to update records", err.Error())
	}
	return diags
}

// refreshState fetches current records from GoDaddy and stores them on the model.
func (r *domainRecordResource) refreshState(ctx context.Context, state *domainRecordResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	customer := state.Customer.ValueString()
	domain := state.Domain.ValueString()

	domainInfo, err := lookupDomain(r.client, customer, domain)
	if err != nil {
		diags.AddError("Couldn't find domain", err.Error())
		return diags
	}
	state.ID = types.StringValue(strconv.FormatInt(domainInfo.ID, 10))

	tflog.Info(ctx, "fetching domain records", map[string]any{"domain": domain})
	records, err := r.client.GetDomainRecords(customer, domain)
	if err != nil {
		diags.AddError("Couldn't read domain records", err.Error())
		return diags
	}

	// If the user has nameservers in state, we treat the default NS records as
	// managed; otherwise we leave them alone (GoDaddy's defaults).
	hasNameservers := !state.Nameservers.IsNull() && len(state.Nameservers.Elements()) > 0

	aRecs := []string{}
	nsRecs := []string{}
	other := []*api.DomainRecord{}
	for _, rec := range records {
		switch {
		case api.IsDefaultNSRecord(rec):
			nsRecs = append(nsRecs, rec.Data)
		case api.IsDefaultARecord(rec):
			aRecs = append(aRecs, rec.Data)
		default:
			other = append(other, rec)
		}
	}

	aList, d := types.ListValueFrom(ctx, types.StringType, aRecs)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.Addresses = aList

	if hasNameservers {
		nsList, d := types.ListValueFrom(ctx, types.StringType, nsRecs)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		state.Nameservers = nsList
	}

	recSet, d := recordsToSet(other)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	state.Record = recSet

	return diags
}

func buildRecords(ctx context.Context, plan *domainRecordResourceModel) ([]*api.DomainRecord, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := []*api.DomainRecord{}

	if !plan.Record.IsNull() && !plan.Record.IsUnknown() {
		var recs []recordModel
		diags.Append(plan.Record.ElementsAs(ctx, &recs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, rec := range recs {
			built, err := api.NewDomainRecord(
				rec.Name.ValueString(),
				rec.Type.ValueString(),
				rec.Data.ValueString(),
				int(rec.TTL.ValueInt64()),
				api.Priority(int(rec.Priority.ValueInt64())),
				api.Weight(int(rec.Weight.ValueInt64())),
				api.Port(int(rec.Port.ValueInt64())),
				api.Service(rec.Service.ValueString()),
				api.Protocol(rec.Protocol.ValueString()),
			)
			if err != nil {
				diags.AddError("Invalid record", err.Error())
				return nil, diags
			}
			out = append(out, built)
		}
	}

	if !plan.Nameservers.IsNull() && !plan.Nameservers.IsUnknown() {
		var ns []string
		diags.Append(plan.Nameservers.ElementsAs(ctx, &ns, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, n := range ns {
			n = strings.TrimSpace(n)
			if err := api.ValidateData(api.NSType, n); err != nil {
				diags.AddError("Invalid nameserver", err.Error())
				return nil, diags
			}
			rec, err := api.NewNSRecord(n)
			if err != nil {
				diags.AddError("Invalid nameserver", err.Error())
				return nil, diags
			}
			out = append(out, rec)
		}
	}

	if !plan.Addresses.IsNull() && !plan.Addresses.IsUnknown() {
		var addrs []string
		diags.Append(plan.Addresses.ElementsAs(ctx, &addrs, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for _, a := range addrs {
			if err := api.ValidateData(api.AType, a); err != nil {
				diags.AddError("Invalid address", err.Error())
				return nil, diags
			}
			rec, err := api.NewARecord(a)
			if err != nil {
				diags.AddError("Invalid address", err.Error())
				return nil, diags
			}
			out = append(out, rec)
		}
	}

	return out, diags
}

func recordsToSet(recs []*api.DomainRecord) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	objs := make([]attr.Value, 0, len(recs))
	for _, r := range recs {
		port := int64(0)
		if r.Port != nil {
			port = int64(*r.Port)
		}
		obj, d := types.ObjectValue(recordObjectType().AttrTypes, map[string]attr.Value{
			"name":     types.StringValue(r.Name),
			"type":     types.StringValue(r.Type),
			"data":     types.StringValue(r.Data),
			"ttl":      types.Int64Value(int64(r.TTL)),
			"priority": types.Int64Value(int64(r.Priority)),
			"weight":   types.Int64Value(int64(r.Weight)),
			"service":  types.StringValue(r.Service),
			"protocol": types.StringValue(r.Protocol),
			"port":     types.Int64Value(port),
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.SetNull(recordObjectType()), diags
		}
		objs = append(objs, obj)
	}
	set, d := types.SetValue(recordObjectType(), objs)
	diags.Append(d...)
	return set, diags
}

// lookupDomain retries fetching the domain a few times — GoDaddy returns 404
// while domain registrations propagate.
func lookupDomain(client *api.Client, customer, domain string) (*api.Domain, error) {
	var err error
	var d *api.Domain
	for i := 0; i < 10; i++ {
		d, err = client.GetDomain(customer, domain)
		if err == nil {
			return d, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, err
}
