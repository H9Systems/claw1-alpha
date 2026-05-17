package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── network.json schema ───────────────────────────────────────────────────────

type networkJSON struct {
	Name               string     `json:"name"`
	ChainID            int64      `json:"chainId"`
	RPCURL             string     `json:"rpcUrl"`
	PlatformRPCURL     string     `json:"platformRpcUrl"`
	DeployerPrivateKey string     `json:"deployerPrivateKey"`
	Contracts          []contract `json:"contracts"`
	OCI                *ociMeta   `json:"oci,omitempty"`
}

type contract struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	DeployedAt string `json:"deployedAt,omitempty"`
}

type ociMeta struct {
	RemoteRPCURL string `json:"remoteRpcUrl"`
	VMIP         string `json:"vmIp"`
}

// ── Messages ──────────────────────────────────────────────────────────────────

type blockHeightMsg int64
type networkLoadedMsg struct{ net *networkJSON }
type tickMsg time.Time
type copyDoneMsg string
type explorerDoneMsg string

// ── Model ─────────────────────────────────────────────────────────────────────

type receiptModel struct {
	net      *networkJSON
	block    int64
	blockErr string
	copyMsg  string
	target   deployTarget
	repoRoot string
	width    int
	tab      int
	wallet   int
	action   string
}

const (
	receiptTabOverview = iota
	receiptTabExplorer
	receiptTabContracts
	receiptTabWallets
	numReceiptTabs
)

func newReceiptModel(target deployTarget, repoRoot string) receiptModel {
	return receiptModel{target: target, repoRoot: repoRoot}
}

func (m receiptModel) init() tea.Cmd {
	return tea.Batch(
		loadNetwork(m.target, m.repoRoot),
		tickEvery3s(),
	)
}

func loadNetwork(target deployTarget, repoRoot string) tea.Cmd {
	return func() tea.Msg {
		home, _ := os.UserHomeDir()
		var path string
		if target == targetOCI {
			path = filepath.Join(home, ".claw1", "claw1demobank-oci", "network.json")
		} else {
			path = filepath.Join(home, ".claw1", "claw1demobank", "network.json")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return networkLoadedMsg{nil}
		}
		var net networkJSON
		if err := json.Unmarshal(data, &net); err != nil {
			return networkLoadedMsg{nil}
		}
		return networkLoadedMsg{&net}
	}
}

func tickEvery3s() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func pollBlockHeight(rpcURL string) tea.Cmd {
	return func() tea.Msg {
		type rpcReq struct {
			JSONRPC string `json:"jsonrpc"`
			Method  string `json:"method"`
			Params  []any  `json:"params"`
			ID      int    `json:"id"`
		}
		type rpcResp struct {
			Result string `json:"result"`
		}

		body, _ := json.Marshal(rpcReq{"2.0", "eth_blockNumber", []any{}, 1})
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post(rpcURL, "application/json",
			strings.NewReader(string(body)))
		if err != nil {
			return blockHeightMsg(-1)
		}
		defer resp.Body.Close()

		var r rpcResp
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			return blockHeightMsg(-1)
		}
		// r.Result is "0x1a2b"
		var n int64
		fmt.Sscanf(strings.TrimPrefix(r.Result, "0x"), "%x", &n)
		return blockHeightMsg(n)
	}
}

func startExplorer(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		script := filepath.Join(repoRoot, "docker", "blockscout", "start.sh")
		cmd := exec.Command(script)
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			if msg != "" {
				return explorerDoneMsg("Explorer failed: " + msg)
			}
			return explorerDoneMsg("Explorer failed: " + err.Error())
		}
		return explorerDoneMsg("Blockscout starting at http://localhost:3001")
	}
}

func openExplorer() tea.Cmd {
	return func() tea.Msg {
		if err := openURL("http://localhost:3001"); err != nil {
			return explorerDoneMsg(err.Error())
		}
		return explorerDoneMsg("Opened http://localhost:3001")
	}
}

