package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	ociDeployedToRe = regexp.MustCompile(`Deployed to:\s+(0x[a-fA-F0-9]{40})`)
	ociRemotePortRe = regexp.MustCompile(`^http://127\.0\.0\.1:(\d+)/`)
	ociRPCPathRe    = regexp.MustCompile(`^http://127\.0\.0\.1:\d+/(ext/.+)$`)
)

// ── Messages ─────────────────────────────────────────────────────────────────

type logLineMsg string
type stepAdvanceMsg int
type deployDoneMsg struct{ err error }

// ── Deploy step ───────────────────────────────────────────────────────────────

type stepStatus int

const (
	stepWaiting stepStatus = iota
	stepRunning
	stepDone
	stepFailed
)

type deployStep struct {
	name    string
	status  stepStatus
	elapsed time.Duration
	started time.Time
}

// ── Model ─────────────────────────────────────────────────────────────────────

type deployModel struct {
	steps   []deployStep
	logs    []string
	maxLogs int
	logCh   chan string
	advCh   chan int
	errCh   chan error
	err     error
	cfg     deployConfig
	done    bool
}

func newDeployModel(cfg deployConfig) deployModel {
	steps := []deployStep{
		{name: "Write credentials"},
		{name: "Provision OCI infrastructure"},
		{name: "Bootstrap Avalanche L1"},
		{name: "Deploy compliance contracts"},
	}
	if cfg.target == targetOCI && cfg.enableICTT {
		steps = append(steps, deployStep{name: "Enable ICTT bridge"})
	}
	if cfg.target == targetLocal {
		steps = []deployStep{
			{name: "Build Terraform provider"},
			{name: "Deploy Avalanche L1"},
			{name: "Deploy compliance contracts"},
			{name: "Deploy ERC-3643 suite"},
		}
		if cfg.enableICTT {
			steps = append(steps, deployStep{name: "Run ICTT bridge workbench"})
		}
	}
	return deployModel{
		steps:   steps,
		maxLogs: 200,
		logCh:   make(chan string, 200),
		advCh:   make(chan int, 10),
		errCh:   make(chan error, 1),
		cfg:     cfg,
	}
}

// start kicks off the deploy goroutine and returns the first waitForLog cmd.
func (m *deployModel) start() tea.Cmd {
	go m.run()
	return waitForLog(m.logCh, m.advCh, m.errCh)
}

func waitForLog(logCh chan string, advCh chan int, errCh chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case err := <-errCh:
			return deployDoneMsg{err: err}
		case line, ok := <-logCh:
			if !ok {
				select {
				case err := <-errCh:
					return deployDoneMsg{err: err}
				default:
				}
				return deployDoneMsg{}
			}
			return logLineMsg(line)
		case step := <-advCh:
			return stepAdvanceMsg(step)
		}
	}
}

func (m deployModel) Update(msg tea.Msg) (deployModel, tea.Cmd) {
	switch msg := msg.(type) {
	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > m.maxLogs {
			m.logs = m.logs[len(m.logs)-m.maxLogs:]
		}
		return m, waitForLog(m.logCh, m.advCh, m.errCh)

	case stepAdvanceMsg:
		idx := int(msg)
		if idx < len(m.steps) {
			if idx > 0 {
				prev := idx - 1
				m.steps[prev].status = stepDone
				m.steps[prev].elapsed = time.Since(m.steps[prev].started)
			}
			m.steps[idx].status = stepRunning
			m.steps[idx].started = time.Now()
		}
		return m, waitForLog(m.logCh, m.advCh, m.errCh)

	case deployDoneMsg:
		m.done = true
		if msg.err != nil {
			m.err = msg.err
			for i := range m.steps {
				if m.steps[i].status == stepRunning {
					m.steps[i].status = stepFailed
				}
			}
		} else {
			for i := range m.steps {
				if m.steps[i].status == stepRunning {
					m.steps[i].status = stepDone
					m.steps[i].elapsed = time.Since(m.steps[i].started)
				}
			}
		}
	}
	return m, nil
}

