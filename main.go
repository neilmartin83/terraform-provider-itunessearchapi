package main

import (
	"context"
	"log"

	itunessearchapi "github.com/neilmartin83/terraform-provider-itunessearchapi/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	if err := providerserver.Serve(context.Background(), itunessearchapi.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/neilmartin83/itunessearchapi",
	}); err != nil {
		log.Fatalf("Error starting provider server: %v", err)
	}
}
