package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), itunessearchapi.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/neilmartin83/itunessearchapi",
	})
}