func (m deployModel) View(width int) string {
	var b strings.Builder

	targetLabel := "PRODUCTION TARGET"
	topology := "OCI VM + Avalanche L1"
	if m.cfg.target == targetLocal {
		targetLabel = "DEVELOPER APPLIANCE"
		topology = "local Avalanche devnet + custom L1"
	}
	b.WriteString(styleBrand.Render("CLAW1") + "  " +
		styleHeader.Render("DEPLOY RUN") + "  " +
		statusPill(targetLabel, blue) + "\n")
	b.WriteString(styleDim.Render("  "+topology+"    compliance: ERC-3643 / T-REX    evidence: local") + "\n\n")

	// Steps
	b.WriteString(styleSectionTitle.Render("RUNBOOK") + "\n")
	for i, s := range m.steps {
		index := styleDim.Render(fmt.Sprintf("%02d", i+1))
		switch s.status {
		case stepWaiting:
			b.WriteString("  " + index + "  " + circle() + "  " + styleDim.Render(fmt.Sprintf("%-32s", s.name)) +
				styleDim.Render("waiting") + "\n")
		case stepRunning:
			elapsed := time.Since(s.started).Round(time.Second)
			b.WriteString("  " + index + "  " + dot(yellow) + "  " + styleYellow.Render(fmt.Sprintf("%-32s", s.name)) +
				styleYellow.Render(fmt.Sprintf("running %s", elapsed)) + "\n")
		case stepDone:
			b.WriteString("  " + index + "  " + dot(green) + "  " + styleGreen.Render(fmt.Sprintf("%-32s", s.name)) +
				styleDim.Render(fmt.Sprintf("done %s", s.elapsed.Round(time.Second))) + "\n")
		case stepFailed:
			b.WriteString("  " + index + "  " + dot(red) + "  " + styleRed.Render(fmt.Sprintf("%-32s", s.name)) +
				styleRed.Render("failed") + "\n")
		}
	}

	// Log panel
	b.WriteString("\n" + styleSectionTitle.Render("EVENT STREAM") + "\n")
	b.WriteString(rule(width-6) + "\n")
	logHeight := 12
	start := 0
	if len(m.logs) > logHeight {
		start = len(m.logs) - logHeight
	}
	for _, line := range m.logs[start:] {
		// Truncate long lines
		if len(line) > width-8 {
			line = line[:width-8]
		}
		b.WriteString(styleDim.Render("  "+line) + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + styleSectionTitle.Render("FAILURE") + "\n")
		b.WriteString("  " + styleRed.Render("Deploy stopped") + "\n")
		b.WriteString(styleDim.Render("  "+failureHint(m.err.Error())) + "\n")
		b.WriteString(styleDim.Render("  Raw error: "+oneLine(m.err.Error(), width-15)) + "\n")
	}

	if m.done && m.err == nil {
		b.WriteString("\n" + statusPill("DEPLOYED", green) + styleGreen.Render("  Press Enter for dashboard: explorer, contracts, wallets, evidence") + "\n")
	}

	if m.err != nil {
		b.WriteString(styleKeys.Render("\n  [Q/Esc] exit   Retry after resolving the failure above. Use scripts/reset.sh if Terraform state is stale."))
	} else {
		b.WriteString(styleKeys.Render("\n  [Q/Esc] exit"))
	}

	return styleBox.Width(width - 4).Render(b.String())
}

func failureHint(err string) string {
	lower := strings.ToLower(err)
	switch {
	case strings.Contains(lower, "error accessing local wallet") ||
		strings.Contains(lower, "private key or mnemonic"):
		return "Foundry did not receive a deployer key. Rebuild the provider, then retry deploy."
	case strings.Contains(lower, "connection refused"):
		return "The L1 RPC is not ready or the tunnel is down. Reset the demo and retry."
	case strings.Contains(lower, "network.json"):
		return "The deploy state file is missing or malformed. Run scripts/reset.sh, then deploy again."
	default:
		return "Check the last log lines above. The deploy failed closed before continuing."
	}
}

func oneLine(s string, limit int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if limit > 0 && len(s) > limit {
		return s[:limit-1] + "…"
	}
	return s
}

// ── Orchestration ─────────────────────────────────────────────────────────────

func (m *deployModel) run() {
	if m.cfg.target == targetOCI {
		m.runOCI()
	} else {
		m.runLocal()
	}
}

