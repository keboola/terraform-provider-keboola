package keboola

import (
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

var (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the StorageAPI client is properly configured.
	providerConfig = `
provider "keboola" {
  host  = "` + os.Getenv("TEST_KBC_HOST") + `"
  token = "` + os.Getenv("TEST_KBC_TOKEN") + `"
}
`
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"keboola": providerserver.NewProtocol6WithError(New()),
	}
)
