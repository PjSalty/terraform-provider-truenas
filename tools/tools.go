//go:build tools

// Package tools tracks development-time dependencies so `go mod tidy` keeps
// them in go.sum even when they are not imported by production code.
//
// This file is the canonical tools manifest. Build tag `tools` gates it out
// of normal compiles; `go mod tidy` reads the imports to pin versions.
package tools

import (
	// Acceptance testing framework for Terraform providers.
	_ "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	// Registry documentation generator (invoked via `make docs`).
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)
