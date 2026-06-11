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

func TestHydrateProjectResourceModelFromContentDependsOnNullable(t *testing.T) {
	t.Parallel()

	content := `
services:
  app:
    image: example/app:1
    depends_on:
      db:
        condition: service_started
      cache:
        condition: service_started
        restart: true
  db:
    image: example/db:1
  cache:
    image: example/cache:1
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

	app, ok := services["app"]
	if !ok {
		t.Fatalf("expected app service to be present")
	}

	deps := map[string]ServiceDependency{}
	diags = app.Dependencies.ElementsAs(ctx, &deps, true)
	if diags.HasError() {
		t.Fatalf("failed to decode dependencies: %v", diags)
	}

	dbDep, ok := deps["db"]
	if !ok {
		t.Fatalf("expected db dependency to be present")
	}
	if dbDep.Condition.IsNull() || dbDep.Condition.ValueString() != "service_started" {
		t.Fatalf("db condition = %v, want known service_started", dbDep.Condition)
	}
	if !dbDep.Restart.IsNull() {
		t.Fatalf("db restart should be null when unset in YAML, got %v", dbDep.Restart)
	}

	cacheDep, ok := deps["cache"]
	if !ok {
		t.Fatalf("expected cache dependency to be present")
	}
	if cacheDep.Restart.IsNull() || !cacheDep.Restart.ValueBool() {
		t.Fatalf("cache restart should be known true, got %v", cacheDep.Restart)
	}
}

func TestHydrateProjectResourceModelFromContentVolumeWithoutBind(t *testing.T) {
	t.Parallel()

	// A service with one volume that has a bind block and a second volume
	// without one — the volumes list must be typeable as a single ServiceVolume
	// schema, so the bind-less entry must carry a TYPED null Object, not a
	// schema-less zero value (which trips ListValueFrom with a type mismatch).
	content := `
services:
  app:
    image: example/app:1
    volumes:
      - type: bind
        source: /srv/data
        target: /data
        bind:
          create_host_path: true
      - type: bind
        source: /srv/source
        target: /input
        read_only: true
`

	ctx := context.Background()
	model := ProjectResourceModel{}

	diags := HydrateProjectResourceModelFromContent(ctx, &model, content)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if model.Services.IsNull() {
		t.Fatalf("expected services to be populated")
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

func TestHydrateProjectResourceModelFromContentNetworksNullable(t *testing.T) {
	t.Parallel()

	// A network declared without an explicit name, and a service attached to a
	// network without explicit aliases, must hydrate as TYPED NULLS so the
	// post-apply state matches a plan that left them unset. Otherwise the
	// round-trip injects the map key as the network name and the service name
	// as a network alias, and Terraform reports "inconsistent result after
	// apply" for networks[*].name and the sensitive services attribute.
	content := `
services:
  app:
    image: example/app:1
    networks:
      appnet: {}
networks:
  appnet:
    driver: bridge
`

	ctx := context.Background()
	model := ProjectResourceModel{}

	diags := HydrateProjectResourceModelFromContent(ctx, &model, content)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	networks := map[string]Network{}
	diags = model.Networks.ElementsAs(ctx, &networks, true)
	if diags.HasError() {
		t.Fatalf("failed to decode networks: %v", diags)
	}

	appnet, ok := networks["appnet"]
	if !ok {
		t.Fatalf("expected appnet network to be present")
	}
	if !appnet.Name.IsNull() {
		t.Fatalf("network name should be null when unset in YAML, got %v", appnet.Name)
	}
	if appnet.Driver.IsNull() || appnet.Driver.ValueString() != "bridge" {
		t.Fatalf("network driver = %v, want known bridge", appnet.Driver)
	}

	services := map[string]Service{}
	diags = model.Services.ElementsAs(ctx, &services, true)
	if diags.HasError() {
		t.Fatalf("failed to decode services: %v", diags)
	}

	app, ok := services["app"]
	if !ok {
		t.Fatalf("expected app service to be present")
	}

	serviceNetworks := map[string]ServiceNetwork{}
	diags = app.Networks.ElementsAs(ctx, &serviceNetworks, true)
	if diags.HasError() {
		t.Fatalf("failed to decode service networks: %v", diags)
	}

	appAttach, ok := serviceNetworks["appnet"]
	if !ok {
		t.Fatalf("expected app to be attached to appnet")
	}
	if !appAttach.Aliases.IsNull() {
		t.Fatalf(
			"service network aliases should be null when unset in YAML, got %v",
			appAttach.Aliases,
		)
	}
}