func (m *deployModel) runOCI() {
	// Step 0: write credentials
	m.advCh <- 0
	if err := writeOCIConfig(m.cfg); err != nil {
		m.logCh <- "[claw1] ERROR: " + err.Error()
		close(m.logCh)
		return
	}
	if err := writeTFVars(m.cfg); err != nil {
		m.logCh <- "[claw1] ERROR: " + err.Error()
		close(m.logCh)
		return
	}
	m.logCh <- "[claw1] Credentials written"

	// Step 1+2: terraform init + apply (provisions VM + bootstraps L1)
	m.advCh <- 1
	tfDir := filepath.Join(m.cfg.repoRoot, "terraform", "oci")
	if err := m.runCmd(tfDir, "terraform", "init", "-input=false"); err != nil {
		m.logCh <- "[terraform] init failed: " + err.Error()
		close(m.logCh)
		return
	}
	if err := m.runCmd(tfDir, "terraform", "apply", "-auto-approve", "-input=false"); err != nil {
		m.logCh <- "[terraform] apply failed: " + err.Error()
		close(m.logCh)
		return
	}
	m.advCh <- 2 // L1 done (bootstrap ran inside terraform apply)

	// Step 3: deploy contracts into the OCI L1 via SSH tunnel
	m.advCh <- 3
	tunnel, activeRPC, err := m.openOCITunnel(tfDir)
	if err != nil {
		m.logCh <- "[oci] tunnel: " + err.Error()
		close(m.logCh)
		return
	}
	defer tunnel.Process.Kill()

	if err := m.deployOCIContracts(activeRPC); err != nil {
		m.logCh <- "[oci] contracts: " + err.Error()
		close(m.logCh)
		return
	}

	// Step 4 (optional): ICTT bridge
	if m.cfg.enableICTT {
		m.advCh <- 4
		if err := m.deployOCIICTT(activeRPC); err != nil {
			m.logCh <- "[ictt] " + err.Error()
			// non-fatal: log and continue
		}
	}

	close(m.logCh)
}

// openOCITunnel reads the OCI terraform outputs, kills any existing tunnel on
// port 54320, opens a new SSH tunnel, waits for the RPC to be reachable, and
// returns the tunnel process (caller must Kill()) and the active RPC URL.
func (m *deployModel) openOCITunnel(tfDir string) (*exec.Cmd, string, error) {
	home, _ := os.UserHomeDir()
	netPath := filepath.Join(home, ".claw1", "claw1demobank-oci", "network.json")

	data, err := os.ReadFile(netPath)
	if err != nil {
		return nil, "", fmt.Errorf("read network.json: %w (run terraform apply first)", err)
	}
	var net struct {
		RPCURL string `json:"rpcUrl"`
	}
	if err := json.Unmarshal(data, &net); err != nil {
		return nil, "", fmt.Errorf("parse network.json: %w", err)
	}

	mPort := ociRemotePortRe.FindStringSubmatch(net.RPCURL)
	if len(mPort) != 2 {
		return nil, "", fmt.Errorf("could not parse port from rpcUrl: %s", net.RPCURL)
	}
	mPath := ociRPCPathRe.FindStringSubmatch(net.RPCURL)
	if len(mPath) != 2 {
		return nil, "", fmt.Errorf("could not parse path from rpcUrl: %s", net.RPCURL)
	}
	remotePort := mPort[1]
	activeRPC := "http://127.0.0.1:54320/" + mPath[1]

	ociIP, err := m.tfOutput(tfDir, "oci_vm_ip")
	if err != nil {
		return nil, "", fmt.Errorf("terraform output oci_vm_ip: %w", err)
	}
	sshKey, err := m.tfOutput(tfDir, "ssh_private_key_path")
	if err != nil {
		sshKey = filepath.Join(home, ".ssh", "id_ed25519")
	}

	// Kill any stale tunnel
	exec.Command("pkill", "-f", "ssh.*54320").Run() //nolint:errcheck
	time.Sleep(500 * time.Millisecond)

	tunnel := exec.Command("ssh",
		"-N",
		"-o", "StrictHostKeyChecking=no",
		"-i", sshKey,
		"-L", fmt.Sprintf("54320:127.0.0.1:%s", remotePort),
		fmt.Sprintf("ubuntu@%s", ociIP),
	)
	if err := tunnel.Start(); err != nil {
		return nil, "", fmt.Errorf("ssh tunnel start: %w", err)
	}
	m.logCh <- fmt.Sprintf("[oci] tunnel: localhost:54320 -> %s:%s", ociIP, remotePort)

	// Wait up to 30s for the RPC to respond
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Post(activeRPC, "application/json",
			strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			m.logCh <- "[oci] RPC ready at " + activeRPC
			return tunnel, activeRPC, nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	tunnel.Process.Kill()
	return nil, "", fmt.Errorf("RPC at %s did not respond within 30s", activeRPC)
}

// deployOCIContracts deploys ComplianceRegistry, DividendDistributor, and the
// ERC-3643 suite onto the L1 reachable at activeRPC, then updates the local
// OCI network.json with contract addresses and tunnel metadata.
func (m *deployModel) deployOCIContracts(activeRPC string) error {
	home, _ := os.UserHomeDir()
	netPath := filepath.Join(home, ".claw1", "claw1demobank-oci", "network.json")
	contractsDir := filepath.Join(m.cfg.repoRoot, "contracts")
	tfDir := filepath.Join(m.cfg.repoRoot, "terraform", "oci")

	data, err := os.ReadFile(netPath)
	if err != nil {
		return fmt.Errorf("read network.json: %w", err)
	}
	var net ociNetworkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return fmt.Errorf("parse network.json: %w", err)
	}

	deployer := net.DeployerPrivateKey
	chainID := fmt.Sprintf("%d", net.ChainID)
	const ewoqAddr = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	keyEnv := "FOUNDRY_ETH_PRIVATE_KEY=" + deployer

	// ComplianceRegistry
	m.logCh <- "[forge] deploying ComplianceRegistry..."
	crOut, err := m.captureForge(contractsDir, []string{keyEnv},
		"create", "src/ComplianceRegistry.sol:ComplianceRegistry",
		"--root", contractsDir,
		"--rpc-url", activeRPC,
		"--broadcast",
		"--private-key", deployer,
		"--constructor-args", chainID, ewoqAddr,
		"0x0000000000000000000000000000000000000000", "0", "demo",
	)
	if err != nil {
		return fmt.Errorf("ComplianceRegistry: %w", err)
	}
	crAddr := parseDeployedTo(crOut)
	if crAddr == "" {
		return fmt.Errorf("could not parse ComplianceRegistry address")
	}
	m.logCh <- "[forge] ComplianceRegistry: " + crAddr

	// DividendDistributor
	m.logCh <- "[forge] deploying DividendDistributor..."
	ddOut, err := m.captureForge(contractsDir, []string{keyEnv},
		"create", "src/DividendDistributor.sol:DividendDistributor",
		"--root", contractsDir,
		"--rpc-url", activeRPC,
		"--broadcast",
		"--private-key", deployer,
		"--constructor-args", "0x0000000000000000000000000000000000000000", "0",
	)
	if err != nil {
		return fmt.Errorf("DividendDistributor: %w", err)
	}
	ddAddr := parseDeployedTo(ddOut)
	if ddAddr == "" {
		return fmt.Errorf("could not parse DividendDistributor address")
	}
	m.logCh <- "[forge] DividendDistributor: " + ddAddr

	// ERC-3643 suite
	m.logCh <- "[forge] deploying ERC-3643 (T-REX) suite..."
	erc3643Env := []string{
		"DEPLOYER_PRIVATE_KEY=" + hexPrivateKey(deployer),
		"DEMO_INVESTOR_ADDRESS=" + ewoqAddr,
	}
	_, err = m.captureForge(contractsDir, erc3643Env,
		"script", "script/DeployERC3643.s.sol:DeployERC3643",
		"--root", contractsDir,
		"--rpc-url", activeRPC,
		"--broadcast",
	)
	if err != nil {
		m.logCh <- "[forge] ERC-3643 warning: " + err.Error()
	}

	// Update local network.json
	ociIP, _ := m.tfOutput(tfDir, "oci_vm_ip")
	now := time.Now().UTC().Format(time.RFC3339)
	net.RPCURL = activeRPC
	if net.OCI == nil {
		net.OCI = &ociNetMeta{}
	}
	net.OCI.RemoteRPCURL = data2rpcURL(data)
	net.OCI.VMIP = ociIP
	net.Contracts = []ociContract{
		{Name: "ComplianceRegistry", Address: crAddr, DeployedAt: now},
		{Name: "DividendDistributor", Address: ddAddr, DeployedAt: now},
	}
	updated, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return err
	}
	tmp := netPath + ".tmp"
	if err := os.WriteFile(tmp, updated, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, netPath)
}

