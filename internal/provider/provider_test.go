package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is the entrypoint for terraform-plugin-testing
// based acceptance tests. Acceptance tests aren't run as part of the default
// `go test` suite; add `TF_ACC=1` and target this package to exercise the real
// GoDaddy API.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"godaddy": providerserver.NewProtocol6WithError(New("test")()),
}

func TestProvider(t *testing.T) {
	if _, err := testAccProtoV6ProviderFactories["godaddy"](); err != nil {
		t.Fatalf("provider server failed to construct: %s", err)
	}
}
