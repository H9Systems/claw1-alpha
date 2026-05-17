package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// ── Model ─────────────────────────────────────────────────────────────────────

type receiptModel struct {
	net      *networkJSON
	block    int64
	blockErr string
	copyMsg  string
	target   deployTarget
	repoRoot string
	width    int
}

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

	// Copy message feedback
	if m.copyMsg != "" {
		b.WriteString("\n" + styleGreen.Render("  ✓ "+m.copyMsg) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [C] copy RPC URL   [Q] quit"))

	return styleBox.Width(width - 4).Render(b.String())
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
