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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
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
	// Identity — required, genesis-time.
	Name    types.String `tfsdk:"name"`
	ChainID types.Int64  `tfsdk:"chain_id"`

	// EVM genesis — optional, genesis-time.
	TokenSymbol        types.String `tfsdk:"token_symbol"`
	VMVersion          types.String `tfsdk:"vm_version"`
	ProductionDefaults types.Bool   `tfsdk:"production_defaults"`

	// Precompiles — optional, genesis-time.
	EnableWarp  types.Bool `tfsdk:"enable_warp"`
	EnableICM   types.Bool `tfsdk:"enable_icm"`
	EnableDebug types.Bool `tfsdk:"enable_debug"`

	// Consensus — optional, genesis-time.
	// Valid values: "poa" (default), "pos-native", "pos-erc20".
	Consensus             types.String `tfsdk:"consensus"`
	Sovereign             types.Bool   `tfsdk:"sovereign"`
	ValidatorManagerOwner types.String `tfsdk:"validator_manager_owner"`
	ProxyContractOwner    types.String `tfsdk:"proxy_contract_owner"`

	// PoS ERC20 staking token — only used when consensus = "pos-erc20".
	ERC20TokenAddress types.String `tfsdk:"erc20_token_address"`
	ERC20TokenSupply  types.Int64  `tfsdk:"erc20_token_supply"`
	ERC20TokenSymbol  types.String `tfsdk:"erc20_token_symbol"`
	RewardBasisPoints types.Int64  `tfsdk:"reward_basis_points"`

	// Custom genesis file — optional. When set, the CLI uses this file directly and
	// the automatic TxAllowList / Warp genesis injection is skipped.
	GenesisFile types.String `tfsdk:"genesis_file"`

	// Network / deploy target — optional, deploy-time.
	// network valid values: "local" (default), "devnet", "testnet", "mainnet".
	Network            types.String `tfsdk:"network"`
	Endpoint           types.String `tfsdk:"endpoint"`
	Cluster            types.String `tfsdk:"cluster"`
	AvalancheGoVersion types.String `tfsdk:"avalanchego_version"`

	// Bootstrap validators — optional, deploy-time.
	NumNodes                  types.Int64   `tfsdk:"num_nodes"`
	NumBootstrapValidators    types.Int64   `tfsdk:"num_bootstrap_validators"`
	BootstrapValidatorBalance types.Float64 `tfsdk:"bootstrap_validator_balance"`
	BootstrapValidatorWeight  types.Int64   `tfsdk:"bootstrap_validator_weight"`
	BootstrapFilepath         types.String  `tfsdk:"bootstrap_filepath"`
	ChangeOwnerAddress        types.String  `tfsdk:"change_owner_address"`

	// Computed outputs.
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
	replace := func() []planmodifier.String { return []planmodifier.String{stringplanmodifier.RequiresReplace()} }
	replaceB := func() []planmodifier.Bool { return []planmodifier.Bool{boolplanmodifier.RequiresReplace()} }
	replaceI := func() []planmodifier.Int64 { return []planmodifier.Int64{int64planmodifier.RequiresReplace()} }
	replaceF := func() []planmodifier.Float64 { return []planmodifier.Float64{float64planmodifier.RequiresReplace()} }

	resp.Schema = schema.Schema{
		Description: "Deploys a private Avalanche L1. Exposes the full avalanche-cli parameter surface.",
		Attributes: map[string]schema.Attribute{

			// ── Identity ────────────────────────────────────────────────────────────────

			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Blockchain identifier used by avalanche-cli.",
				PlanModifiers: replace(),
			},
			"chain_id": schema.Int64Attribute{
				Required:      true,
				Description:   "EVM chain ID written into the genesis block.",
				PlanModifiers: replaceI(),
			},

			// ── EVM genesis ──────────────────────────────────────────────────────────────

			"token_symbol": schema.StringAttribute{
				Optional:      true,
				Description:   "EVM native token symbol (--evm-token). Defaults to \"CLAW\".",
				PlanModifiers: replace(),
			},
			"vm_version": schema.StringAttribute{
				Optional:      true,
				Description:   "Specific subnet-evm release to pin (--vm-version). Omit to use the CLI default.",
				PlanModifiers: replace(),
			},
			"production_defaults": schema.BoolAttribute{
				Optional:      true,
				Description:   "Use --production-defaults instead of --test-defaults. Off by default (test settings).",
				PlanModifiers: replaceB(),
			},

			// ── Precompiles ──────────────────────────────────────────────────────────────

			"enable_warp": schema.BoolAttribute{
				Optional:      true,
				Description:   "Generate subnet-evm with Avalanche Warp Messaging support (--warp). Required for ICM / ICTT bridges.",
				PlanModifiers: replaceB(),
			},
			"enable_icm": schema.BoolAttribute{
				Optional:      true,
				Description:   "Deploy the ICM registry contract at genesis (--icm, implies --warp). Experimental.",
				PlanModifiers: replaceB(),
			},
			"enable_debug": schema.BoolAttribute{
				Optional:      true,
				Description:   "Enable blockchain debugging (--debug). Defaults to true.",
				PlanModifiers: replaceB(),
			},

			// ── Consensus ────────────────────────────────────────────────────────────────

			"consensus": schema.StringAttribute{
				Optional:      true,
				Description:   "Validator management model. One of: \"poa\" (default), \"pos-native\", \"pos-erc20\".",
				PlanModifiers: replace(),
			},
			"sovereign": schema.BoolAttribute{
				Optional:      true,
				Description:   "Deploy as a sovereign Subnet-Only-Validator (--sovereign). Defaults to true.",
				PlanModifiers: replaceB(),
			},
			"validator_manager_owner": schema.StringAttribute{
				Optional:      true,
				Description:   "EVM address that controls the ValidatorManager proxy owner (--validator-manager-owner). Defaults to the ewoq dev address.",
				PlanModifiers: replace(),
			},
			"proxy_contract_owner": schema.StringAttribute{
				Optional:      true,
				Description:   "EVM address that controls the ProxyAdmin for the ValidatorManager TransparentProxy (--proxy-contract-owner). Defaults to the ewoq dev address.",
				PlanModifiers: replace(),
			},

			// ── PoS ERC20 ────────────────────────────────────────────────────────────────

			"erc20_token_address": schema.StringAttribute{
				Optional:      true,
				Description:   "Address of an existing ERC20 staking token (--erc20-token-address). Only used when consensus = \"pos-erc20\".",
				PlanModifiers: replace(),
			},
			"erc20_token_supply": schema.Int64Attribute{
				Optional:      true,
				Description:   "Initial supply of a new ERC20 staking token (--erc20-token-supply). Only used when consensus = \"pos-erc20\" and no existing token address is given.",
				PlanModifiers: replaceI(),
			},
			"erc20_token_symbol": schema.StringAttribute{
				Optional:      true,
				Description:   "Symbol for a new ERC20 staking token (--erc20-token-symbol). Only used when consensus = \"pos-erc20\".",
				PlanModifiers: replace(),
			},
			"reward_basis_points": schema.Int64Attribute{
				Optional:      true,
				Description:   "PoS reward basis points for the reward calculator (--reward-basis-points). Default 100.",
				PlanModifiers: replaceI(),
			},

			// ── Custom genesis ───────────────────────────────────────────────────────────

			"genesis_file": schema.StringAttribute{
				Optional:      true,
				Description:   "Path to a custom genesis JSON file (--genesis). When set, automatic TxAllowList and Warp precompile injection is skipped — the caller is responsible for the full genesis config.",
				PlanModifiers: replace(),
			},

			// ── Deploy target ────────────────────────────────────────────────────────────

			"network": schema.StringAttribute{
				Optional:      true,
				Description:   "Deploy target network. One of: \"local\" (default), \"devnet\", \"testnet\", \"mainnet\".",
				PlanModifiers: replace(),
			},
			"endpoint": schema.StringAttribute{
				Optional:      true,
				Description:   "Custom network endpoint URL (--endpoint). Used with devnet or private clusters.",
				PlanModifiers: replace(),
			},
			"cluster": schema.StringAttribute{
				Optional:      true,
				Description:   "Named cluster to deploy into (--cluster).",
				PlanModifiers: replace(),
			},
			"avalanchego_version": schema.StringAttribute{
				Optional:      true,
				Description:   "Pin a specific AvalancheGo version for local/devnet nodes (--avalanchego-version, e.g. \"v1.12.0\").",
				PlanModifiers: replace(),
			},

			// ── Bootstrap validators ─────────────────────────────────────────────────────

			"num_nodes": schema.Int64Attribute{
				Optional:      true,
				Description:   "Number of nodes to create on a local network (--num-nodes). Default determined by --test-defaults or --production-defaults.",
				PlanModifiers: replaceI(),
			},
			"num_bootstrap_validators": schema.Int64Attribute{
				Optional:      true,
				Description:   "Number of bootstrap validators for the sovereign L1 (--num-bootstrap-validators).",
				PlanModifiers: replaceI(),
			},
			"bootstrap_validator_balance": schema.Float64Attribute{
				Optional:      true,
				Description:   "AVAX balance per bootstrap validator for continuous P-Chain fees (--balance). Default determined by CLI.",
				PlanModifiers: replaceF(),
			},
			"bootstrap_validator_weight": schema.Int64Attribute{
				Optional:      true,
				Description:   "Stake weight for each bootstrap validator (--weight).",
				PlanModifiers: replaceI(),
			},
			"bootstrap_filepath": schema.StringAttribute{
				Optional:      true,
				Description:   "JSON file providing pre-configured bootstrap validator details (--bootstrap-filepath).",
				PlanModifiers: replace(),
			},
			"change_owner_address": schema.StringAttribute{
				Optional:      true,
				Description:   "P-Chain address that receives change when a bootstrap validator exits (--change-owner-address).",
				PlanModifiers: replace(),
			},

			// ── Computed outputs ──────────────────────────────────────────────────────────

			"rpc_url": schema.StringAttribute{
				Computed:      true,
				Description:   "HTTP RPC endpoint for the deployed L1.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"subnet_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"blockchain_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"deployer_key": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Funded dev account private key for the local devnet. Do not use in production.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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

	if !r.networkExists(name) {
		if err := r.createL1(ctx, &plan, name, chainID); err != nil {
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
	// All attributes force replacement; Update is never called.
}

// Delete is state-only. avalanche network clean is a global operation that would
// destroy all local networks on the machine. demo/reset.sh owns actual teardown.
func (r *l1Resource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *l1Resource) networkExists(name string) bool {
	path := filepath.Join(r.cfg.DataDir, name, "network.json")
	_, err := os.Stat(path)
	return err == nil
}

func (r *l1Resource) createL1(ctx context.Context, plan *l1ResourceModel, name string, chainID int64) error {
	const ewoqAddr = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"

	tokenSymbol := stringOr(plan.TokenSymbol, "CLAW")
	consensus := stringOr(plan.Consensus, "poa")
	validatorOwner := stringOr(plan.ValidatorManagerOwner, ewoqAddr)
	proxyOwner := stringOr(plan.ProxyContractOwner, ewoqAddr)

	// ── Step 1: avalanche blockchain create ──────────────────────────────────────

	createArgs := []string{"blockchain", "create", name, "--evm", "--force",
		"--evm-chain-id", fmt.Sprintf("%d", chainID),
		"--evm-token", tokenSymbol,
	}

	if plan.ProductionDefaults.ValueBool() {
		createArgs = append(createArgs, "--production-defaults")
	} else {
		createArgs = append(createArgs, "--test-defaults")
	}

	if !plan.VMVersion.IsNull() && !plan.VMVersion.IsUnknown() {
		createArgs = append(createArgs, "--vm-version", plan.VMVersion.ValueString())
	}

	switch consensus {
	case "pos-native":
		createArgs = append(createArgs, "--proof-of-stake")
	case "pos-erc20":
		createArgs = append(createArgs, "--proof-of-stake-erc20")
	default: // "poa"
		createArgs = append(createArgs, "--proof-of-authority")
	}

	createArgs = append(createArgs,
		"--validator-manager-owner", validatorOwner,
		"--proxy-contract-owner", proxyOwner,
	)

	if !plan.Sovereign.IsNull() && !plan.Sovereign.IsUnknown() && !plan.Sovereign.ValueBool() {
		createArgs = append(createArgs, "--sovereign=false")
	}

	// Warp / ICM: --icm implies --warp.
	if plan.EnableICM.ValueBool() {
		createArgs = append(createArgs, "--icm")
	} else if plan.EnableWarp.ValueBool() {
		createArgs = append(createArgs, "--warp")
	}

	if !plan.EnableDebug.IsNull() && !plan.EnableDebug.IsUnknown() && !plan.EnableDebug.ValueBool() {
		createArgs = append(createArgs, "--debug=false")
	}

	if !plan.GenesisFile.IsNull() && !plan.GenesisFile.IsUnknown() {
		createArgs = append(createArgs, "--genesis", plan.GenesisFile.ValueString())
	}

	createOut, err := exec.CommandContext(ctx, "avalanche", createArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("avalanche blockchain create: %w\n%s", err, createOut)
	}

	// ── Step 1b: inject genesis precompiles ──────────────────────────────────────
	// Only when no custom genesis file is provided; the caller owns genesis in that case.
	if plan.GenesisFile.IsNull() || plan.GenesisFile.IsUnknown() {
		injectWarp := plan.EnableWarp.ValueBool() || plan.EnableICM.ValueBool()
		if err := injectGenesisPrecompiles(name, ewoqAddr, injectWarp); err != nil {
			return fmt.Errorf("inject genesis precompiles: %w", err)
		}
	}

	// ── Step 2: avalanche blockchain deploy ──────────────────────────────────────

	deployCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	deployArgs := []string{"blockchain", "deploy", name}
	deployArgs = append(deployArgs, networkFlag(plan))

	if !plan.AvalancheGoVersion.IsNull() && !plan.AvalancheGoVersion.IsUnknown() {
		deployArgs = append(deployArgs, "--avalanchego-version", plan.AvalancheGoVersion.ValueString())
	}
	if !plan.Endpoint.IsNull() && !plan.Endpoint.IsUnknown() {
		deployArgs = append(deployArgs, "--endpoint", plan.Endpoint.ValueString())
	}
	if !plan.Cluster.IsNull() && !plan.Cluster.IsUnknown() {
		deployArgs = append(deployArgs, "--cluster", plan.Cluster.ValueString())
	}
	if !plan.NumNodes.IsNull() && !plan.NumNodes.IsUnknown() && plan.NumNodes.ValueInt64() > 0 {
		deployArgs = append(deployArgs, "--num-nodes", fmt.Sprintf("%d", plan.NumNodes.ValueInt64()))
	}
	if !plan.NumBootstrapValidators.IsNull() && !plan.NumBootstrapValidators.IsUnknown() && plan.NumBootstrapValidators.ValueInt64() > 0 {
		deployArgs = append(deployArgs, "--num-bootstrap-validators", fmt.Sprintf("%d", plan.NumBootstrapValidators.ValueInt64()))
	}
	if !plan.BootstrapValidatorBalance.IsNull() && !plan.BootstrapValidatorBalance.IsUnknown() && plan.BootstrapValidatorBalance.ValueFloat64() > 0 {
		deployArgs = append(deployArgs, "--balance", strconv.FormatFloat(plan.BootstrapValidatorBalance.ValueFloat64(), 'f', -1, 64))
	}
	if !plan.BootstrapValidatorWeight.IsNull() && !plan.BootstrapValidatorWeight.IsUnknown() && plan.BootstrapValidatorWeight.ValueInt64() > 0 {
		deployArgs = append(deployArgs, "--weight", fmt.Sprintf("%d", plan.BootstrapValidatorWeight.ValueInt64()))
	}
	if !plan.BootstrapFilepath.IsNull() && !plan.BootstrapFilepath.IsUnknown() {
		deployArgs = append(deployArgs, "--bootstrap-filepath", plan.BootstrapFilepath.ValueString())
	}
	if !plan.ChangeOwnerAddress.IsNull() && !plan.ChangeOwnerAddress.IsUnknown() {
		deployArgs = append(deployArgs, "--change-owner-address", plan.ChangeOwnerAddress.ValueString())
	}

	// PoS ERC20 staking token.
	if consensus == "pos-erc20" {
		if !plan.ERC20TokenAddress.IsNull() && !plan.ERC20TokenAddress.IsUnknown() {
			deployArgs = append(deployArgs, "--erc20-token-address", plan.ERC20TokenAddress.ValueString())
		}
		if !plan.ERC20TokenSupply.IsNull() && !plan.ERC20TokenSupply.IsUnknown() && plan.ERC20TokenSupply.ValueInt64() > 0 {
			deployArgs = append(deployArgs, "--erc20-token-supply", fmt.Sprintf("%d", plan.ERC20TokenSupply.ValueInt64()))
		}
		if !plan.ERC20TokenSymbol.IsNull() && !plan.ERC20TokenSymbol.IsUnknown() {
			deployArgs = append(deployArgs, "--erc20-token-symbol", plan.ERC20TokenSymbol.ValueString())
		}
		if !plan.RewardBasisPoints.IsNull() && !plan.RewardBasisPoints.IsUnknown() && plan.RewardBasisPoints.ValueInt64() > 0 {
			deployArgs = append(deployArgs, "--reward-basis-points", fmt.Sprintf("%d", plan.RewardBasisPoints.ValueInt64()))
		}
	}

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

// networkFlag returns the --local / --devnet / --testnet / --mainnet / --cluster / --endpoint flag.
func networkFlag(plan *l1ResourceModel) string {
	if !plan.Cluster.IsNull() && !plan.Cluster.IsUnknown() {
		return "--cluster=" + plan.Cluster.ValueString()
	}
	switch plan.Network.ValueString() {
	case "devnet":
		return "--devnet"
	case "testnet", "fuji":
		return "--testnet"
	case "mainnet":
		return "--mainnet"
	default:
		return "--local"
	}
}

// stringOr returns the string value of t if set, otherwise fallback.
func stringOr(t types.String, fallback string) string {
	if !t.IsNull() && !t.IsUnknown() && t.ValueString() != "" {
		return t.ValueString()
	}
	return fallback
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

// injectGenesisPrecompiles patches the genesis.json written by avalanche-cli.
// It always adds TxAllowList with adminAddr as the sole admin.
// When injectWarp is true, it also adds warpConfig (required for ICM / ICTT).
// Must be called after `avalanche blockchain create` and before `avalanche blockchain deploy`.
func injectGenesisPrecompiles(name, adminAddr string, injectWarp bool) error {
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

	if injectWarp {
		// quorumNumerator 0 means use the subnet-evm default (67 = 2/3 of stake).
		cfg["warpConfig"] = map[string]interface{}{
			"blockTimestamp":  0,
			"quorumNumerator": 0,
		}
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

// injectTxAllowList is preserved for backward compatibility with existing tests.
func injectTxAllowList(name, adminAddr string) error {
	return injectGenesisPrecompiles(name, adminAddr, false)
}
