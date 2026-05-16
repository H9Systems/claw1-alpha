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
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	// CLI v1.9.6 uses "RPC Endpoint:" (older used "RPC URL:").
	rpcRe = regexp.MustCompile(`(?:RPC Endpoint|RPC URL):\s+(http://127\.0\.0\.1:\d+/ext/bc/[^\s|]+/rpc)`)
	// CLI v1.9.6 shows the ewoq key in a table without 0x prefix.
	keyRe = regexp.MustCompile(`ewoq\s+\|\s+([a-fA-F0-9]{64})`)
)

type l1Resource struct {
	cfg *ProviderConfig
}

type l1ResourceModel struct {
	Name         types.String `tfsdk:"name"`
	ChainID      types.Int64  `tfsdk:"chain_id"`
	RPCURL       types.String `tfsdk:"rpc_url"`
	SubnetID     types.String `tfsdk:"subnet_id"`
	BlockchainID types.String `tfsdk:"blockchain_id"`
	DeployerKey  types.String `tfsdk:"deployer_key"`
}

// networkJSON is the schema written to ~/.claw1/{name}/network.json.
type networkJSON struct {
	Name            string          `json:"name"`
	ChainID         int64           `json:"chainId"`
	SubnetID        string          `json:"subnetId"`
	BlockchainID    string          `json:"blockchainId"`
	RPCURL          string          `json:"rpcUrl"`
	PlatformRPCURL  string          `json:"platformRpcUrl"`
	DeployerPrivKey string          `json:"deployerPrivateKey"`
	Contracts       []contractEntry `json:"contracts"`
}

type contractEntry struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	DeployedAt string `json:"deployedAt"`
}

func NewL1Resource() resource.Resource {
	return &l1Resource{}
}

func (r *l1Resource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "claw1_l1"
}

