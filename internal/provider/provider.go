package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/CruGlobal/terraform-provider-godaddy/internal/api"
)

var _ provider.Provider = &godaddyProvider{}

type godaddyProvider struct {
	version string
}

type godaddyProviderModel struct {
	Key     types.String `tfsdk:"key"`
	Secret  types.String `tfsdk:"secret"`
	BaseURL types.String `tfsdk:"baseurl"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &godaddyProvider{version: version}
	}
}

func (p *godaddyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "godaddy"
	resp.Version = p.version
}

func (p *godaddyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The godaddy provider lets you manage DNS records, nameservers, and domain registrations for domains on your GoDaddy account.",
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description: "GoDaddy API Key. May also be set with the `GODADDY_API_KEY` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret": schema.StringAttribute{
				Description: "GoDaddy API Secret. May also be set with the `GODADDY_API_SECRET` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"baseurl": schema.StringAttribute{
				Description: "GoDaddy API base URL. Defaults to `https://api.godaddy.com`.",
				Optional:    true,
			},
		},
	}
}

func (p *godaddyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg godaddyProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key := stringValueOrEnv(cfg.Key, "GODADDY_API_KEY", "")
	secret := stringValueOrEnv(cfg.Secret, "GODADDY_API_SECRET", "")
	baseURL := stringValueOrEnv(cfg.BaseURL, "GODADDY_API_URL", "https://api.godaddy.com")

	if key == "" {
		resp.Diagnostics.AddError(
			"Missing GoDaddy API key",
			"Set the `key` provider attribute or the `GODADDY_API_KEY` environment variable.",
		)
	}
	if secret == "" {
		resp.Diagnostics.AddError(
			"Missing GoDaddy API secret",
			"Set the `secret` provider attribute or the `GODADDY_API_SECRET` environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := api.NewClient(baseURL, key, secret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to configure GoDaddy client",
			fmt.Sprintf("Error creating GoDaddy API client: %s", err),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *godaddyProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainRecordResource,
		NewDomainPurchaseResource,
		NewDomainNameserversResource,
	}
}

func (p *godaddyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func stringValueOrEnv(v types.String, envVar, fallback string) string {
	if !v.IsNull() && !v.IsUnknown() {
		return v.ValueString()
	}
	if env := os.Getenv(envVar); env != "" {
		return env
	}
	return fallback
}