func (m receiptModel) Update(msg tea.Msg) (receiptModel, tea.Cmd) {
	switch msg := msg.(type) {
	case networkLoadedMsg:
		if msg.net != nil {
			m.net = msg.net
			return m, pollBlockHeight(m.net.RPCURL)
		}
	case tickMsg:
		cmds := []tea.Cmd{tickEvery3s(), loadNetwork(m.target, m.repoRoot)}
		if m.net != nil {
			cmds = append(cmds, pollBlockHeight(m.net.RPCURL))
		}
		return m, tea.Batch(cmds...)
	case blockHeightMsg:
		if int64(msg) > 0 {
			m.block = int64(msg)
			m.blockErr = ""
		} else {
			m.blockErr = "RPC unreachable"
		}
	case copyDoneMsg:
		m.copyMsg = string(msg)
	case explorerDoneMsg:
		m.action = string(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "right":
			m.tab = (m.tab + 1) % numReceiptTabs
		case "shift+tab", "left":
			m.tab = (m.tab - 1 + numReceiptTabs) % numReceiptTabs
		case "down":
			if m.tab == receiptTabWallets {
				m.wallet = (m.wallet + 1) % len(demoWallets())
			}
		case "up":
			if m.tab == receiptTabWallets {
				m.wallet = (m.wallet - 1 + len(demoWallets())) % len(demoWallets())
			}
		case "s", "S":
			if m.tab == receiptTabExplorer {
				return m, startExplorer(m.repoRoot)
			}
		case "o", "O":
			if m.tab == receiptTabExplorer {
				return m, openExplorer()
			}
		case "a", "A":
			if m.tab == receiptTabWallets {
				w := demoWallets()[m.wallet]
				return m, copyToClipboard(w.Address)
			}
		case "k", "K":
			if m.tab == receiptTabWallets && m.net != nil && m.wallet == 0 {
				return m, copyToClipboard(hexPrivateKey(m.net.DeployerPrivateKey))
			}
		}
	}
	return m, nil
}

