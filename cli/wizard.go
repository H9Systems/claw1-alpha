package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type deployTarget int

const (
	targetOCI deployTarget = iota
	targetLocal
)

type deployConfig struct {
	target      deployTarget
	tenancy     string
	user        string
	fingerprint string
	keyPath     string
	region      string
	shape       string
	ocpus       string
	memoryGBs   string
	repoRoot    string
	enableICTT  bool // attempt ICTT TokenHome+TokenRemote deployment when prerequisites are available
}

type wizardModel struct {
	target       deployTarget
	activeTab    int
	deployCursor int
	itemCursor   int
	focus        int
	inputs       []textinput.Model
	err          string
	action       string
	repoRoot     string
	enableICTT   bool
}

const (
	tabOverview = iota
	tabTrex
	tabExplorer
	tabContracts
	tabDeploy
	tabCloud
	numTabs
)

const (
	deployCursorLocal = iota
	deployCursorCChain
	deployCursorOCI
	deployCursorICTT
	deployCursorStart
	deployCursorDashboard
	numDeployCursors
)

const (
	inTenancy = iota
	inUser
	inFingerprint
	inKeyPath
	inRegion
	inShape
	inOcpus
	inMemory
	numInputs
)

func newWizardModel(repoRoot string) wizardModel {
	inputs := make([]textinput.Model, numInputs)

	mk := func(placeholder, value string, width int, mask bool) textinput.Model {
		t := textinput.New()
		t.Placeholder = placeholder
		t.SetValue(value)
		t.Width = width
		if mask {
			t.EchoMode = textinput.EchoPassword
		}
		return t
	}

	inputs[inTenancy] = mk("ocid1.tenancy.oc1..XXXXXXXXXX", "", 60, false)
	inputs[inUser] = mk("ocid1.user.oc1..XXXXXXXXXX", "", 60, false)
	inputs[inFingerprint] = mk("xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx", "", 50, false)
	inputs[inKeyPath] = mk("~/.oci/oci_api_key.pem", "~/.oci/oci_api_key.pem", 40, false)
	inputs[inRegion] = mk("us-ashburn-1", "us-ashburn-1", 30, false)
	inputs[inShape] = mk("VM.Standard.A1.Flex", "VM.Standard.A1.Flex", 30, false)
	inputs[inOcpus] = mk("2", "2", 5, false)
	inputs[inMemory] = mk("8", "8", 5, false)

	return wizardModel{
		target:       targetLocal,
		deployCursor: deployCursorLocal,
		focus:        inTenancy,
		inputs:       inputs,
		repoRoot:     repoRoot,
	}
}

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % numTabs
			m.itemCursor = 0
			m.syncFocus()
		case "shift+tab", "left":
			m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
			m.itemCursor = 0
			m.syncFocus()
		case "down":
			switch {
			case m.activeTab == tabDeploy:
				m.deployCursor = (m.deployCursor + 1) % numDeployCursors
			case m.activeTab == tabContracts:
				m.itemCursor = clampCursor(m.itemCursor+1, len(loadNetworkSnapshot(m.target).contracts))
			case m.activeTab == tabTrex:
				m.itemCursor = clampCursor(m.itemCursor+1, len(trexRecipients()))
			case m.activeTab == tabCloud && m.target == targetOCI:
				m.focus = (m.focus + 1) % numInputs
			}
			m.syncFocus()
		case "up":
			switch {
			case m.activeTab == tabDeploy:
				m.deployCursor = (m.deployCursor - 1 + numDeployCursors) % numDeployCursors
			case m.activeTab == tabContracts:
				m.itemCursor = clampCursor(m.itemCursor-1, len(loadNetworkSnapshot(m.target).contracts))
			case m.activeTab == tabTrex:
				m.itemCursor = clampCursor(m.itemCursor-1, len(trexRecipients()))
			case m.activeTab == tabCloud && m.target == targetOCI:
				m.focus = (m.focus - 1 + numInputs) % numInputs
			}
			m.syncFocus()
		case "enter":
			switch m.activeTab {
			case tabDeploy:
				if m.deployCursor == deployCursorDashboard {
					m.action = "Dashboard opens after deployment. Run `claw1 receipt` to open it directly."
				}
				if m.deployCursor == deployCursorCChain {
					m.action = "C-Chain is shown as the production liquidity rail. Deployment is not implemented yet; use ICTT workbench when ready."
				}
			case tabExplorer:
				m.action = "Explorer refreshed from selected L1 RPC."
			case tabContracts:
				snap := loadNetworkSnapshot(m.target)
				if len(snap.contracts) > 0 {
					return m, copyToClipboard(snap.contracts[m.itemCursor].Address)
				}
			case tabTrex:
				recipient := selectedTrexRecipient(m.itemCursor)
				m.action = simulateTrexTransfer(m.target, recipient.Address, "1").Message
			}
		case "s", "S":
			if m.activeTab == tabExplorer {
				m.action = "Explorer refreshed from selected L1 RPC."
			}
		case "o", "O":
			if m.activeTab == tabExplorer {
				m.action = "Embedded explorer uses the selected L1 RPC. No external explorer is required."
			}
		case "a", "A":
			if m.activeTab == tabContracts {
				snap := loadNetworkSnapshot(m.target)
				if len(snap.contracts) > 0 {
					return m, copyToClipboard(snap.contracts[m.itemCursor].Address)
				}
			}
			if m.activeTab == tabTrex {
				recipient := selectedTrexRecipient(m.itemCursor)
				return m, copyToClipboard(recipient.Address)
			}
		case "k", "K":
			if m.activeTab == tabTrex {
				snap := loadNetworkSnapshot(m.target)
				if snap.net != nil {
					return m, copyToClipboard(hexPrivateKey(snap.net.DeployerPrivateKey))
				}
			}
		case "i", "I":
			if m.activeTab == tabDeploy {
				m.enableICTT = !m.enableICTT
			}
		case "r", "R":
			if m.activeTab == tabTrex {
				recipient := selectedTrexRecipient(m.itemCursor)
				m.action = simulateTrexTransfer(m.target, recipient.Address, "1").Message
			}
		case "t", "T":
			if m.activeTab == tabTrex {
				recipient := selectedTrexRecipient(m.itemCursor)
				sim := simulateTrexTransfer(m.target, recipient.Address, "1")
				if !sim.Approved {
					m.action = sim.Message
					break
				}
				tx, err := sendTrexTransfer(m.target, recipient.Address, "1")
				if err != nil {
					m.action = "Transfer failed: " + oneLine(err.Error(), 120)
				} else {
					m.action = "Sent 1 CEQ to " + shortAddr(recipient.Address) + " tx " + shortAddr(tx)
				}
			}
		}
	case copyDoneMsg:
		m.action = string(msg)
	case explorerDoneMsg:
		m.action = string(msg)
	}

	if m.activeTab == tabCloud && m.target == targetOCI {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *wizardModel) syncFocus() {
	for i := range m.inputs {
		if m.activeTab == tabCloud && m.target == targetOCI && i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m wizardModel) validate() (deployConfig, error) {
	if m.target == targetLocal {
		return deployConfig{target: targetLocal, repoRoot: m.repoRoot, enableICTT: m.enableICTT}, nil
	}

	tenancy := strings.TrimSpace(m.inputs[inTenancy].Value())
	user := strings.TrimSpace(m.inputs[inUser].Value())
	fingerprint := strings.TrimSpace(m.inputs[inFingerprint].Value())
	keyPath := strings.TrimSpace(m.inputs[inKeyPath].Value())

	if !strings.HasPrefix(tenancy, "ocid1.tenancy") {
		return deployConfig{}, fmt.Errorf("tenancy OCID must start with ocid1.tenancy")
	}
	if !strings.HasPrefix(user, "ocid1.user") {
		return deployConfig{}, fmt.Errorf("user OCID must start with ocid1.user")
	}
	if fingerprint == "" {
		return deployConfig{}, fmt.Errorf("fingerprint required")
	}

	expanded := keyPath
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		expanded = filepath.Join(home, keyPath[2:])
	}
	if _, err := os.Stat(expanded); err != nil {
		return deployConfig{}, fmt.Errorf("key file not found: %s", expanded)
	}

	return deployConfig{
		target:      targetOCI,
		tenancy:     tenancy,
		user:        user,
		fingerprint: fingerprint,
		keyPath:     keyPath,
		region:      strings.TrimSpace(m.inputs[inRegion].Value()),
		shape:       strings.TrimSpace(m.inputs[inShape].Value()),
		ocpus:       strings.TrimSpace(m.inputs[inOcpus].Value()),
		memoryGBs:   strings.TrimSpace(m.inputs[inMemory].Value()),
		repoRoot:    m.repoRoot,
		enableICTT:  m.enableICTT,
	}, nil
}