func (r *l1Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Deploys a private Avalanche L1 on the local network.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the Avalanche blockchain (used as the L1 identifier).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"chain_id": schema.Int64Attribute{
				Required:    true,
				Description: "EVM chain ID written into the genesis block.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"rpc_url": schema.StringAttribute{
				Computed:    true,
				Description: "HTTP RPC endpoint for the deployed L1.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"blockchain_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deployer_key": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "Funded dev account private key for the local devnet. Do not use in production.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *l1Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.cfg = req.ProviderData.(*ProviderConfig)
}

func (r *l1Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan l1ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	chainID := plan.ChainID.ValueInt64()

	// Idempotent: skip create if L1 already exists.
	if !r.networkExists(name) {
		if err := r.createL1(ctx, name, chainID); err != nil {
			resp.Diagnostics.AddError("L1 create failed", err.Error())
			return
		}
	}

	net, err := r.readNetworkJSON(name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read network.json", err.Error())
		return
	}

	plan.RPCURL = types.StringValue(net.RPCURL)
	plan.SubnetID = types.StringValue(net.SubnetID)
	plan.BlockchainID = types.StringValue(net.BlockchainID)
	plan.DeployerKey = types.StringValue(net.DeployerPrivKey)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *l1Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state l1ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.networkExists(state.Name.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *l1Resource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All fields force replacement; Update is never called.
}

// Delete is state-only. avalanche network clean is a global operation that would
// destroy all local networks on the machine. demo/reset.sh owns actual teardown.
func (r *l1Resource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentional no-op: state removed by framework when this returns without error.
}

func (r *l1Resource) networkExists(name string) bool {
	path := filepath.Join(r.cfg.DataDir, name, "network.json")
	_, err := os.Stat(path)
	return err == nil
}

func (r *l1Resource) createL1(ctx context.Context, name string, chainID int64) error {
	// Step 1: create the blockchain config
	// ewoq is the standard Avalanche local-network funded key.
	const ewoqAddr = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	createArgs := []string{
		"blockchain", "create", name,
		"--evm",
		"--evm-chain-id", fmt.Sprintf("%d", chainID),
		"--evm-token", "CLAW",
		"--test-defaults",
		"--proof-of-authority",
		"--validator-manager-owner", ewoqAddr,
		"--proxy-contract-owner", ewoqAddr,
		"--force",
	}
	createOut, err := exec.CommandContext(ctx, "avalanche", createArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("avalanche blockchain create: %w\n%s", err, createOut)
	}

	// Step 1b: inject TxAllowList into genesis before deploy — fail loudly if this fails.
	if err := injectTxAllowList(name, ewoqAddr); err != nil {
		return fmt.Errorf("inject TxAllowList: %w", err)
	}

	// Step 2: deploy with a 10-minute timeout (deploy takes 60-120s)
	deployCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	deployArgs := []string{"blockchain", "deploy", name, "--local"}
	deployOut, err := exec.CommandContext(deployCtx, "avalanche", deployArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("avalanche blockchain deploy: %w\n%s", err, deployOut)
	}

	rpcURL, deployerKey, err := parseDeployOutput(string(deployOut))
	if err != nil {
		return fmt.Errorf("parse deploy output: %w\nstdout:\n%s", err, deployOut)
	}

	if err := verifyTxAllowListAdmin(ctx, rpcURL, ewoqAddr); err != nil {
		return fmt.Errorf("verify TxAllowList admin: %w", err)
	}

	return r.writeNetworkJSON(name, chainID, rpcURL, deployerKey)
}

const txAllowListPrecompileAddress = "0x0200000000000000000000000000000000000002"

func verifyTxAllowListAdmin(ctx context.Context, rpcURL, adminAddr string) error {
	role, err := readTxAllowListRole(ctx, rpcURL, adminAddr)
	if err != nil {
		return err
	}
	if role != 3 {
		return fmt.Errorf("expected %s to have TxAllowList admin role 3, got %d", adminAddr, role)
	}
	return nil
}

func readTxAllowListRole(ctx context.Context, rpcURL, addr string) (uint64, error) {
	callData, err := encodeReadAllowListCall(addr)
	if err != nil {
		return 0, err
	}

	reqBody := fmt.Sprintf(
		`{"jsonrpc":"2.0","method":"eth_call","params":[{"to":%q,"data":%q},"latest"],"id":1}`,
		txAllowListPrecompileAddress,
		callData,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewBufferString(reqBody))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("eth_call readAllowList: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, fmt.Errorf("decode eth_call readAllowList response: %w", err)
	}
	if out.Error != nil {
		return 0, fmt.Errorf("eth_call readAllowList returned %d: %s", out.Error.Code, out.Error.Message)
	}
	if out.Result == "" || out.Result == "0x" {
		return 0, fmt.Errorf("eth_call readAllowList returned empty result")
	}

	return strconv.ParseUint(strings.TrimPrefix(out.Result, "0x"), 16, 64)
}

func encodeReadAllowListCall(addr string) (string, error) {
	addr = strings.TrimPrefix(strings.ToLower(addr), "0x")
	if len(addr) != 40 {
		return "", fmt.Errorf("invalid address length for %q", addr)
	}
	if _, err := strconv.ParseUint(addr[:16], 16, 64); err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if _, err := strconv.ParseUint(addr[16:32], 16, 64); err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}
	if _, err := strconv.ParseUint(addr[32:], 16, 64); err != nil {
		return "", fmt.Errorf("invalid address %q: %w", addr, err)
	}
	return "0xeb54dae1" + strings.Repeat("0", 24) + addr, nil
}

func parseDeployOutput(output string) (rpcURL, deployerKey string, err error) {
	if m := rpcRe.FindStringSubmatch(output); len(m) == 2 {
		rpcURL = m[1]
	}
	if m := keyRe.FindStringSubmatch(output); len(m) == 2 {
		deployerKey = m[1]
	}
	if rpcURL == "" || deployerKey == "" {
		return "", "", fmt.Errorf("could not parse RPC URL or private key from output")
	}
	return rpcURL, deployerKey, nil
}

func (r *l1Resource) writeNetworkJSON(name string, chainID int64, rpcURL, deployerKey string) error {
	dir := filepath.Join(r.cfg.DataDir, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	net := networkJSON{
		Name:            name,
		ChainID:         chainID,
		RPCURL:          rpcURL,
		PlatformRPCURL:  "http://127.0.0.1:9650",
		DeployerPrivKey: deployerKey,
		Contracts:       []contractEntry{},
	}

	data, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "network.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (r *l1Resource) readNetworkJSON(name string) (*networkJSON, error) {
	path := filepath.Join(r.cfg.DataDir, name, "network.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var net networkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &net, nil
}

// injectTxAllowList patches the genesis.json written by avalanche-cli to add
// txAllowListConfig with adminAddr as the sole admin. Must be called after
// `avalanche blockchain create` and before `avalanche blockchain deploy`.
func injectTxAllowList(name, adminAddr string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	genesisPath := filepath.Join(home, ".avalanche-cli", "subnets", name, "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("read genesis: %w", err)
	}

	var genesis map[string]interface{}
	if err := json.Unmarshal(data, &genesis); err != nil {
		return fmt.Errorf("parse genesis: %w", err)
	}

	cfg, _ := genesis["config"].(map[string]interface{})
	if cfg == nil {
		cfg = make(map[string]interface{})
		genesis["config"] = cfg
	}
	cfg["txAllowListConfig"] = map[string]interface{}{
		"blockTimestamp": 0,
		"adminAddresses": []string{adminAddr},
	}

	updated, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}

	tmp := genesisPath + ".tmp"
	if err := os.WriteFile(tmp, updated, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, genesisPath)
}
