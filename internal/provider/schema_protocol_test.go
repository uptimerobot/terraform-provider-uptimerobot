package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProtocol6ProviderSchemaIsValid(t *testing.T) {
	t.Parallel()

	server := providerserver.NewProtocol6(New("test")())()
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic == nil || diagnostic.Severity != tfprotov6.DiagnosticSeverityError {
			continue
		}
		t.Errorf("provider schema error: %s: %s", diagnostic.Summary, diagnostic.Detail)
	}
}