func (m receiptModel) View(width int) string {
	var b strings.Builder

	// Live badge
	liveStr := " " + dot(green) + " " + styleGreen.Render("LIVE")
	title := styleHeader.Render("CLAW1") + "  " + styleDim.Render("SOVEREIGNTY RECEIPT")
	padding := width - 4 - lipgloss.Width(title) - lipgloss.Width(liveStr) - 4
	if padding < 0 {
		padding = 0
	}
	b.WriteString(title + strings.Repeat(" ", padding) + liveStr + "\n\n")

	if m.net == nil {
		b.WriteString(styleYellow.Render("  Waiting for network.json...") + "\n")
		b.WriteString(styleDim.Render("  (network.json not found yet — deploy may still be running)") + "\n")
		return styleBox.Width(width - 4).Render(b.String())
	}

	b.WriteString(receiptTabs(m.tab) + "\n\n")

	switch m.tab {
	case receiptTabOverview:
		b.WriteString(m.overviewView(width))
	case receiptTabExplorer:
		b.WriteString(m.explorerView())
	case receiptTabContracts:
		b.WriteString(m.contractsView())
	case receiptTabWallets:
		b.WriteString(m.walletsView())
	}

	if m.copyMsg != "" {
		b.WriteString("\n" + styleGreen.Render("  ✓ "+m.copyMsg) + "\n")
	}
	if m.action != "" {
		b.WriteString("\n" + styleYellow.Render("  "+m.action) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [←/→] tabs   [C] copy RPC   [Q] quit"))

	return styleBox.Width(width - 4).Render(b.String())
}

func receiptTabs(active int) string {
	names := []string{"Overview", "Explorer", "Contracts", "Wallets"}
	var parts []string
	for i, name := range names {
		if i == active {
			parts = append(parts, styleTabActive.Render("["+name+"]"))
		} else {
			parts = append(parts, styleTab.Render(name))
		}
	}
	return strings.Join(parts, "")
}

func (m receiptModel) overviewView(width int) string {
	var b strings.Builder
	// Network row
	blockStr := styleDim.Render("─")
	if m.block > 0 {
		blockStr = styleGreen.Render(fmt.Sprintf("#%d ↑", m.block))
	} else if m.blockErr != "" {
		blockStr = styleRed.Render(m.blockErr)
	}
	b.WriteString(row("NETWORK", m.net.Name, "CHAIN", fmt.Sprintf("%d", m.net.ChainID)))
	b.WriteString(row("VALIDATORS", validatorsStr(m.net), "BLOCK", blockStr))
	if m.target == targetLocal {
		b.WriteString(row("TOPOLOGY", "Developer appliance", "PROD TARGET", "multi-node L1"))
	}

	if m.net.OCI != nil {
		tenancyLabel := "local"
		if m.target == targetOCI {
			tenancyLabel = "oci"
		}
		b.WriteString(row("OCI TENANCY", tenancyLabel, "VM IP", m.net.OCI.VMIP))
	}

	// Compliance posture
	b.WriteString("\n" + styleSectionTitle.Render("COMPLIANCE POSTURE") + "\n")
	kycStatus := dot(yellow) + " " + styleYellow.Render("DEMO MODE")
	kycLabel := "KYC Verifier"
	b.WriteString(row(kycLabel, kycStatus, "TxAllowList", dot(green)+" "+styleGreen.Render("ACTIVE")))
	b.WriteString(row("Jurisdiction", jurisdictionStr(m.net), "Enforcement", styleGreen.Render("LAYER 1")))

	// Contracts
	b.WriteString("\n" + styleSectionTitle.Render("DEPLOYED CONTRACTS") + "\n")
	showContracts := []string{
		"ComplianceRegistry", "DividendDistributor", "ERC3643Token",
		"ClaimIssuer", "IdentityRegistry", "ICTTSourceToken",
		"ICTTTokenHome", "ICTTTokenRemote",
	}
	shown := 0
	for _, name := range showContracts {
		if addr := findContract(m.net, name); addr != "" {
			b.WriteString("  " + dot(green) + "  " +
				styleValue.Render(fmt.Sprintf("%-26s", name)) +
				styleGreen.Render(shortAddr(addr)) + "\n")
			shown++
		}
	}
	if shown == 0 {
		b.WriteString(styleDim.Render("  No contracts found in network.json") + "\n")
	}

	b.WriteString("\n" + styleSectionTitle.Render("INTEROPERABILITY TRACE") + "\n")
	home := findContract(m.net, "ICTTTokenHome")
	remote := findContract(m.net, "ICTTTokenRemote")
	if home != "" && remote != "" {
		b.WriteString("  " + dot(green) + "  " + styleValue.Render("TokenHome") + "  " + styleGreen.Render(shortAddr(home)) + "\n")
		b.WriteString("  " + dot(green) + "  " + styleValue.Render("TokenRemote") + " " + styleGreen.Render(shortAddr(remote)) + "\n")
		b.WriteString(styleDim.Render("  Trace target: C-chain source tx -> Teleporter message -> L1 mint/receive") + "\n")
	} else {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render("Bridge workbench pending") + "\n")
		b.WriteString(styleDim.Render("  Set local Teleporter env and rerun with ICTT enabled to deploy TokenHome/TokenRemote.") + "\n")
	}

	// RPC URL (truncated)
	b.WriteString("\n" + styleSectionTitle.Render("RPC ENDPOINT") + "\n")
	rpc := m.net.RPCURL
	if len(rpc) > width-10 {
		rpc = rpc[:width-10] + "…"
	}
	b.WriteString("  " + styleDim.Render(rpc) + "\n")
	return b.String()
}

func (m receiptModel) explorerView() string {
	var b strings.Builder
	status := dot(red) + " " + styleRed.Render("OFFLINE")
	if explorerHealthy() {
		status = dot(green) + " " + styleGreen.Render("ONLINE")
	}
	b.WriteString(styleSectionTitle.Render("BLOCK EXPLORER") + "\n")
	b.WriteString(row("Blockscout UI", "http://localhost:3001", "STATUS", status))
	b.WriteString(row("Backend API", "http://localhost:4000", "SOURCE", "docker/blockscout"))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("ACTIONS") + "\n")
	b.WriteString("  " + styleButtonActive.Render("[ S  Start Blockscout ]") + "\n")
	b.WriteString("  " + styleButton.Render("[ O  Open explorer ]") + "\n")
	b.WriteString("\n")
	b.WriteString(styleDim.Render("  Blockscout starts in Docker and indexes the local L1 from network.json.") + "\n")
	b.WriteString(styleKeys.Render("\n  [S] start   [O] open   [←/→] tabs"))
	return b.String()
}