func (m *wizardModel) activate() bool {
	if m.activeTab != tabDeploy {
		return false
	}
	switch m.deployCursor {
	case deployCursorOCI:
		m.target = targetOCI
		m.syncFocus()
	case deployCursorLocal:
		m.target = targetLocal
		m.syncFocus()
	case deployCursorCChain:
		m.action = "C-Chain is planned as the public liquidity rail. It is visible here to match the production topology, not deployed yet."
	case deployCursorICTT:
		m.enableICTT = !m.enableICTT
	case deployCursorStart:
		return true
	}
	return false
}

func (m wizardModel) openDashboard() bool {
	return m.activeTab == tabDeploy && m.deployCursor == deployCursorDashboard
}

func (m wizardModel) View(width int) string {
	var b strings.Builder
	contentWidth := width - 8
	if contentWidth < 72 {
		contentWidth = 72
	}

	b.WriteString(styleBrand.Render("CLAW1") + "  " +
		styleHeader.Render("PRIVATE L1 DEV CONSOLE") + "  " +
		styleDim.Render("T-REX workbench for regulated Avalanche L1s") + "\n")
	b.WriteString(styleKicker.Render("  Build, transfer, inspect, and prove CEQ flows without leaving the terminal.") + "\n")
	b.WriteString(styleDim.Render("  Direct RPC explorer, T-REX wallet actions, contract map, deploy runbook, and cloud controls.") + "\n\n")
	b.WriteString(m.tabs() + "\n\n")

	switch m.activeTab {
	case tabOverview:
		b.WriteString(m.overviewTab(contentWidth))
	case tabTrex:
		b.WriteString(m.trexTab(contentWidth))
	case tabExplorer:
		b.WriteString(m.explorerTab(contentWidth))
	case tabContracts:
		b.WriteString(m.contractsTab(contentWidth))
	case tabDeploy:
		b.WriteString(m.deployTab(contentWidth))
	case tabCloud:
		b.WriteString(m.cloudTab(contentWidth))
	}

	if m.err != "" {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err) + "\n")
	}
	if m.action != "" {
		b.WriteString("\n" + styleYellow.Render("  "+m.action) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [←/→] workspace   [↑/↓] select   [Enter] primary   [T] send CEQ   [R] simulate   [A] copy   [Q] quit"))

	inner := b.String()
	return styleBox.Width(width - 4).Render(inner)
}

func (m wizardModel) tabs() string {
	names := []string{"Overview", "T-REX", "Explorer", "Contracts", "Deploy", "Cloud"}
	var parts []string
	for i, name := range names {
		if i == m.activeTab {
			parts = append(parts, styleTabActive.Render("["+name+"]"))
		} else {
			parts = append(parts, styleTab.Render(name))
		}
	}
	return strings.Join(parts, "")
}

func (m wizardModel) overviewTab(contentWidth int) string {
	var b strings.Builder
	snap := loadNetworkSnapshot(m.target)
	b.WriteString(styleSectionTitle.Render("OVERVIEW") + "\n")
	if snap.net == nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render("No L1 deployed. Go to Deploy, choose Developer appliance, then run Deploy / reconcile.") + "\n\n")
		b.WriteString(styleSectionTitle.Render("NEXT ACTIONS") + "\n")
		b.WriteString(featureRow("Deploy", "create local private L1 + T-REX contracts", contentWidth))
		b.WriteString(featureRow("Cloud", "configure OCI only when production target is needed", contentWidth))
		return b.String()
	}
	block := "unreachable"
	if val, err := rpcString(snap.net.RPCURL, "eth_blockNumber", []any{}); err == nil {
		block = hexBig(val).String()
	}
	token := trexTokenAddress(snap.net)
	tokenStatus := "missing"
	if token != "" {
		tokenStatus = "deployed " + shortAddr(token)
	}
	transfers, _ := trexTransferHistory(m.target, demoWallets()[0].Address, 1)
	lastTransfer := "none yet"
	if len(transfers) > 0 {
		lastTransfer = transfers[0].Amount + " CEQ at block " + transfers[0].Block + "  " + shortAddr(transfers[0].TxHash)
	}
	b.WriteString(featureRow("Network", snap.net.Name, contentWidth))
	b.WriteString(featureRow("Chain ID", fmt.Sprintf("%d", snap.net.ChainID), contentWidth))
	b.WriteString(featureRow("Latest block", block, contentWidth))
	b.WriteString(featureRow("RPC", snap.net.RPCURL, contentWidth))
	b.WriteString(featureRow("T-REX token", tokenStatus, contentWidth))
	b.WriteString(featureRow("C-Chain rail", cChainRailStatus(), contentWidth))
	b.WriteString(featureRow("Last CEQ transfer", lastTransfer, contentWidth))
	b.WriteString("\n" + styleSectionTitle.Render("WORKFLOW") + "\n")
	b.WriteString(featureRow("1. Deploy", "start or reconcile private L1 and contracts", contentWidth))
	b.WriteString(featureRow("2. T-REX", "simulate and send CEQ transfers", contentWidth))
	b.WriteString(featureRow("3. Explorer", "inspect blocks, tx hashes, and Transfer events", contentWidth))
	b.WriteString(featureRow("4. Contracts", "copy deployed addresses for CLI/API work", contentWidth))
	return b.String()
}

