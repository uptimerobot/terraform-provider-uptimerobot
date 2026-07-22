package monitor

import (
	"strings"
	"testing"

	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAPIAssertionsSchemaSensitivityAndInternalMetadataBoundary(t *testing.T) {
	t.Parallel()

	s := monitorSchema(6, true)
	if s.Version != 6 {
		t.Fatalf("API assertions v2 must not require a state-shape version bump: got schema version %d", s.Version)
	}
	config, ok := s.Attributes["config"].(resourceschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("config schema has unexpected type %T", s.Attributes["config"])
	}
	assertions, ok := config.Attributes["api_assertions"].(resourceschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("api_assertions schema has unexpected type %T", config.Attributes["api_assertions"])
	}
	for _, forbidden := range []string{"semantics_version", "semanticsVersion", "diagnostics"} {
		if _, exists := assertions.Attributes[forbidden]; exists {
			t.Fatalf("API Internal-owned field %q must not enter Terraform configuration or state", forbidden)
		}
	}
	checks, ok := assertions.Attributes["checks"].(resourceschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("checks schema has unexpected type %T", assertions.Attributes["checks"])
	}
	if _, ok := checks.CustomType.(apiAssertionChecksType); !ok {
		t.Fatalf("checks must use duplicate-preserving order-insensitive semantic equality, got %T", checks.CustomType)
	}
	for _, forbidden := range []string{"id", "index", "diagnostics"} {
		if _, exists := checks.NestedObject.Attributes[forbidden]; exists {
			t.Fatalf("internal check field %q must not enter Terraform configuration or state", forbidden)
		}
	}
	target, ok := checks.NestedObject.Attributes["target"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("target schema has unexpected type %T", checks.NestedObject.Attributes["target"])
	}
	if !target.Sensitive {
		t.Fatal("assertion target must be marked sensitive")
	}
	if !strings.Contains(strings.ToLower(target.Description), "terraform state") {
		t.Fatalf("target documentation must disclose nested state exposure: %q", target.Description)
	}
	property, ok := checks.NestedObject.Attributes["property"].(resourceschema.StringAttribute)
	if !ok || property.Sensitive {
		t.Fatal("property must remain visible for useful diffs while its selector exposure is documented")
	}
}
