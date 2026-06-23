package apps

import (
	"testing"

	"cloudiac/portal/models"
)

func TestOciOkeClusterEndpointPrefersKubernetesEndpoint(t *testing.T) {
	endpoint := ociOkeClusterEndpoint(models.ResAttrs{
		"endpoints": models.ResAttrs{
			"kubernetes":      "https://oke-api.example.com:6443",
			"publicEndpoint":  "https://public.example.com:6443",
			"privateEndpoint": "https://private.example.com:6443",
		},
	})
	if endpoint != "https://oke-api.example.com:6443" {
		t.Fatalf("expected kubernetes endpoint, got %q", endpoint)
	}
}

func TestOciOkeClusterEndpointFallsBackToPublicEndpoint(t *testing.T) {
	endpoint := ociOkeClusterEndpoint(models.ResAttrs{
		"endpoints": map[string]interface{}{
			"publicEndpoint": "https://public.example.com:6443",
		},
	})
	if endpoint != "https://public.example.com:6443" {
		t.Fatalf("expected public endpoint fallback, got %q", endpoint)
	}
}