func (m wizardModel) deployTab(contentWidth int) string {
	var b strings.Builder
	snap := loadNetworkSnapshot(m.target)
	b.WriteString(styleSectionTitle.Render("DEPLOY RUNBOOK") + "\n")
	b.WriteString(m.optionRow(deployCursorLocal, m.target == targetLocal, "Developer appliance", "local private L1") + "\n")
	b.WriteString(m.optionRow(deployCursorCChain, false, "C-Chain liquidity rail", "planned public liquidity endpoint") + "\n")
	b.WriteString(m.optionRow(deployCursorOCI, m.target == targetOCI, "Production target", "OCI private L1") + "\n")
	b.WriteString(m.optionRow(deployCursorICTT, m.enableICTT, "ICTT bridge to C-Chain", "optional bridge workbench") + "\n")
	b.WriteString(m.optionRow(deployCursorStart, false, "Deploy / reconcile", "apply Terraform + contracts") + "\n")
	b.WriteString(m.optionRow(deployCursorDashboard, false, "Open dashboard", "post-deploy operations view") + "\n\n")

	if m.target == targetOCI {
		b.WriteString(featureRow("Selected", "OCI VM + private L1 + T-REX compliance suite", contentWidth))
		b.WriteString(featureRow("Before deploy", "complete the OCI tab; secrets stay local", contentWidth))
	} else {
		b.WriteString(featureRow("Selected", "local private L1 + T-REX, C-Chain rail planned", contentWidth))
		b.WriteString(featureRow("Network file", networkPath(targetLocal), contentWidth))
	}
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("CURRENT ENVIRONMENT") + "\n")
	if snap.net == nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render("No deployed network found. Select Deploy / reconcile and press Enter.") + "\n")
	} else {
		b.WriteString(featureRow("Name", snap.net.Name, contentWidth))
		b.WriteString(featureRow("Chain ID", fmt.Sprintf("%d", snap.net.ChainID), contentWidth))
		b.WriteString(featureRow("RPC", snap.net.RPCURL, contentWidth))
		b.WriteString(featureRow("Contracts", fmt.Sprintf("%d tracked", len(snap.net.Contracts)), contentWidth))
	}
	return b.String()
}

