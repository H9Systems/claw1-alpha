package provider

import (
	"context"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type ProviderConfig struct {
	DataDir string
}

type claw1Provider struct{}

func New() provider.Provider {
	return &claw1Provider{}
}

func (p *claw1Provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "claw1"
}

func (p *claw1Provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *claw1Provider) Configure(ctx context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	dataDir := os.Getenv("CLAW1_DATA_DIR")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".claw1")
	}
	cfg := &ProviderConfig{DataDir: dataDir}
	resp.DataSourceData = cfg
	resp.ResourceData = cfg
}

func (p *claw1Provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewL1Resource,
		NewContractResource,
	}
}

func (p *claw1Provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