// deployOCIICTT runs the ICTT bridge deploy script if the lib is installed.
// For OCI deploys this uses the Fuji C-chain with its canonical Teleporter Registry.
func (m *deployModel) deployOCIICTT(activeRPC string) error {
	icttLib := filepath.Join(m.cfg.repoRoot, "contracts", "lib", "avalanche-interchain-token-transfer")
	if _, err := os.Stat(icttLib); err != nil {
		m.logCh <- "[ictt] lib not installed — run scripts/ictt-setup.sh first"
		return nil
	}

	home, _ := os.UserHomeDir()
	netPath := filepath.Join(home, ".claw1", "claw1demobank-oci", "network.json")
	data, err := os.ReadFile(netPath)
	if err != nil {
		return fmt.Errorf("read network.json: %w", err)
	}
	var net struct {
		DeployerPrivateKey string `json:"deployerPrivateKey"`
		BlockchainID       string `json:"blockchainId"`
	}
	if err := json.Unmarshal(data, &net); err != nil {
		return fmt.Errorf("parse network.json: %w", err)
	}

	// Fuji Teleporter Registry (canonical, v1.0.0+).
	// The deploy script auto-deploys L1 TeleporterRegistry if L1_TELEPORTER_REGISTRY is unset.
	const fujiTeleporterRegistry = "0xF86Cb19Ad8405AEFa7d09C778215D2Cb6eBfB228"
	const fujicChainRPC = "https://api.avax-test.network/ext/bc/C/rpc"
	// Fuji C-chain blockchainID (bytes32)
	const fujiCChainID = "0x7fc93d85c6d62be589232824d4c06ca2f89b8800dc83c98a804fcddabb3ae2d5"

	contractsDir := filepath.Join(m.cfg.repoRoot, "contracts")
	env := []string{
		"DEPLOYER_PRIVATE_KEY=" + net.DeployerPrivateKey,
		"C_CHAIN_RPC_URL=" + fujicChainRPC,
		"C_CHAIN_BLOCKCHAIN_ID=" + fujiCChainID,
		"L1_RPC_URL=" + activeRPC,
		// OCI: use Fuji's canonical Teleporter Registry for C-chain.
		// L1 side auto-deploys a fresh registry via the script.
		"C_CHAIN_TELEPORTER_REGISTRY=" + fujiTeleporterRegistry,
	}
	m.logCh <- "[ictt] deploying TokenHome (Fuji C-chain) + TokenRemote (OCI L1)..."
	m.logCh <- "[ictt] C-chain: Fuji (TeleporterRegistry: " + fujiTeleporterRegistry + ")"
	m.logCh <- "[ictt] L1: OCI (TeleporterRegistry: auto-deploy)"
	out, err := m.captureForge(contractsDir, env,
		"script", "script/DeployICTT.s.sol:DeployICTT",
		"--root", contractsDir,
		"--multi", // broadcast on multiple forks
		"--broadcast",
	)
	if err != nil {
		return fmt.Errorf("forge script: %w", err)
	}
	m.logCh <- "[ictt] " + strings.TrimSpace(out)
	return nil
}