func (m wizardModel) explorerTab(contentWidth int) string {
	var b strings.Builder
	x := loadExplorerSnapshot(m.target, 6)
	transfers, transferErr := trexTransferHistory(m.target, "", 8)
	b.WriteString(styleSectionTitle.Render("EMBEDDED EXPLORER") + "\n")
	if x.Err != "" {
		b.WriteString("  " + dot(red) + "  " + styleRed.Render(x.Err) + "\n")
		b.WriteString(styleDim.Render("  Deploy or reconnect the selected private L1, then return here.") + "\n")
		return b.String()
	}
	b.WriteString(featureRow("Latest block", x.BlockHeight, contentWidth))
	b.WriteString(featureRow("Source", "direct JSON-RPC from selected L1", contentWidth))
	b.WriteString("\n" + styleSectionTitle.Render("RECENT BLOCKS") + "\n")
	for _, block := range x.Blocks {
		hash := shortAddr(block.Hash)
		b.WriteString("  " + styleKicker.Render("#"+fmt.Sprintf("%-8s", block.Number)) +
			styleValue.Render(fmt.Sprintf(" tx %-3d gas %-10s ", block.TxCount, block.GasUsed)) +
			styleDim.Render(block.Timestamp+"  "+hash) + "\n")
		if len(block.Transactions) > 0 {
			for i, tx := range block.Transactions {
				if i >= 2 {
					b.WriteString(styleDim.Render("             ...") + "\n")
					break
				}
				b.WriteString(styleDim.Render("             tx "+shortAddr(tx)) + "\n")
			}
		}
	}
	b.WriteString("\n" + styleSectionTitle.Render("T-REX TRANSFERS") + "\n")
	if transferErr != nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render(oneLine(transferErr.Error(), 110)) + "\n")
		return b.String()
	}
	if len(transfers) == 0 {
		b.WriteString(styleDim.Render("  No CEQ Transfer events yet. Send from T-REX, then refresh this tab.") + "\n")
		return b.String()
	}
	for _, tx := range transfers {
		b.WriteString("  " + styleKicker.Render("#"+fmt.Sprintf("%-6s", tx.Block)) +
			styleValue.Render(fmt.Sprintf(" %-10s CEQ ", tx.Amount)) +
			styleDim.Render(shortAddr(tx.From)+" -> "+shortAddr(tx.To)+"  "+shortAddr(tx.TxHash)) + "\n")
	}
	return b.String()
}

