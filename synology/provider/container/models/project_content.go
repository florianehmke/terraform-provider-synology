package models

import (
	"context"
	"strings"

	"github.com/florianehmke/terraform-provider-synology/synology/models/composetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

func HydrateProjectResourceModelFromContent(
	ctx context.Context,
	model *ProjectResourceModel,
	content string,
) diag.Diagnostics {
	existingConfigs := model.Configs
	existingSecrets := model.Secrets

	resetProjectResourceCollections(model)

	if strings.TrimSpace(content) == "" {
		return nil
	}

	var project composetypes.Project
	if err := yaml.Unmarshal([]byte(content), &project); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("Failed to parse project content", err.Error()),
		}
	}

	normalizeComposeProject(&project)

	diags := diag.Diagnostics{}

	services, servicesDiags := servicesMapFromCompose(ctx, project.Services)
	diags.Append(servicesDiags...)
	model.Services = services

	networks, networksDiags := networksMapFromCompose(ctx, project.Networks)
	diags.Append(networksDiags...)
	model.Networks = networks

	volumes, volumesDiags := volumesMapFromCompose(ctx, project.Volumes)
	diags.Append(volumesDiags...)
	model.Volumes = volumes

	configs, configsDiags := configsMapFromCompose(ctx, project.Configs)
	diags.Append(configsDiags...)
	model.Configs = configs
	diags.Append(mergeExistingConfigContent(ctx, existingConfigs, &model.Configs)...)

	secrets, secretsDiags := secretsMapFromCompose(ctx, project.Secrets)
	diags.Append(secretsDiags...)
	model.Secrets = secrets
	diags.Append(mergeExistingSecretContent(ctx, existingSecrets, &model.Secrets)...)

	return diags
}

func resetProjectResourceCollections(model *ProjectResourceModel) {
	model.Services = types.MapNull(Service{}.ModelType())
	model.Networks = types.MapNull(Network{}.ModelType())
	model.Volumes = types.MapNull(Volume{}.ModelType())
	model.Configs = types.MapNull(Config{}.ModelType())
	model.Secrets = types.MapNull(Secret{}.ModelType())
}

func normalizeComposeProject(project *composetypes.Project) {
	for name, service := range project.Services {
		service.Name = name
		project.Services[name] = service
	}

	for name, network := range project.Networks {
		if network.Name == "" {
			network.Name = name
		}
		project.Networks[name] = network
	}

	for name, volume := range project.Volumes {
		if volume.Name == "" {
			volume.Name = name
		}
		project.Volumes[name] = volume
	}

	for name, config := range project.Configs {
		if config.Name == "" {
			config.Name = name
		}
		project.Configs[name] = config
	}

	for name, secret := range project.Secrets {
		if secret.Name == "" {
			secret.Name = name
		}
		project.Secrets[name] = secret
	}
}

func mergeExistingConfigContent(
	ctx context.Context,
	existing types.Map,
	current *types.Map,
) diag.Diagnostics {
	if existing.IsNull() || existing.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return nil
	}

	existingValues := map[string]Config{}
	diags := existing.ElementsAs(ctx, &existingValues, true)
	if diags.HasError() {
		return diags
	}

	currentValues := map[string]Config{}
	diags.Append(current.ElementsAs(ctx, &currentValues, true)...)
	if diags.HasError() {
		return diags
	}

	changed := false
	for key, currentValue := range currentValues {
		existingValue, ok := existingValues[key]
		if !ok {
			continue
		}

		if currentValue.Content.IsNull() || currentValue.Content.IsUnknown() || currentValue.Content.ValueString() == "" {
			if !existingValue.Content.IsNull() && !existingValue.Content.IsUnknown() && existingValue.Content.ValueString() != "" {
				currentValue.Content = existingValue.Content
				currentValues[key] = currentValue
				changed = true
			}
		}
	}

	if !changed {
		return diags
	}

	mapValue, mapDiags := types.MapValueFrom(ctx, Config{}.ModelType(), currentValues)
	diags.Append(mapDiags...)
	if !diags.HasError() {
		*current = mapValue
	}

	return diags
}

