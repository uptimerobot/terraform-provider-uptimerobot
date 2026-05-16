//go:build acceptance

package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCurrentUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + `
data "uptimerobot_current_user" "current" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.uptimerobot_current_user.current", "id", "current"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_current_user.current", "monitor_limit"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_current_user.current", "monitors_count"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_current_user.current", "plan"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_current_user.current", "sms_credits"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_current_user.current", "subscription_monitor_limit"),
				),
			},
		},
	})
}

func TestAccTagDataSources(t *testing.T) {
	testAccPreCheck(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	tags, err := testAccAPIClient().ListAllTags(ctx)
	if err != nil {
		t.Fatalf("could not list tags for acceptance precheck: %v", err)
	}
	if len(tags) == 0 {
		t.Skip("acceptance account has no tags to look up")
	}

	tag := tags[0]
	id := fmt.Sprintf("%d", tag.ID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTagDataSourcesConfig(id, tag.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.uptimerobot_tag.by_id", "id", id),
					resource.TestCheckResourceAttr("data.uptimerobot_tag.by_id", "name", tag.Name),
					resource.TestCheckResourceAttr("data.uptimerobot_tag.by_name", "id", id),
					resource.TestCheckResourceAttr("data.uptimerobot_tag.by_name", "name", tag.Name),
					testAccCheckTagIDsContain("data.uptimerobot_tags.all", id),
					resource.TestCheckResourceAttr("data.uptimerobot_tags.by_name", "ids.#", "1"),
					resource.TestCheckResourceAttr("data.uptimerobot_tags.by_name", "ids.0", id),
				),
			},
		},
	})
}

func testAccTagDataSourcesConfig(id, name string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "uptimerobot_tag" "by_id" {
  id = %q
}

data "uptimerobot_tag" "by_name" {
  name = %q
}

data "uptimerobot_tags" "all" {}

data "uptimerobot_tags" "by_name" {
  name = %q
}
`, id, name, name)
}

func testAccCheckTagIDsContain(resourceName, id string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %s not found", resourceName)
		}

		count := rs.Primary.Attributes["ids.#"]
		for i := 0; i < len(rs.Primary.Attributes); i++ {
			if rs.Primary.Attributes[fmt.Sprintf("ids.%d", i)] == id {
				return nil
			}
		}

		return fmt.Errorf("ids list with count %s does not contain %q", count, id)
	}
}