func (m wizardModel) contractsTab(contentWidth int) string {
	var b strings.Builder
	snap := loadNetworkSnapshot(m.target)
	b.WriteString(styleSectionTitle.Render("CONTRACTS") + "\n")
	if snap.net == nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render("Deploy a network first. Contracts are loaded from network.json.") + "\n")
		return b.String()
	}
	for i, c := range snap.contracts {
		prefix := "  "
		name := styleValue.Render(fmt.Sprintf("%-28s", c.Name))
		if i == m.itemCursor {
			prefix = styleGreen.Render("› ")
			name = styleButtonActive.Render(fmt.Sprintf("%-28s", c.Name))
		}
		b.WriteString(prefix + name + styleGreen.Render(c.Address) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("ACTIONS") + "\n")
	b.WriteString(featureRow("Enter / A", "copy selected contract address", contentWidth))
	b.WriteString(featureRow("Inspect", "claw1 inspect --local --json", contentWidth))
	return b.String()
}

func (m wizardModel) trexTab(contentWidth int) string {
	var b strings.Builder
	snap := loadNetworkSnapshot(m.target)
	b.WriteString(styleSectionTitle.Render("T-REX WORKSPACE") + "\n")
	if snap.net == nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render("Deploy a network first. T-REX needs the selected L1 RPC and CEQ token.") + "\n")
		return b.String()
	}
	token := trexTokenAddress(snap.net)
	identity := identityRegistryAddress(snap.net)
	sender := demoWallets()[0]
	recipient := selectedTrexRecipient(m.itemCursor)
	sim := simulateTrexTransfer(m.target, recipient.Address, "1")
	verdict := sim.Message
	if sim.Approved {
		verdict = styleGreen.Render(sim.Message)
	} else {
		verdict = styleYellow.Render(sim.Message)
	}
	b.WriteString(featureRow("Sender", sender.Name+"  "+sender.Address, contentWidth))
	b.WriteString(featureRow("Native balance", walletBalance(snap.net.RPCURL, sender.Address)+"  nonce "+walletNonce(snap.net.RPCURL, sender.Address), contentWidth))
	if token == "" {
		b.WriteString(featureRow("CEQ token", "not deployed", contentWidth))
	} else {
		b.WriteString(featureRow("CEQ balance", trexBalance(snap.net.RPCURL, token, sender.Address), contentWidth))
	}
	b.WriteString(featureRow("Recipient", recipient.Name+"  "+recipient.Address, contentWidth))
	b.WriteString(featureRow("Amount", "1 CEQ", contentWidth))
	b.WriteString(featureRow("Simulation", verdict, contentWidth))
	b.WriteString(featureRow("C-Chain", "planned rail; CEQ transfer executes on local private L1 today", contentWidth))
	b.WriteString("\n" + styleSectionTitle.Render("RECIPIENTS") + "\n")
	for i, r := range trexRecipients() {
		prefix := "  "
		name := styleValue.Render(fmt.Sprintf("%-24s", r.Name))
		if i == m.itemCursor {
			prefix = styleGreen.Render("› ")
			name = styleButtonActive.Render(fmt.Sprintf("%-24s", r.Name))
		}
		verified := "unverified"
		if trexIsVerified(snap.net.RPCURL, identity, r.Address) {
			verified = "verified"
		}
		ceq := "n/a"
		if token != "" {
			ceq = trexBalance(snap.net.RPCURL, token, r.Address)
		}
		b.WriteString(prefix + name + styleGreen.Render(r.Address) + styleDim.Render("  "+verified+"  "+ceq) + "\n")
	}
	b.WriteString("\n" + styleSectionTitle.Render("ACTIONS") + "\n")
	b.WriteString(featureRow("Enter / R", "simulate selected 1 CEQ transfer", contentWidth))
	b.WriteString(featureRow("T", "send selected 1 CEQ transfer if simulation passes", contentWidth))
	b.WriteString(featureRow("A", "copy selected recipient address", contentWidth))
	b.WriteString(featureRow("K", "copy deployer private key for local demo wallet", contentWidth))
	b.WriteString(featureRow("CLI", "claw1 wallet send --to <addr> --amount <n> --json", contentWidth))
	b.WriteString("\n" + styleSectionTitle.Render("TRANSFER HISTORY") + "\n")
	transfers, err := trexTransferHistory(m.target, sender.Address, 6)
	if err != nil {
		b.WriteString("  " + dot(yellow) + "  " + styleYellow.Render(oneLine(err.Error(), 100)) + "\n")
		return b.String()
	}
	if len(transfers) == 0 {
		b.WriteString(styleDim.Render("  No CEQ Transfer events for sender yet.\n"))
		return b.String()
	}
	for _, tx := range transfers {
		b.WriteString("  " + styleKicker.Render("#"+fmt.Sprintf("%-6s", tx.Block)) +
			styleValue.Render(fmt.Sprintf(" %-10s CEQ ", tx.Amount)) +
			styleDim.Render(shortAddr(tx.From)+" -> "+shortAddr(tx.To)+"  "+shortAddr(tx.TxHash)) + "\n")
	}
	return b.String()
}

