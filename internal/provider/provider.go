package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/go-client/pkg/keboola"
)

// Ensure the implementation satisfies the expected interfaces
var _ provider.Provider = (*keboolaProvider)(nil)

const KBC_HOST = "KBC_HOST"
const KBC_TOKEN = "KBC_TOKEN"

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &keboolaProvider{
			version: version,
		}
	}
}

// keboolaProvider is the provider implementation.
type keboolaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type providerData struct {
	client *keboola.API
	token  *keboola.Token
}

// keboolaProviderModel maps provider schema data to a Go type.
type keboolaProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// Metadata returns the provider type name.
func (p *keboolaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "keboola"
	resp.Version = p.version
}

// GetSchema defines the provider-level schema for configuration data.
func (p *keboolaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Keboola Storage API (https://keboola.docs.apiary.io/).",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "URI for Keboola Storage API. May also be provided via " + KBC_HOST + " environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Storage API Token for the Keboola Storage API. May also be provided via " + KBC_TOKEN + " environment variable.",
				Sensitive:   true,
				Optional:    true,
			},
		},
	}
}

// Configure prepares a Keboola API client for data sources and resources.
func (p *keboolaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Keboola API client")
	// Retrieve provider data from configuration
	var config keboolaProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Keboola API Host",
			"The provider cannot create the Keboola API client as there is an unknown configuration value for the Keboola API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the "+KBC_HOST+" environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Keboola API Token",
			"The provider cannot create the Keboola API client as there is an unknown configuration value for the Keboola API token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the "+KBC_TOKEN+" environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	host := os.Getenv(KBC_HOST)
	token := os.Getenv(KBC_TOKEN)

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Keboola API Host",
			"The provider cannot create the Keboola API client as there is a missing or empty value for the Keboola API host. "+
				"Set the host value in the configuration or use the "+KBC_HOST+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Keboola API token",
			"The provider cannot create the Keboola API client as there is a missing or empty value for the Keboola API token. "+
				"Set the token value in the configuration or use the "+KBC_TOKEN+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "keboola_host", host)
	ctx = tflog.SetField(ctx, "keboola_token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "keboola_token")

	// Create a new Keboola Storage api client using the configuration values
	sapiClient, err := keboola.NewAPI(ctx, host, keboola.WithToken(token))
	if err != nil {
		resp.Diagnostics.AddError("Could not initialize Keboola client", err.Error())
	}
	tokenObject, err := sapiClient.VerifyTokenRequest(token).Send(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Could not initialize Keboola client, given token is invalid:", err.Error())
	}

	// Make the Keboola client available during DataSource and Resource
	// type Configure methods.
	data := providerData{sapiClient, tokenObject}
	resp.DataSourceData = &data
	resp.ResourceData = &data

	tflog.Info(ctx, "Configured Keboola API client", map[string]any{"success": true})
}

// DataSources defines the data sources implemented in the provider.
func (p *keboolaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *keboolaProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewConfigResource,
		NewEnryptionResource,
	}
}
