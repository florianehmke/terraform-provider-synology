package models

import (
	"context"
	"testing"
)

func TestHydrateProjectResourceModelFromContent(t *testing.T) {
	t.Parallel()

	content := `
services:
  vault:
    image: hashicorp/vault:1.19
    environment:
      VAULT_ADDR: https://vault.synology.example.com
    ports:
      - target: 8200
        published: "8200"
        protocol: tcp
        host_ip: 127.0.0.1
configs:
  vault_hcl:
    name: vault_hcl
    content: |
      api_addr = "https://vault.synology.example.com"
secrets:
  unseal_key:
    name: unseal_key
    file: unseal_key
volumes:
  data:
    name: data
`

	ctx := context.Background()
	model := ProjectResourceModel{}

	diags := HydrateProjectResourceModelFromContent(ctx, &model, content)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	services := map[string]Service{}
	diags = model.Services.ElementsAs(ctx, &services, true)
	if diags.HasError() {
		t.Fatalf("failed to decode services: %v", diags)
	}

	vault, ok := services["vault"]
	if !ok {
		t.Fatalf("expected vault service to be present")
	}

	if got := vault.Image.ValueString(); got != "hashicorp/vault:1.19" {
		t.Fatalf("vault image = %q, want %q", got, "hashicorp/vault:1.19")
	}

	environment := map[string]string{}
	diags = vault.Environment.ElementsAs(ctx, &environment, true)
	if diags.HasError() {
		t.Fatalf("failed to decode environment: %v", diags)
	}

	if got := environment["VAULT_ADDR"]; got != "https://vault.synology.example.com" {
		t.Fatalf("VAULT_ADDR = %q, want %q", got, "https://vault.synology.example.com")
	}

	ports := []Port{}
	diags = vault.Ports.ElementsAs(ctx, &ports, true)
	if diags.HasError() {
		t.Fatalf("failed to decode ports: %v", diags)
	}

	if len(ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(ports))
	}

	if got := ports[0].HostIP.ValueString(); got != "127.0.0.1" {
		t.Fatalf("host_ip = %q, want %q", got, "127.0.0.1")
	}

	configs := map[string]Config{}
	diags = model.Configs.ElementsAs(ctx, &configs, true)
	if diags.HasError() {
		t.Fatalf("failed to decode configs: %v", diags)
	}

	if got := configs["vault_hcl"].Content.ValueString(); got != "api_addr = \"https://vault.synology.example.com\"\n" {
		t.Fatalf("vault_hcl content = %q", got)
	}

	volumes := map[string]Volume{}
	diags = model.Volumes.ElementsAs(ctx, &volumes, true)
	if diags.HasError() {
		t.Fatalf("failed to decode volumes: %v", diags)
	}

	if _, ok := volumes["data"]; !ok {
		t.Fatalf("expected data volume to be present")
	}
}

func TestHydrateProjectResourceModelFromContentEmpty(t *testing.T) {
	t.Parallel()

	model := ProjectResourceModel{}
	diags := HydrateProjectResourceModelFromContent(context.Background(), &model, "")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if !model.Services.IsNull() {
		t.Fatalf("expected services to be null")
	}

	if !model.Configs.IsNull() {
		t.Fatalf("expected configs to be null")
	}

	if !model.Secrets.IsNull() {
		t.Fatalf("expected secrets to be null")
	}
}
