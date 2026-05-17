//go:build acceptance

package acctest

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	sdkacctest "github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/uptimerobot/terraform-provider-uptimerobot/internal/client"
	providerpkg "github.com/uptimerobot/terraform-provider-uptimerobot/internal/provider"
)

// ProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"uptimerobot": providerserver.NewProtocol6WithError(providerpkg.New("test")()),
}

func PreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance tests are skipped unless TF_ACC=1")
	}

	if os.Getenv("UPTIMEROBOT_API_KEY") == "" {
		t.Fatal("UPTIMEROBOT_API_KEY must be set for acceptance tests when TF_ACC=1")
	}
}

func ProviderConfig() string {
	if !needsExplicitProviderSource() {
		return fmt.Sprintf(`
provider "uptimerobot" {
  api_key = "%s"
}
`, os.Getenv("UPTIMEROBOT_API_KEY"))
	}

	source := providerSource()
	version, hasVersion := providerVersion()

	requiredProvider := fmt.Sprintf(`
    uptimerobot = {
      source = "%s"
    }`, source)
	if hasVersion {
		requiredProvider = fmt.Sprintf(`
    uptimerobot = {
      source  = "%s"
      version = "%s"
    }`, source, version)
	}

	return fmt.Sprintf(`
terraform {
  required_providers {
%s
  }
}

provider "uptimerobot" {
  api_key = "%s"
}
`, requiredProvider, os.Getenv("UPTIMEROBOT_API_KEY"))
}

func needsExplicitProviderSource() bool {
	return os.Getenv("TF_ACC_PROVIDER_HOST") != "" ||
		os.Getenv("TF_ACC_PROVIDER_NAMESPACE") != "" ||
		os.Getenv("TF_ACC_PROVIDER_VERSION") != ""
}

func providerSource() string {
	namespace := os.Getenv("TF_ACC_PROVIDER_NAMESPACE")
	if namespace == "" {
		namespace = "uptimerobot"
	}

	host := os.Getenv("TF_ACC_PROVIDER_HOST")
	if host == "" {
		return fmt.Sprintf("%s/uptimerobot", namespace)
	}

	return fmt.Sprintf("%s/%s/uptimerobot", host, namespace)
}

func providerVersion() (string, bool) {
	version := os.Getenv("TF_ACC_PROVIDER_VERSION")
	if version == "" {
		return "", false
	}

	return version, true
}

func APIClient() *client.Client {
	apiClient := client.NewClient(os.Getenv("UPTIMEROBOT_API_KEY"))
	if apiURL := os.Getenv("UPTIMEROBOT_API_URL"); apiURL != "" {
		apiClient.SetBaseURL(apiURL)
	}
	apiClient.SetUserAgent("terraform-provider-uptimerobot/acc-test")
	apiClient.AddHeader("X-Terraform-Provider", "uptimerobot/acc-test")
	return apiClient
}

func RandomName(prefix string) string {
	return sdkacctest.RandomWithPrefix(prefix)
}

func OptionalEnv(key string) (string, bool) {
	v := os.Getenv(key)
	return v, v != ""
}

func HCLStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	out := "{"
	for i, key := range keys {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf(" %q = %q", key, values[key])
	}
	out += " }"
	return out
}

// UniqueURL produces a stable and per-name unique URL to satisfy API
// deduplication validations for GET and HEAD monitors.
func UniqueURL(name string) string {
	if v, ok := uniqueURLCache.Load(name); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	slug := slugify(name)
	if strings.Trim(slug, "-") == "" {
		slug = "monitor"
	}
	suffix := sdkacctest.RandStringFromCharSet(8, sdkacctest.CharSetAlphaNum)
	url := fmt.Sprintf("https://example.com/%s-%s", slug, suffix)
	uniqueURLCache.Store(name, url)
	return url
}

// UniqueDomain returns a unique domain for API validations like DNS monitors.
func UniqueDomain(name string) string {
	if v, ok := uniqueDomainCache.Load(name); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	slug := slugify(name)
	if strings.Trim(slug, "-") == "" {
		slug = "dns"
	}
	suffix := sdkacctest.RandStringFromCharSet(6, sdkacctest.CharSetAlphaNum)
	domain := fmt.Sprintf("%s-%s.example.com", slug, suffix)
	uniqueDomainCache.Store(name, domain)
	return domain
}

func slugify(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, value)
}

var uniqueURLCache sync.Map
var uniqueDomainCache sync.Map

// CheckDestroy functions for each resource type.
func CheckMonitorDestroy(s *terraform.State) error {
	apiClient := APIClient()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_monitor" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting monitor ID to int64: %v", err)
		}

		if err := apiClient.WaitMonitorDeleted(ctx, id, 90*time.Second); err != nil {
			return fmt.Errorf("Monitor %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func CheckMonitorGroupDestroy(s *terraform.State) error {
	apiClient := APIClient()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_monitor_group" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting monitor group ID to int64: %v", err)
		}

		if err := apiClient.WaitMonitorGroupDeleted(ctx, id, 90*time.Second); err != nil {
			return fmt.Errorf("monitor group %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func CheckIntegrationDestroy(s *terraform.State) error {
	apiClient := APIClient()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_integration" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting integration ID to int64: %v", err)
		}

		if err := apiClient.WaitIntegrationDeleted(ctx, id, 90*time.Second); err != nil {
			return fmt.Errorf("Integration %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func CheckMaintenanceWindowDestroy(s *terraform.State) error {
	apiClient := APIClient()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_maintenance_window" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting maintenance window ID to int64: %v", err)
		}

		if err := apiClient.WaitMaintenanceWindowDeleted(ctx, id, 90*time.Second); err != nil {
			return fmt.Errorf("Maintenance window %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}

func CheckPSPDestroy(s *terraform.State) error {
	apiClient := APIClient()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "uptimerobot_psp" {
			continue
		}

		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("Error converting PSP ID to int64: %v", err)
		}

		if err := apiClient.WaitPSPDeleted(ctx, id, 90*time.Second); err != nil {
			return fmt.Errorf("PSP %s still exists: %w", rs.Primary.ID, err)
		}
	}

	return nil
}
