// Package test provides acceptance testing utilities for the Keboola Terraform provider
package test

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/resources/branch"
	"github.com/keboola/terraform-provider-keboola/internal/provider/resources/configuration"
	"github.com/keboola/terraform-provider-keboola/internal/provider/resources/encryption"
	"github.com/keboola/terraform-provider-keboola/internal/provider/resources/scheduler"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

const (
	KbcHost  = "KBC_HOST"
	KbcToken = "KBC_TOKEN"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &testKeboolaProvider{}
)

// testKeboolaProvider is a simplified provider implementation for testing.
type testKeboolaProvider struct {
	version string
}

// testKeboolaProviderModel maps provider schema data to a Go type.
type testKeboolaProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// ProviderConfig returns a provider configuration for testing.
func ProviderConfig() string {
	host := os.Getenv("TEST_KBC_HOST")   //nolint: forbidigo
	token := os.Getenv("TEST_KBC_TOKEN") //nolint: forbidigo

	return `
provider "keboola" {
  host  = "` + host + `"
  token = "` + token + `"
}
`
}

// AccProtoV6ProviderFactories returns a map of provider server factories for testing.
func AccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"keboola": providerserver.NewProtocol6WithError(New("test")()),
	}
}

// New creates a new provider instance for testing.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &testKeboolaProvider{
			version: version,
		}
	}
}

// AccPreCheck is a function to run before tests to ensure test environment is properly set up.
func AccPreCheck() {
	// This can be expanded to check for required environment variables
	if os.Getenv("TEST_KBC_HOST") == "" || os.Getenv("TEST_KBC_TOKEN") == "" { //nolint: forbidigo
		panic("TEST_KBC_HOST and TEST_KBC_TOKEN must be set for acceptance tests")
	}
}

// Metadata returns the provider type name.
func (p *testKeboolaProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "keboola"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *testKeboolaProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Keboola Storage API (https://keboola.docs.apiary.io/).",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "URL of the Keboola Connection API. Can be also provided via " + KbcHost + " environment variable.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "API Token used to authenticate against the API. Can be also provided via " + KbcToken + " environment variable.", //nolint: lll
			},
		},
	}
}

// Configure configures the provider.
func (p *testKeboolaProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	tflog.Info(ctx, "Configuring Keboola API client for tests")

	// Get the user-provided configuration
	var config testKeboolaProviderModel
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
			"The provider cannot create the Keboola API client as there is an unknown configuration value for the Keboola API host. "+ //nolint: lll
				"Either target apply the source of the value first, set the value statically in the configuration, or use the "+KbcHost+" environment variable.", //nolint: lll
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown Keboola API Token",
			"The provider cannot create the Keboola API client as there is an unknown configuration value for the Keboola API token. "+ //nolint: lll
				"Either target apply the source of the value first, set the value statically in the configuration, or use the "+KbcToken+" environment variable.", //nolint: lll
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.
	host := os.Getenv(KbcHost)   //nolint: forbidigo
	token := os.Getenv(KbcToken) //nolint: forbidigo

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
				"Set the host value in the configuration or use the "+KbcHost+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing Keboola API token",
			"The provider cannot create the Keboola API client as there is a missing or empty value for the Keboola API token. "+
				"Set the token value in the configuration or use the "+KbcToken+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "keboola_host", host)
	ctx = tflog.SetField(ctx, "keboola_token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "keboola_token")

	// Create a new Keboola Storage API client using the configuration values
	sapiClient, err := keboola.NewAuthorizedAPI(ctx, host, token)
	if err != nil {
		resp.Diagnostics.AddError("Could not initialize Keboola client", err.Error())

		return
	}

	tokenObject, err := sapiClient.VerifyTokenRequest(token).Send(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Could not initialize Keboola client, given token is invalid:", err.Error())

		return
	}

	// Retrieve all components from Keboola Connection
	tflog.Info(ctx, "Fetching all components from Keboola Connection")
	// The IndexComponentsRequest retrieves all components available in the Keboola Connection project.
	// This is a one-time fetch during provider configuration to avoid repeated API calls.
	stackComponents, err := sapiClient.IndexComponentsRequest().Send(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve components from Keboola Connection", err.Error())

		return
	}

	// Make the Keboola client available during DataSource and Resource
	// type Configure methods.
	data := &providermodels.ProviderData{
		Client:     sapiClient,
		Token:      tokenObject,
		Components: stackComponents.Components,
	}

	// Set the provider data
	resp.DataSourceData = data
	resp.ResourceData = data

	tflog.Info(ctx, "Configured Keboola API client for tests")
}

// DataSources defines the data sources implemented by the provider.
// This is intentionally empty for testing purposes.
func (p *testKeboolaProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

// Resources defines the resources implemented by the provider.
// This adds resource factories for testing purposes only.
func (p *testKeboolaProvider) Resources(_ context.Context) []func() resource.Resource {
	cResource := func() resource.Resource {
		return configuration.NewResource()
	}

	eResource := func() resource.Resource {
		return encryption.NewResource()
	}

	sResource := func() resource.Resource {
		return scheduler.NewResource()
	}

	bResource := func() resource.Resource {
		return branch.NewResource()
	}

	return []func() resource.Resource{
		cResource,
		eResource,
		sResource,
		bResource,
	}
}
