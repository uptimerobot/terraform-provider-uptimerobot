//go:build acceptance

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccAlertContactDataSourceConfig(id string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
data "uptimerobot_alert_contact" "by_id" {
  id = %q
}

data "uptimerobot_alert_contacts" "all" {}

data "uptimerobot_all_alert_contacts" "all" {}
`, id)
}

func TestAccAlertContactDataSource(t *testing.T) {
	id := os.Getenv("UPTIMEROBOT_TEST_ALERT_CONTACT_ID")
	if id == "" {
		t.Skip("Set UPTIMEROBOT_TEST_ALERT_CONTACT_ID to run alert contact data source acceptance")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertContactDataSourceConfig(id),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.uptimerobot_alert_contact.by_id", "id", id),
					resource.TestCheckResourceAttrSet("data.uptimerobot_alert_contact.by_id", "type"),
					resource.TestCheckResourceAttrSet("data.uptimerobot_alert_contact.by_id", "status"),
					testAccCheckAlertContactIDsContain("data.uptimerobot_alert_contacts.all", id),
					testAccCheckAlertContactIDsContain("data.uptimerobot_all_alert_contacts.all", id),
				),
			},
		},
	})
}

func testAccCheckAlertContactIDsContain(resourceName, id string) resource.TestCheckFunc {
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
