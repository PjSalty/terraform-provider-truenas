package main

import (
	"context"
	"flag"
	"log"
	"os"

	pfprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"

	"github.com/PjSalty/terraform-provider-truenas/internal/provider"
)

var version = "dev"

// Test seams: overridden in main_test.go.
var (
	serveFn  = providerserver.Serve
	logFatal = log.Fatal
)

func main() {
	run(os.Args[1:])
}

// run parses args from the given slice and calls serveFn. Factored out of
// main so the dispatch can be unit-tested without actually binding a Serve
// socket — tests stub serveFn and logFatal.
func run(args []string) {
	fs := flag.NewFlagSet("terraform-provider-truenas", flag.ContinueOnError)
	var debug bool
	fs.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	if err := fs.Parse(args); err != nil {
		logFatal(err.Error())
		return
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/saltstice/truenas",
		Debug:   debug,
	}

	if err := serveFn(context.Background(), func() pfprovider.Provider { return provider.New(version)() }, opts); err != nil {
		logFatal(err.Error())
	}
}
