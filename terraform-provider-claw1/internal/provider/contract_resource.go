package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var deployedToRe = regexp.MustCompile(`Deployed to:\s+(0x[a-fA-F0-9]{40})`)

type contractResource struct {
	cfg *ProviderConfig
}

type contractResourceModel struct {
	Source          types.String `tfsdk:"source"`
	Name            types.String `tfsdk:"name"`
	RPCURL          types.String `tfsdk:"rpc_url"`
	DeployerKey     types.String `tfsdk:"deployer_key"`
	ConstructorArgs types.List   `tfsdk:"constructor_args"`
	Address         types.String `tfsdk:"address"`
}

func NewContractResource() resource.Resource {
	return &contractResource{}
}

func (r *contractResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "claw1_contract"
}

func (r *contractResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a Solidity contract via forge create onto a claw1 L1.",
		Attributes: map[string]schema.Attribute{
			"source": schema.StringAttribute{
				Required:    true,
				Description: "Path to the .sol file to deploy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Contract name as it appears in the Solidity source.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rpc_url": schema.StringAttribute{
				Required:    true,
				Description: "RPC URL of the target L1 (from claw1_l1.rpc_url).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"deployer_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Private key of the funded deployer account (from claw1_l1.deployer_key).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"constructor_args": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arguments passed to the contract constructor (as strings, in order).",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"address": schema.StringAttribute{
				Computed:    true,
				Description: "Deployed contract address (0x...).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *contractResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.cfg = req.ProviderData.(*ProviderConfig)
}

func (r *contractResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contractResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rpcURL := plan.RPCURL.ValueString()

	// D8: poll eth_chainId until RPC accepts connections before invoking forge create.
	if err := waitForRPC(ctx, rpcURL, 30*time.Second); err != nil {
		resp.Diagnostics.AddError("RPC not ready", err.Error())
		return
	}

	var ctorArgs []string
	if !plan.ConstructorArgs.IsNull() && !plan.ConstructorArgs.IsUnknown() {
		diags := plan.ConstructorArgs.ElementsAs(ctx, &ctorArgs, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	addr, err := r.deployContract(ctx, plan, ctorArgs)
	if err != nil {
		resp.Diagnostics.AddError("Contract deploy failed", err.Error())
		return
	}

	plan.Address = types.StringValue(addr)

	// Persist the contract address to network.json so other tools can find it.
	l1Name := r.l1NameFromRPC(rpcURL)
	if l1Name == "" {
		resp.Diagnostics.AddError("Failed to update network.json", fmt.Sprintf("could not find L1 network.json for RPC URL %s", rpcURL))
		return
	}
	if err := r.appendContractToNetworkJSON(l1Name, plan.Name.ValueString(), addr); err != nil {
		resp.Diagnostics.AddError("Failed to update network.json", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contractResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contractResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exists, err := contractCodeExists(ctx, state.RPCURL.ValueString(), state.Address.ValueString())
	if err != nil || !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contractResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields force replacement; Update is never called.
}

// Delete is state-only. Contracts are immutable on-chain; there is no undeploy.
// After terraform destroy + apply, Terraform redeploys to a new address.
func (r *contractResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentional no-op.
}

func (r *contractResource) deployContract(ctx context.Context, plan contractResourceModel, ctorArgs []string) (string, error) {
	l1Name := r.l1NameFromRPC(plan.RPCURL.ValueString())
	logPath := ""
	if l1Name != "" && r.cfg != nil {
		logPath = filepath.Join(r.cfg.DataDir, l1Name, "contract-deploy.log")
		_ = os.MkdirAll(filepath.Dir(logPath), 0700)
	}

	src := plan.Source.ValueString()
	name := plan.Name.ValueString()

	// forge create <file:ContractName> --root <foundry-project-root>
	// Source is e.g. "./../contracts/src/Foo.sol"; the Foundry root is one level
	// above src/ (i.e. the directory that contains foundry.toml).
	contractArg := name // fallback: bare name
	var rootArgs []string
	if strings.HasSuffix(src, ".sol") {
		srcDir := filepath.Dir(src)         // ./../contracts/src
		projectRoot := filepath.Dir(srcDir) // ./../contracts
		relPath := filepath.Base(srcDir) + string(filepath.Separator) + filepath.Base(src)
		contractArg = relPath + ":" + name // src/Foo.sol:Foo
		rootArgs = []string{"--root", projectRoot}
	}

	args := []string{
		"create",
		contractArg,
		"--rpc-url", plan.RPCURL.ValueString(),
		"--broadcast",
	}
	args = append(args, rootArgs...)
	if len(ctorArgs) > 0 {
		args = append(args, "--constructor-args")
		args = append(args, ctorArgs...)
	}

	cmd := exec.CommandContext(ctx, "forge", args...)
	// Pass key via env var to avoid exposure in ps aux.
	cmd.Env = append(os.Environ(), "FOUNDRY_ETH_PRIVATE_KEY="+plan.DeployerKey.ValueString())
	output, err := cmd.CombinedOutput()

	if logPath != "" {
		_ = os.WriteFile(logPath, output, 0644)
	}

	if err != nil {
		return "", fmt.Errorf("forge create: %w\n%s", err, output)
	}

	m := deployedToRe.FindStringSubmatch(string(output))
	if len(m) != 2 {
		return "", fmt.Errorf("could not parse contract address from forge output:\n%s", output)
	}

	return m[1], nil
}

// waitForRPC polls eth_chainId until the RPC endpoint responds or deadline is reached.
func waitForRPC(ctx context.Context, rpcURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	body := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`

	for time.Now().Before(deadline) {
		r, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewBufferString(body))
		if err != nil {
			return err
		}
		r.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(r)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return fmt.Errorf("RPC at %s did not respond within %s", rpcURL, timeout)
}

// contractCodeExists returns true if eth_getCode returns non-empty bytecode.
func contractCodeExists(ctx context.Context, rpcURL, address string) (bool, error) {
	reqBody := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_getCode","params":[%q,"latest"],"id":1}`,
		address,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewBufferString(reqBody))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil // treat network error as "not found" — triggers recreation
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, nil
	}

	return result.Result != "" && result.Result != "0x", nil
}

// l1NameFromRPC extracts the network name from network.json by scanning known dirs.
// Returns "" if not determinable (non-fatal — log file just won't be written).
func (r *contractResource) l1NameFromRPC(rpcURL string) string {
	if r.cfg == nil {
		return ""
	}
	entries, err := os.ReadDir(r.cfg.DataDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		net, err := r.readNetworkJSONDirect(filepath.Join(r.cfg.DataDir, e.Name(), "network.json"))
		if err != nil {
			continue
		}
		if net.RPCURL == rpcURL {
			return e.Name()
		}
	}
	return ""
}

func (r *contractResource) readNetworkJSONDirect(path string) (*networkJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var net networkJSON
	return &net, json.Unmarshal(data, &net)
}

func (r *contractResource) appendContractToNetworkJSON(l1Name, contractName, address string) error {
	if r.cfg == nil {
		return nil
	}
	path := filepath.Join(r.cfg.DataDir, l1Name, "network.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var net networkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return err
	}

	net.Contracts = append(net.Contracts, contractEntry{
		Name:       contractName,
		Address:    address,
		DeployedAt: time.Now().UTC().Format(time.RFC3339),
	})

	updated, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, updated, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