func (m wizardModel) optionRow(cursor int, selected bool, label, desc string) string {
	prefix := "  "
	marker := circle()
	if selected {
		marker = dot(green)
	}
	body := fmt.Sprintf("[ %s  %-28s %s ]", marker, label, desc)
	if m.activeTab == tabDeploy && m.deployCursor == cursor {
		prefix = styleGreen.Render("› ")
		body = styleButtonActive.Render(body)
	} else {
		body = styleButton.Render(body)
	}
	return prefix + body
}

func (m wizardModel) primaryAction() string {
	label := "Start deployment"
	if m.target == targetOCI {
		label = "Start OCI deployment"
	}
	body := styleButton.Render("[ " + label + " ]")
	if m.activeTab == tabDeploy && m.deployCursor == deployCursorStart {
		body = styleButtonActive.Render("[ " + label + " ]")
	}
	return "  " + body + "\n"
}

func (m wizardModel) cloudTab(contentWidth int) string {
	var b strings.Builder
	if m.target != targetOCI {
		b.WriteString(styleSectionTitle.Render("CLOUD") + "\n")
		b.WriteString(styleDim.Render("  Current target is Developer appliance. Select Production target in Deploy to edit cloud settings.\n"))
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("PRODUCTION CHECKLIST") + "\n")
		b.WriteString(featureRow("forge", "build and deploy contracts", contentWidth))
		b.WriteString(featureRow("avalanche-cli", "start the local network and L1", contentWidth))
		b.WriteString(featureRow("terraform", "declare and apply infrastructure", contentWidth))
		b.WriteString(featureRow("OCI destroy", "dry-run, inventory, destroy, verify, evidence", contentWidth))
		return b.String()
	}

	b.WriteString(styleSectionTitle.Render("CLOUD PRODUCTION TARGET") + "\n")
	b.WriteString(m.inputRow("Tenancy OCID", inTenancy))
	b.WriteString(m.inputRow("User OCID", inUser))
	b.WriteString(m.inputRow("Fingerprint", inFingerprint))
	b.WriteString(m.inputRow("API key path", inKeyPath))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("INFRASTRUCTURE") + "\n")
	b.WriteString(m.inputRow("Region", inRegion))
	b.WriteString(m.inputRow("Shape", inShape))
	b.WriteString(m.inputRow("OCPUs", inOcpus))
	b.WriteString(m.inputRow("Memory (GB)", inMemory))
	b.WriteString("\n" + styleSectionTitle.Render("DESTROY SAFETY") + "\n")
	b.WriteString(featureRow("Default", "dry-run unless --yes is explicit", contentWidth))
	b.WriteString(featureRow("Evidence", "--preserve-evidence is local-only; --evidence-bucket retains cloud data", contentWidth))
	b.WriteString(styleDim.Render("\n  [↑/↓] move field. Secrets are read locally, not written to evidence.\n"))
	return b.String()
}

