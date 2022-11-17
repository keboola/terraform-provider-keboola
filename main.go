package main

import (
	"context"
	"terraform-provider-keboola/keboola"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), keboola.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/keboola/keboola",
	})
}
