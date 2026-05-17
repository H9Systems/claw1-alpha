package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/h9-systems/terraform-provider-claw1/internal/provider"
)

func main() {
	err := providerserver.Serve(context.Background(), provider.New, providerserver.ServeOpts{
		Address: "local/h9-systems/claw1",
	})
	if err != nil {
		log.Fatal(err)
	}
}