// captureForge runs a forge command, streams output line-by-line to logCh,
// and returns the combined output as a string.
func (m *deployModel) captureForge(dir string, extraEnv []string, args ...string) (string, error) {
	cmd := exec.Command("forge", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"HOME="+os.Getenv("HOME"),
		"PATH="+os.Getenv("PATH")+":/snap/bin:/usr/local/bin",
	)
	cmd.Env = append(cmd.Env, extraEnv...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	var mu sync.Mutex
	var buf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	scanPipe := func(r interface {
		Scan() bool
		Text() string
	}) {
		defer wg.Done()
		for r.Scan() {
			line := r.Text()
			m.logCh <- line
			mu.Lock()
			buf.WriteString(line + "\n")
			mu.Unlock()
		}
	}
	go scanPipe(bufio.NewScanner(stdout))
	go scanPipe(bufio.NewScanner(stderr))
	wg.Wait()

	err = cmd.Wait()
	return buf.String(), err
}

// tfOutput runs `terraform -chdir=<dir> output -raw <key>`.
func (m *deployModel) tfOutput(dir, key string) (string, error) {
	out, err := exec.Command("terraform", "-chdir="+dir, "output", "-raw", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// parseDeployedTo extracts the contract address from forge create output.
func parseDeployedTo(output string) string {
	m := ociDeployedToRe.FindStringSubmatch(output)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// data2rpcURL extracts the rpcUrl field from a raw network.json byte slice.
func data2rpcURL(data []byte) string {
	var v struct {
		RPCURL string `json:"rpcUrl"`
	}
	json.Unmarshal(data, &v) //nolint:errcheck
	return v.RPCURL
}

// ociNetworkJSON is the shape of ~/.claw1/<name>/network.json for OCI deploys.
type ociNetworkJSON struct {
	Name               string        `json:"name"`
	ChainID            int64         `json:"chainId"`
	SubnetID           string        `json:"subnetId,omitempty"`
	BlockchainID       string        `json:"blockchainId,omitempty"`
	RPCURL             string        `json:"rpcUrl"`
	PlatformRPCURL     string        `json:"platformRpcUrl,omitempty"`
	DeployerPrivateKey string        `json:"deployerPrivateKey"`
	Contracts          []ociContract `json:"contracts"`
	OCI                *ociNetMeta   `json:"oci,omitempty"`
}

type ociContract struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	DeployedAt string `json:"deployedAt,omitempty"`
}

type ociNetMeta struct {
	RemoteRPCURL string `json:"remoteRpcUrl,omitempty"`
	VMIP         string `json:"vmIp,omitempty"`
}

func (m *deployModel) runLocal() {
	m.advCh <- 0
	providerDir := filepath.Join(m.cfg.repoRoot, "terraform", "providers", "terraform-provider-claw1")
	if err := m.runCmd(providerDir, "make", "install"); err != nil {
		m.logCh <- "[make] install failed: " + err.Error()
		m.errCh <- err
		close(m.logCh)
		return
	}

	m.advCh <- 1
	tfDir := filepath.Join(m.cfg.repoRoot, "terraform")
	// Remove stale lock file so init regenerates it
	os.Remove(filepath.Join(tfDir, ".terraform.lock.hcl"))
	if err := m.runCmd(tfDir, "terraform", "init", "-upgrade", "-input=false"); err != nil {
		m.logCh <- "[terraform] init failed: " + err.Error()
		m.errCh <- err
		close(m.logCh)
		return
	}

	m.advCh <- 2
	if err := m.runCmd(tfDir, "terraform", "apply", "-auto-approve", "-input=false"); err != nil {
		m.logCh <- "[terraform] apply failed: " + err.Error()
		m.errCh <- err
		close(m.logCh)
		return
	}

	m.advCh <- 3
	if err := m.deployERC3643Local(); err != nil {
		m.logCh <- "[erc3643] deploy failed: " + err.Error()
		m.errCh <- err
		close(m.logCh)
		return
	}

	if m.cfg.enableICTT {
		m.advCh <- 4
		if err := m.deployLocalICTT(); err != nil {
			m.logCh <- "[ictt] bridge workbench stopped: " + err.Error()
			m.logCh <- "[ictt] non-fatal for developer appliance: ERC-3643 L1 remains usable"
		}
	}

	close(m.logCh)
}

// deployERC3643Local reads network.json to get the RPC + deployer key, then
// runs forge script to deploy the T-REX suite.
func (m *deployModel) deployERC3643Local() error {
	home, _ := os.UserHomeDir()
	netPath := filepath.Join(home, ".claw1", "claw1demobank", "network.json")
	data, err := os.ReadFile(netPath)
	if err != nil {
		return fmt.Errorf("read network.json: %w", err)
	}
	var net ociNetworkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return fmt.Errorf("parse network.json: %w", err)
	}
	if net.RPCURL == "" || net.DeployerPrivateKey == "" {
		return fmt.Errorf("network.json missing rpcUrl or deployerPrivateKey")
	}

	contractsDir := filepath.Join(m.cfg.repoRoot, "contracts")
	const ewoqAddr = "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	m.logCh <- "[forge] deploying ERC-3643 (T-REX) suite..."
	out, err := m.captureForge(contractsDir, []string{
		"DEPLOYER_PRIVATE_KEY=" + hexPrivateKey(net.DeployerPrivateKey),
		"DEMO_INVESTOR_ADDRESS=" + ewoqAddr,
	},
		"script", "script/DeployERC3643.s.sol:DeployERC3643",
		"--root", contractsDir,
		"--rpc-url", net.RPCURL,
		"--broadcast",
	)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	net.Contracts = upsertContract(net.Contracts, "ERC3643Token", parseLabelAddress(out, "Token (CEQ):"), now)
	net.Contracts = upsertContract(net.Contracts, "IdentityRegistry", parseLabelAddress(out, "IdentityRegistry:"), now)
	net.Contracts = upsertContract(net.Contracts, "ClaimIssuer", parseLabelAddress(out, "ClaimIssuer (KYC auth):"), now)
	return writeNetworkFile(netPath, &net)
}

func hexPrivateKey(key string) string {
	key = strings.TrimSpace(key)
	if strings.HasPrefix(key, "0x") || strings.HasPrefix(key, "0X") {
		return key
	}
	return "0x" + key
}

// cChainBlockchainID queries the local Avalanche primary C-chain for its blockchain ID.
func cChainBlockchainID(rpcURL string) (string, error) {
	// Avalanche C-chain exposes the blockchain ID via the rpcEndpointPrivacyInfo or
	// via eth_chainId. The C-chain blockchainID for a local network changes each
	// restart, so we query it dynamically from the platform API.
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`)
	resp, err := http.Post(rpcURL, "application/json", body)
	if err != nil {
		return "", fmt.Errorf("query C-chain chainId: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode C-chain chainId: %w", err)
	}
	if out.Result == "" {
		return "", fmt.Errorf("empty chainId from C-chain")
	}
	// Avalanche C-chain blockchainID is NOT the same as chainId (which is 43114/43113/...).
	// We need to query the platform API: curl -X POST .../ext/bc/C/rpc AVAX method
	// But the simplest way for local devnets is to query the info API.
	return "", fmt.Errorf("C-chain blockchainID for local devnet cannot be auto-detected from eth_chainId; set C_CHAIN_BLOCKCHAIN_ID env var to the bytes32 hex value from avalanche network status")
}

func (m *deployModel) deployLocalICTT() error {
	icttLib := filepath.Join(m.cfg.repoRoot, "contracts", "lib", "avalanche-interchain-token-transfer")
	if _, err := os.Stat(icttLib); err != nil {
		return fmt.Errorf("ICTT lib not installed; run scripts/ictt-setup.sh")
	}

	home, _ := os.UserHomeDir()
	netPath := filepath.Join(home, ".claw1", "claw1demobank", "network.json")
	data, err := os.ReadFile(netPath)
	if err != nil {
		return fmt.Errorf("read network.json: %w", err)
	}
	var net ociNetworkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return fmt.Errorf("parse network.json: %w", err)
	}
	if net.RPCURL == "" || net.DeployerPrivateKey == "" {
		return fmt.Errorf("network.json missing rpcUrl or deployerPrivateKey")
	}

	// On-prem topology: local Avalanche primary C-chain  ←→  L1 subnet
	// The deploy script auto-deploys TeleporterRegistry on both chains if not provided,
	// so L1_TELEPORTER_REGISTRY and C_CHAIN_TELEPORTER_REGISTRY are optional.
	cChainRPC := os.Getenv("C_CHAIN_RPC_URL")
	if cChainRPC == "" {
		cChainRPC = "http://127.0.0.1:9650/ext/bc/C/rpc"
	}

	// C_CHAIN_BLOCKCHAIN_ID is required for ICTT bridge wiring.
	// For local devnets it changes every restart — query `avalanche network status`
	// or extract from platform API.
	cChainID := os.Getenv("C_CHAIN_BLOCKCHAIN_ID")
	if cChainID == "" {
		// Try to query the Platform API for the C-chain blockchainID.
		// Local Avalanche C-chains use: POST http://127.0.0.1:9650/ext/bc/C/rpc
		// with platform.getBlockchainID { alias: "C" }
		// Fall back to prompting.
		m.logCh <- "[ictt] WARNING: C_CHAIN_BLOCKCHAIN_ID not set"
		m.logCh <- "[ictt] For local devnets, run: avalanche network status"
		m.logCh <- "[ictt] Look for the C-chain blockchain ID and set C_CHAIN_BLOCKCHAIN_ID env var"
		return fmt.Errorf("set C_CHAIN_BLOCKCHAIN_ID to the local C-chain bytes32 hex (get from 'avalanche network status')")
	}

	// L1_TELEPORTER_REGISTRY and C_CHAIN_TELEPORTER_REGISTRY are optional:
	// the DeployICTT script auto-deploys TeleporterRegistry on each chain if unset.
	l1Registry := os.Getenv("L1_TELEPORTER_REGISTRY")
	cRegistry := os.Getenv("C_CHAIN_TELEPORTER_REGISTRY")

	env := []string{
		"DEPLOYER_PRIVATE_KEY=" + net.DeployerPrivateKey,
		"C_CHAIN_RPC_URL=" + cChainRPC,
		"C_CHAIN_BLOCKCHAIN_ID=" + cChainID,
		"L1_RPC_URL=" + net.RPCURL,
	}
	if l1Registry != "" {
		env = append(env, "L1_TELEPORTER_REGISTRY="+l1Registry)
	}
	if cRegistry != "" {
		env = append(env, "C_CHAIN_TELEPORTER_REGISTRY="+cRegistry)
	}

	contractsDir := filepath.Join(m.cfg.repoRoot, "contracts")
	m.logCh <- "[ictt] on-prem topology: local C-chain -> " + net.Name + " L1"
	m.logCh <- "[ictt] C-chain RPC: " + cChainRPC
	m.logCh <- "[ictt] C-chain ID: " + cChainID
	m.logCh <- "[ictt] L1 RPC: " + net.RPCURL
	if l1Registry == "" {
		m.logCh <- "[ictt] L1 TeleporterRegistry: auto-deploy"
	} else {
		m.logCh <- "[ictt] L1 TeleporterRegistry: " + l1Registry
	}
	if cRegistry == "" {
		m.logCh <- "[ictt] C-chain TeleporterRegistry: auto-deploy"
	} else {
		m.logCh <- "[ictt] C-chain TeleporterRegistry: " + cRegistry
	}

	out, err := m.captureForge(contractsDir, env,
		"script", "script/DeployICTT.s.sol:DeployICTT",
		"--root", contractsDir,
		"--multi",
		"--broadcast",
	)
	if err != nil {
		return fmt.Errorf("forge script: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	net.Contracts = upsertContract(net.Contracts, "ICTTTokenHome", parseLabelAddress(out, "ICTT_TOKEN_HOME:"), now)
	net.Contracts = upsertContract(net.Contracts, "ICTTSourceToken", parseLabelAddress(out, "ICTT_SOURCE_TOKEN:"), now)
	net.Contracts = upsertContract(net.Contracts, "ICTTTokenRemote", parseLabelAddress(out, "ICTT_TOKEN_REMOTE:"), now)
	net.Contracts = upsertContract(net.Contracts, "CChainTeleporterRegistry", parseLabelAddress(out, "C-chain TeleporterRegistry:"), now)
	net.Contracts = upsertContract(net.Contracts, "L1TeleporterRegistry", parseLabelAddress(out, "L1 TeleporterRegistry: "), now)
	return writeNetworkFile(netPath, &net)
}

func upsertContract(contracts []ociContract, name, address, deployedAt string) []ociContract {
	if address == "" {
		return contracts
	}
	for i := range contracts {
		if contracts[i].Name == name {
			contracts[i].Address = address
			contracts[i].DeployedAt = deployedAt
			return contracts
		}
	}
	return append(contracts, ociContract{Name: name, Address: address, DeployedAt: deployedAt})
}

func writeNetworkFile(path string, net *ociNetworkJSON) error {
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

func parseLabelAddress(output, label string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(label) + `\s*(0x[a-fA-F0-9]{40})`)
	m := re.FindStringSubmatch(output)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

// runCmd runs a command and streams its output line by line to logCh.
func (m *deployModel) runCmd(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	// Merge PATH so homebrew/snap binaries are found
	cmd.Env = append(os.Environ(),
		"HOME="+os.Getenv("HOME"),
		"PATH="+os.Getenv("PATH")+":/snap/bin:/usr/local/bin",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	scan := func(r interface {
		Scan() bool
		Text() string
	}) {
		defer wg.Done()
		for r.Scan() {
			m.logCh <- r.Text()
		}
	}
	go scan(bufio.NewScanner(stdout))
	go scan(bufio.NewScanner(stderr))
	wg.Wait()
	return cmd.Wait()
}

// ── Config writers ─────────────────────────────────────────────────────────────

const ociConfigTpl = `[DEFAULT]
user={{.User}}
fingerprint={{.Fingerprint}}
tenancy={{.Tenancy}}
region={{.Region}}
key_file={{.KeyPath}}
`

func writeOCIConfig(cfg deployConfig) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".oci")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "config"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	keyPath := cfg.keyPath
	if strings.HasPrefix(keyPath, "~/") {
		keyPath = filepath.Join(home, keyPath[2:])
	}

	return template.Must(template.New("").Parse(ociConfigTpl)).Execute(f, map[string]string{
		"User":        cfg.user,
		"Fingerprint": cfg.fingerprint,
		"Tenancy":     cfg.tenancy,
		"Region":      cfg.region,
		"KeyPath":     keyPath,
	})
}

const tfVarsTpl = `compartment_id      = "{{.Tenancy}}"
availability_domain = "{{.AvailabilityDomain}}"
region              = "{{.Region}}"
shape               = "{{.Shape}}"
shape_ocpus         = {{.Ocpus}}
shape_memory_gbs    = {{.Memory}}
`

func writeTFVars(cfg deployConfig) error {
	path := filepath.Join(cfg.repoRoot, "terraform", "oci", "terraform.tfvars")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Derive availability domain from region: region + "-AD-1" with OCI format
	// User can adjust later; this is a best-guess default
	adName := ociDefaultAD(cfg.region)

	return template.Must(template.New("").Parse(tfVarsTpl)).Execute(f, map[string]string{
		"Tenancy":            cfg.tenancy,
		"AvailabilityDomain": adName,
		"Region":             cfg.region,
		"Shape":              cfg.shape,
		"Ocpus":              cfg.ocpus,
		"Memory":             cfg.memoryGBs,
	})
}

// ociDefaultAD returns a best-guess availability domain string.
// OCI ADs are region-specific; user may need to correct this.
func ociDefaultAD(region string) string {
	defaults := map[string]string{
		"us-ashburn-1":   "aBCD:US-ASHBURN-AD-1",
		"us-phoenix-1":   "aBCD:US-PHOENIX-AD-1",
		"us-chicago-1":   "aBCD:US-CHICAGO-1-AD-1",
		"sa-bogota-1":    "aBCD:SA-BOGOTA-1-AD-1",
		"sa-saopaulo-1":  "aBCD:SA-SAOPAULO-1-AD-1",
		"sa-santiago-1":  "aBCD:SA-SANTIAGO-1-AD-1",
		"eu-frankfurt-1": "aBCD:EU-FRANKFURT-1-AD-1",
	}
	if ad, ok := defaults[region]; ok {
		return ad
	}
	return "aBCD:" + strings.ToUpper(strings.ReplaceAll(region, "-", "_")) + "-AD-1"
}
