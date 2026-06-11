package container

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestProjectResourceSchemaContentSensitive(t *testing.T) {
	t.Parallel()

	pr, ok := NewProjectResource().(*ProjectResource)
	if !ok {
		t.Fatalf("NewProjectResource did not return *ProjectResource")
	}

	resp := resource.SchemaResponse{}
	pr.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["content"].(rschema.StringAttribute)
	if !ok {
		t.Fatalf("content attribute missing or not a StringAttribute")
	}
	if !attr.Sensitive {
		t.Fatalf(
			"content should be Sensitive: rendered compose YAML carries " +
				"secrets from services.environment / services.secrets",
		)
	}
}

func TestProjectStreamSucceeded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "plain text success stream",
			err: errors.New(
				"unable to decode response: invalid character 'C' looking for beginning of value\n\nContainer vault Stopping\nContainer vault Stopped\nExit Code: 0\n",
			),
			want: true,
		},
		{
			name: "non zero exit code",
			err: errors.New(
				"unable to decode response: invalid character 'C' looking for beginning of value\n\nContainer vault Error\nExit Code: 1\n",
			),
			want: false,
		},
		{
			name: "unrelated error",
			err:  errors.New("api response error code 105: permission denied"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := projectStreamSucceeded(tt.err); got != tt.want {
				t.Fatalf("projectStreamSucceeded() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestNormalizeProjectStreamError(t *testing.T) {
	t.Parallel()

	err := errors.New(
		"unable to decode response: invalid character 'C' looking for beginning of value\n\nContainer minio Recreate\nExit Code: 0\n",
	)

	if got := normalizeProjectStreamError(err); got != nil {
		t.Fatalf("normalizeProjectStreamError() returned %v, want nil", got)
	}
}