func featureRow(label, value string, width int) string {
	labelWidth := 22
	valueWidth := width - labelWidth - 8
	if valueWidth < 32 {
		valueWidth = 32
	}
	return "  " + styleKicker.Render("│") + "  " +
		styleValue.Width(labelWidth).Render(label) +
		styleDim.Width(valueWidth).Render(value) + "\n"
}

func actionRow(active bool, label, key, desc string) string {
	style := styleButton
	prefix := "  "
	if active {
		style = styleButtonActive
		prefix = styleGreen.Render("› ")
	}
	return prefix + style.Render(fmt.Sprintf("[ %-18s ]", label)) + " " + styleDim.Render(fmt.Sprintf("%-12s %s", key, desc))
}

func (m wizardModel) inputRow(label string, idx int) string {
	style := styleInputInactive
	prefix := "  "
	if m.focus == idx && m.target == targetOCI {
		style = styleInputActive
		prefix = lipgloss.NewStyle().Foreground(green).Render("› ")
	}
	return prefix + style.Render(fmt.Sprintf("%-14s", label)) +
		" " + m.inputs[idx].View() + "\n"
}

type networkSnapshot struct {
	net       *networkJSON
	contracts []contract
}

func loadNetworkSnapshot(target deployTarget) networkSnapshot {
	data, err := os.ReadFile(networkPath(target))
	if err != nil {
		return networkSnapshot{}
	}
	var net networkJSON
	if err := json.Unmarshal(data, &net); err != nil {
		return networkSnapshot{}
	}
	return networkSnapshot{net: &net, contracts: net.Contracts}
}

func clampCursor(next, count int) int {
	if count <= 0 {
		return 0
	}
	if next < 0 {
		return count - 1
	}
	if next >= count {
		return 0
	}
	return next
}

func walletBalance(rpcURL, address string) string {
	result, err := rpcString(rpcURL, "eth_getBalance", []any{address, "latest"})
	if err != nil {
		return "unreachable"
	}
	wei := new(big.Int)
	wei.SetString(strings.TrimPrefix(result, "0x"), 16)
	eth := new(big.Rat).SetFrac(wei, big.NewInt(1_000_000_000_000_000_000))
	return eth.FloatString(4) + " CLAW"
}

func walletNonce(rpcURL, address string) string {
	result, err := rpcString(rpcURL, "eth_getTransactionCount", []any{address, "latest"})
	if err != nil {
		return "?"
	}
	n := new(big.Int)
	n.SetString(strings.TrimPrefix(result, "0x"), 16)
	return n.String()
}

func cChainRailStatus() string {
	if os.Getenv("C_CHAIN_RPC_URL") != "" || os.Getenv("C_CHAIN_BLOCKCHAIN_ID") != "" {
		return "workbench env detected"
	}
	return "planned, not deployed"
}

func simulateKYCRead(target deployTarget) string {
	snap := loadNetworkSnapshot(target)
	if snap.net == nil {
		return "Deploy a network first."
	}
	identityRegistry := findContract(snap.net, "IdentityRegistry")
	if identityRegistry == "" {
		return "IdentityRegistry not found in network.json."
	}
	out, err := exec.Command("cast", "call", identityRegistry, "isVerified(address)", "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC", "--rpc-url", snap.net.RPCURL).CombinedOutput()
	if err != nil {
		return "Simulation failed: " + oneLine(string(out), 100)
	}
	result := strings.TrimSpace(string(out))
	if result == "true" || strings.HasSuffix(result, "1") {
		return "KYC read passed: demo investor is verified."
	}
	return "KYC read returned: " + result
}
