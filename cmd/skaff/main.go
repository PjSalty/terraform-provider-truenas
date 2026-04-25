// Command skaff generates boilerplate for new terraform-provider-truenas
// resources and data sources, matching the AWS provider's skaff tool.
//
// Usage:
//
//	go run ./cmd/skaff resource <name>
//	go run ./cmd/skaff datasource <name>
//
// The generated files follow the conventions documented in CONTRIBUTING.md.
// The tool refuses to overwrite existing files — rename or delete them first
// if you want to regenerate.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// Test seams. Overridden in main_test.go so main and fatal are testable
// without actually exiting the process, and so writeAll's error branches
// can be exercised without depending on filesystem permissions (which
// don't work when tests run as root in CI containers).
var (
	stderr     io.Writer                       = os.Stderr
	stdout     io.Writer                       = os.Stdout
	exitFn                                     = os.Exit
	osMkdirAll func(string, os.FileMode) error = os.MkdirAll
	osCreate   func(string) (*os.File, error)  = os.Create
)

const usage = `skaff — scaffold new terraform-provider-truenas resources.

Usage:
  skaff resource <name>     Generate a new resource skeleton
  skaff datasource <name>   Generate a new data source skeleton
  skaff -h                  Show this help

Name must be snake_case (e.g. 'ftp_config', 'iscsi_target').
`

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func main() {
	exitFn(run(os.Args[1:]))
}

// run is the testable body of main. It returns a process exit code so
// tests can stub exitFn without actually calling os.Exit.
func run(args []string) int {
	fs := flag.NewFlagSet("skaff", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() < 2 {
		fs.Usage()
		return 2
	}

	kind := fs.Arg(0)
	name := fs.Arg(1)

	if !nameRe.MatchString(name) {
		fatal(fmt.Errorf("name %q must be snake_case, starting with lowercase letter", name))
		return 1
	}

	switch kind {
	case "resource":
		if err := scaffoldResource(name); err != nil {
			fatal(err)
			return 1
		}
	case "datasource", "data-source":
		if err := scaffoldDataSource(name); err != nil {
			fatal(err)
			return 1
		}
	default:
		fs.Usage()
		return 2
	}
	return 0
}

func fatal(err error) {
	fmt.Fprintf(stderr, "skaff: %s\n", err)
}

type scaffoldData struct {
	Name        string // snake_case, e.g. "ftp_config"
	CamelName   string // CamelCase, e.g. "FTPConfig"
	TypeName    string // "truenas_ftp_config"
	Description string // One-line description placeholder
}

func buildData(name string) scaffoldData {
	return scaffoldData{
		Name:        name,
		CamelName:   snakeToCamel(name),
		TypeName:    "truenas_" + name,
		Description: "TODO — describe what this manages on TrueNAS SCALE.",
	}
}

// snakeToCamel converts ftp_config to FTPConfig. Known acronyms are
// upper-cased to match Go naming conventions.
func snakeToCamel(s string) string {
	acronyms := map[string]string{
		"api":   "API",
		"ftp":   "FTP",
		"http":  "HTTP",
		"https": "HTTPS",
		"id":    "ID",
		"ip":    "IP",
		"json":  "JSON",
		"nfs":   "NFS",
		"nvme":  "NVMe",
		"scsi":  "SCSI",
		"smb":   "SMB",
		"snmp":  "SNMP",
		"ssh":   "SSH",
		"ssl":   "SSL",
		"tls":   "TLS",
		"ups":   "UPS",
		"url":   "URL",
		"vm":    "VM",
	}
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if a, ok := acronyms[p]; ok {
			parts[i] = a
			continue
		}
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "")
}

func scaffoldResource(name string) error {
	data := buildData(name)
	targets := []scaffoldTarget{
		{
			path: filepath.Join("internal", "resources", name+".go"),
			tmpl: resourceTemplate,
		},
		{
			path: filepath.Join("internal", "client", name+".go"),
			tmpl: clientTemplate,
		},
		{
			path: filepath.Join("docs", "resources", name+".md"),
			tmpl: docTemplate,
		},
		{
			path: filepath.Join("examples", "resources", "truenas_"+name, "resource.tf"),
			tmpl: exampleTemplate,
		},
		{
			path: filepath.Join("examples", "resources", "truenas_"+name, "import.sh"),
			tmpl: importTemplate,
		},
	}
	return writeAll(targets, data)
}

func scaffoldDataSource(name string) error {
	data := buildData(name)
	targets := []scaffoldTarget{
		{
			path: filepath.Join("internal", "datasources", name+".go"),
			tmpl: dataSourceTemplate,
		},
		{
			path: filepath.Join("docs", "data-sources", name+".md"),
			tmpl: dataSourceDocTemplate,
		},
		{
			path: filepath.Join("examples", "data-sources", "truenas_"+name, "data-source.tf"),
			tmpl: dataSourceExampleTemplate,
		},
	}
	return writeAll(targets, data)
}