func mergeExistingSecretContent(
	ctx context.Context,
	existing types.Map,
	current *types.Map,
) diag.Diagnostics {
	if existing.IsNull() || existing.IsUnknown() || current.IsNull() || current.IsUnknown() {
		return nil
	}

	existingValues := map[string]Secret{}
	diags := existing.ElementsAs(ctx, &existingValues, true)
	if diags.HasError() {
		return diags
	}

	currentValues := map[string]Secret{}
	diags.Append(current.ElementsAs(ctx, &currentValues, true)...)
	if diags.HasError() {
		return diags
	}

	changed := false
	for key, currentValue := range currentValues {
		existingValue, ok := existingValues[key]
		if !ok {
			continue
		}

		if currentValue.Content.IsNull() || currentValue.Content.IsUnknown() || currentValue.Content.ValueString() == "" {
			if !existingValue.Content.IsNull() && !existingValue.Content.IsUnknown() && existingValue.Content.ValueString() != "" {
				currentValue.Content = existingValue.Content
				currentValues[key] = currentValue
				changed = true
			}
		}
	}

	if !changed {
		return diags
	}

	mapValue, mapDiags := types.MapValueFrom(ctx, Secret{}.ModelType(), currentValues)
	diags.Append(mapDiags...)
	if !diags.HasError() {
		*current = mapValue
	}

	return diags
}

func servicesMapFromCompose(ctx context.Context, services composetypes.Services) (types.Map, diag.Diagnostics) {
	if len(services) == 0 {
		return types.MapNull(Service{}.ModelType()), nil
	}

	values := map[string]Service{}
	diags := diag.Diagnostics{}

	for name, service := range services {
		model := Service{}
		diags.Append(model.FromComposeConfig(ctx, &service)...)
		values[name] = model
	}

	mapValue, moreDiags := types.MapValueFrom(ctx, Service{}.ModelType(), values)
	diags.Append(moreDiags...)

	return mapValue, diags
}

func networksMapFromCompose(ctx context.Context, networks composetypes.Networks) (types.Map, diag.Diagnostics) {
	if len(networks) == 0 {
		return types.MapNull(Network{}.ModelType()), nil
	}

	values := map[string]Network{}
	diags := diag.Diagnostics{}

	for name, network := range networks {
		model := Network{}
		diags.Append(model.FromComposeConfig(ctx, &network)...)
		values[name] = model
	}

	mapValue, moreDiags := types.MapValueFrom(ctx, Network{}.ModelType(), values)
	diags.Append(moreDiags...)

	return mapValue, diags
}

func volumesMapFromCompose(ctx context.Context, volumes composetypes.Volumes) (types.Map, diag.Diagnostics) {
	if len(volumes) == 0 {
		return types.MapNull(Volume{}.ModelType()), nil
	}

	values := map[string]Volume{}
	diags := diag.Diagnostics{}

	for name, volume := range volumes {
		model := Volume{}
		diags.Append(model.FromComposeConfig(ctx, &volume)...)
		values[name] = model
	}

	mapValue, moreDiags := types.MapValueFrom(ctx, Volume{}.ModelType(), values)
	diags.Append(moreDiags...)

	return mapValue, diags
}

func configsMapFromCompose(ctx context.Context, configs composetypes.Configs) (types.Map, diag.Diagnostics) {
	if len(configs) == 0 {
		return types.MapNull(Config{}.ModelType()), nil
	}

	values := map[string]Config{}
	diags := diag.Diagnostics{}

	for name, config := range configs {
		model := Config{}
		diags.Append(model.FromComposeConfig(ctx, &config)...)
		values[name] = model
	}

	mapValue, moreDiags := types.MapValueFrom(ctx, Config{}.ModelType(), values)
	diags.Append(moreDiags...)

	return mapValue, diags
}

func secretsMapFromCompose(ctx context.Context, secrets composetypes.Secrets) (types.Map, diag.Diagnostics) {
	if len(secrets) == 0 {
		return types.MapNull(Secret{}.ModelType()), nil
	}

	values := map[string]Secret{}
	diags := diag.Diagnostics{}

	for name, secret := range secrets {
		model := Secret{}
		diags.Append(model.FromComposeConfig(ctx, &secret)...)
		values[name] = model
	}

	mapValue, moreDiags := types.MapValueFrom(ctx, Secret{}.ModelType(), values)
	diags.Append(moreDiags...)

	return mapValue, diags
}
