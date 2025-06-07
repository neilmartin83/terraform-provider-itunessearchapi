package itunessearchapi

import (
	"context"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/itunessearchapi"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	providerserver.Serve(context.Background(), itunessearchapi.New, providerserver.ServeOpts{
		Address: "registry.terraform.io/example/itunes",
	})
}