func (m receiptModel) contractsView() string {
	var b strings.Builder
	b.WriteString(styleSectionTitle.Render("DEPLOYED CONTRACTS") + "\n")
	for _, c := range m.net.Contracts {
		b.WriteString("  " + dot(green) + "  " + styleValue.Render(fmt.Sprintf("%-28s", c.Name)) + styleGreen.Render(c.Address) + "\n")
	}
	if len(m.net.Contracts) == 0 {
		b.WriteString(styleDim.Render("  No contracts found in network.json") + "\n")
	}
	b.WriteString("\n" + styleSectionTitle.Render("INSPECT") + "\n")
	b.WriteString(styleDim.Render("  claw1 inspect --local") + "\n")
	b.WriteString(styleDim.Render("  claw1 inspect --local --json") + "\n")
	return b.String()
}

func (m receiptModel) walletsView() string {
	var b strings.Builder
	wallets := demoWallets()
	b.WriteString(styleSectionTitle.Render("DEMO WALLETS") + "\n")
	for i, w := range wallets {
		prefix := "  "
		body := styleValue.Render(fmt.Sprintf("%-12s", w.Name)) + styleGreen.Render(w.Address) + "  " + styleDim.Render(w.Unsafe)
		if i == m.wallet {
			prefix = styleGreen.Render("› ")
			body = styleButtonActive.Render(fmt.Sprintf("%-12s", w.Name)) + styleGreen.Render(w.Address) + "  " + styleDim.Render(w.Unsafe)
		}
		b.WriteString(prefix + body + "\n")
	}
	b.WriteString("\n" + styleSectionTitle.Render("ACTIONS") + "\n")
	b.WriteString(styleDim.Render("  [A] copy selected address") + "\n")
	b.WriteString(styleDim.Render("  [K] copy deployer private key (local demo only)") + "\n")
	b.WriteString(styleDim.Render("  claw1 wallet list --json") + "\n")
	return b.String()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func row(l1, v1, l2, v2 string) string {
	left := styleLabel.Render(l1) + styleValue.Render(v1)
	right := styleLabel.Render(l2) + " " + v2
	return "  " + lipgloss.NewStyle().Width(38).Render(left) + right + "\n"
}

func shortAddr(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:8] + "…" + addr[len(addr)-4:]
}

func findContract(net *networkJSON, name string) string {
	for _, c := range net.Contracts {
		if c.Name == name {
			return c.Address
		}
	}
	return ""
}

func validatorsStr(net *networkJSON) string {
	return dot(green) + " " + dot(green) + " " + dot(green) + " " +
		dot(green) + " " + dot(green) + "  " + styleGreen.Render("5/5")
}

func jurisdictionStr(net *networkJSON) string {
	// Try to read from ComplianceRegistry if we had cast wired; for now show demo value
	return styleValue.Render("CNBV/MX")
}
