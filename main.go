package main

import (
	"context"
	"terraform-provider-keboola/keboola"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Provider documentation generation.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name keboola

func main() {
	providerserver.Serve(context.Background(), keboola.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/keboola/keboola",
	})
}
