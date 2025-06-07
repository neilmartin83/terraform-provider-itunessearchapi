package main

import (
	"context"

	itunessearchapi "github.com/neilmartin83/terraform-provider-itunessearchapi/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), itunessearchapi.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/neilmartin83/itunessearchapi",
	})
}