type scaffoldTarget struct {
	path string
	tmpl string
}

func writeAll(targets []scaffoldTarget, data scaffoldData) error {
	// First pass: verify no target exists.
	for _, t := range targets {
		if _, err := os.Stat(t.path); err == nil {
			return fmt.Errorf("file already exists: %s (refusing to overwrite)", t.path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("stat %s: %w", t.path, err)
		}
	}

	// Second pass: render and write.
	for _, t := range targets {
		if err := osMkdirAll(filepath.Dir(t.path), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(t.path), err)
		}
		tmpl, err := template.New(t.path).Parse(t.tmpl)
		if err != nil {
			return fmt.Errorf("parse template for %s: %w", t.path, err)
		}
		f, err := osCreate(t.path)
		if err != nil {
			return fmt.Errorf("create %s: %w", t.path, err)
		}
		execErr := tmpl.Execute(f, data)
		// Close unconditionally; if Execute succeeded, the close error (rare,
		// typically only on network filesystems) is accepted silently because
		// the rendered content is already flushed and the file is closed.
		_ = f.Close()
		if execErr != nil {
			return fmt.Errorf("render %s: %w", t.path, execErr)
		}
		fmt.Fprintf(stdout, "wrote %s\n", t.path)
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Next steps:")
	fmt.Fprintf(stdout, "  1. Edit the generated files to fill in the schema and client methods.\n")
	fmt.Fprintf(stdout, "  2. Register the new resource/data source in internal/provider/provider.go.\n")
	fmt.Fprintf(stdout, "  3. Add acceptance tests in internal/provider/acc_%s_test.go (triad: _basic, _update, _disappears).\n", data.Name)
	fmt.Fprintf(stdout, "  4. Add a sweeper in internal/provider/sweeper_test.go.\n")
	fmt.Fprintf(stdout, "  5. Run: make test && make lint\n")
	fmt.Fprintf(stdout, "  6. Run: changie new\n")
	return nil
}

// --- Templates ----------------------------------------------------------

const resourceTemplate = `package resources

// {{.CamelName}}Resource manages a {{.Name}} on TrueNAS SCALE.
//
// TODO: implement Schema, Create, Read, Update, Delete, ImportState, and
// mapResponseToModel. See internal/resources/dataset.go for a reference.
//
// Registration lives in internal/provider/provider.go:Resources().

// TODO: import the framework packages once you start filling in the schema.

// type {{.CamelName}}Resource struct {
// 	client *client.Client
// }
`

const clientTemplate = `package client

// TODO: implement Get{{.CamelName}}, Create{{.CamelName}}, Update{{.CamelName}},
// Delete{{.CamelName}} client methods here. See internal/client/dataset.go for
// reference. Every client method needs httptest-mocked unit tests in
// internal/client/{{.Name}}_test.go.
`

const docTemplate = `---
page_title: "{{.TypeName}} Resource - TrueNAS"
subcategory: ""
description: |-
  {{.Description}}
---

# {{.TypeName}} (Resource)

{{.Description}}

## Example Usage

` + "```hcl" + `
# TODO: add example HCL here
` + "```" + `

## Argument Reference

* TODO — list each required / optional argument

## Attribute Reference

* ` + "`id`" + ` — Identifier.
* TODO — list computed attributes

## Import

Import is supported using the numeric ID:

` + "```sh" + `
terraform import {{.TypeName}}.example 1
` + "```" + `
`

const exampleTemplate = `resource "{{.TypeName}}" "example" {
  # TODO: fill in the required attributes
}
`

const importTemplate = `#!/bin/sh
# Replace "1" with the actual resource ID on your TrueNAS instance.
terraform import {{.TypeName}}.example 1
`

const dataSourceTemplate = `package datasources

// {{.CamelName}}DataSource looks up an existing {{.Name}} on TrueNAS SCALE.
//
// TODO: implement Schema, Read, and Configure using the harness from
// internal/datasources/testutil_test.go as reference.
//
// Registration lives in internal/provider/provider.go:DataSources().
`

const dataSourceDocTemplate = `---
page_title: "{{.TypeName}} Data Source - TrueNAS"
subcategory: ""
description: |-
  {{.Description}}
---

# {{.TypeName}} (Data Source)

{{.Description}}

## Example Usage

` + "```hcl" + `
# TODO: add example HCL here
` + "```" + `

## Argument Reference

* TODO — lookup arguments

## Attribute Reference

* TODO — returned attributes
`

const dataSourceExampleTemplate = `data "{{.TypeName}}" "example" {
  # TODO: fill in the lookup key
}
`
