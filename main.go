// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider"
)

// goreleaser can pass other information to the main package, such as the specific commit
// https://goreleaser.com/cookbooks/using-main.version/

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/stytchauth/stytch",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(provider.Version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
