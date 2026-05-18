//go:build acceptance

package currentuser_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	provideracctest "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider/acctest"
)

func TestAccCurrentUserDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { provideracctest.PreCheck(t) },
		ProtoV6ProviderFactories: provideracctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: provideracctest.ProviderConfig() + `
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
