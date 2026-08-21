package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
)

// Reading the real registry is the point of this test: the response shape
// is the one thing a mock cannot prove, and the container resource is
// unusable without a name/version pair that actually resolves.
func testAccContainerImagesPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)

	c, err := acctest.Client()
	if err != nil {
		t.Fatalf("building API client: %v", err)
	}
	ctx, cancel := acctest.Ctx()
	defer cancel()
	if _, err := c.ListContainerImages(ctx); err != nil {
		t.Skipf("container image registry unavailable, skipping: %v", err)
	}
}

func TestAccContainerImagesDataSource_basic(t *testing.T) {
	name := "data.truenas_container_images.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccContainerImagesPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "truenas" {}

data "truenas_container_images" "test" {
  name_prefix = "alpine:"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "container_images"),
					resource.TestCheckResourceAttrSet(name, "images.#"),
					resource.TestCheckResourceAttrSet(name, "images.0.name"),
					// The whole reason this data source exists: a version
					// resolved here is one that currently resolves, unlike
					// a datestamped one written into a config last week.
					resource.TestCheckResourceAttrSet(name, "images.0.latest_version"),
					resource.TestCheckResourceAttrSet(name, "images.0.versions.#"),
					resource.TestCheckResourceAttrSet(name, "names.0"),
					// The prefix filter must actually filter, or a config
					// built on `one(...)` silently picks the wrong image.
					resource.TestMatchResourceAttr(name, "images.0.name", regexpAlpinePrefix()),
				),
			},
		},
	})
}

func regexpAlpinePrefix() *regexp.Regexp { return regexp.MustCompile(`^alpine:`) }
