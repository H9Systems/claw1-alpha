package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
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
	enableICTT  bool
}

type wizardModel struct {
	items       []wizardItem
	cursor      int
	editing     bool
	editInput   textinput.Model
	showPreview bool

	target   deployTarget
	repoRoot string
	cache    wizardCache
	err      string
	action   string
}

func newWizardModel(repoRoot string) wizardModel {
	ti := textinput.New()
	ti.Width = 30
	return wizardModel{
		items:     defaultWizardItems(),
		target:    targetLocal,
		repoRoot:  repoRoot,
		editInput: ti,
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Async cache results ──────────────────────────────────────────────

	case cacheNetMsg:
		old := m.cache.snap
		m.cache.snap = msg.snap
		if old.net == nil && msg.snap.net != nil {
			return m, nil
		}
	case cacheBlockMsg:
		m.cache.blockHeight = msg.height
		m.cache.blockErr = msg.errStr
	case cacheExplorerMsg:
		m.cache.explorer = msg.snap
	case cacheAllTransfersMsg:
		m.cache.allTransfers = msg.transfers
		m.cache.allTransfersErr = msg.errStr
	case cacheSenderTransfersMsg:
		m.cache.senderTransfers = msg.transfers
		m.cache.senderTransfersErr = msg.errStr
	case cacheSenderInfoMsg:
		m.cache.senderBalance = msg.balance
		m.cache.senderNonce = msg.nonce
		m.cache.senderCEQ = msg.ceq
	case cacheRecipientInfoMsg:
		m.cache.recipientVerified = msg.verified
		m.cache.recipientCEQ = msg.ceq
	case cacheSimulationMsg:
		m.cache.simulation = msg.sim
		m.cache.simCursor = msg.cursor
		m.cache.simDone = true
	case cacheSendResultMsg:
		m.action = msg.action
	case wizardTickMsg:
		cmds := []tea.Cmd{wizardTickCmd(), fetchNetworkCmd(m.target)}
		return m, tea.Batch(cmds...)

	// ── Keyboard ─────────────────────────────────────────────────────────

	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter", "tab", "esc":
				m.items[m.cursor].Value = m.editInput.Value()
				m.editing = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.editInput, cmd = m.editInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "down", "j":
			m.cursor = m.nextNavigable(1)
		case "up", "k":
			m.cursor = m.nextNavigable(-1)
		case " ":
			m.interact()
		case "enter":
			it := &m.items[m.cursor]
			switch it.Kind {
			case itemText:
				m.editing = true
				m.editInput.SetValue(it.Value)
				m.editInput.Focus()
				return m, nil
			case itemRadio, itemToggle:
				m.interact()
			case itemAction:
				if it.ID == "preview" {
					m.showPreview = !m.showPreview
				}
				// deploy/destroy/dashboard handled by main.go
			}
		case "tab":
			m.cursor = m.nextSection(1)
		case "shift+tab":
			m.cursor = m.nextSection(-1)
		}

	case copyDoneMsg:
		m.action = string(msg)
	case explorerDoneMsg:
		m.action = string(msg)
	}

	return m, nil
}

func (m *wizardModel) interact() {
	it := &m.items[m.cursor]
	if it.Locked {
		return
	}
	switch it.Kind {
	case itemToggle:
		it.On = !it.On
	case itemRadio:
		wizardSelectRadio(m.items, it.Group, it.ID)
		if it.Group == "target" {
			if it.ID == "t_oci" {
				m.target = targetOCI
			} else {
				m.target = targetLocal
			}
		}
	}
}

func (m wizardModel) nextNavigable(dir int) int {
	n := len(m.items)
	pos := m.cursor
	for i := 0; i < n; i++ {
		pos = (pos + dir + n) % n
		if m.items[pos].navigable() {
			return pos
		}
	}
	return m.cursor
}

func (m wizardModel) nextSection(dir int) int {
	n := len(m.items)
	pos := m.cursor
	for i := 0; i < n; i++ {
		pos = (pos + dir + n) % n
		if m.items[pos].Kind == itemHeading || m.items[pos].Kind == itemDivider {
			for j := 1; j < n; j++ {
				next := (pos + j) % n
				if m.items[next].navigable() {
					return next
				}
				if m.items[next].Kind == itemHeading || m.items[next].Kind == itemDivider {
					break
				}
			}
		}
	}
	return m.cursor
}

// ── Activate / Validate ───────────────────────────────────────────────────────

func (m wizardModel) validate() (deployConfig, error) {
	return deployConfig{
		target:     m.target,
		repoRoot:   m.repoRoot,
		enableICTT: wizardIsOn(m.items, "ictt"),
	}, nil
}

func (m *wizardModel) activate() bool {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return false
	}
	return m.items[m.cursor].Kind == itemAction && m.items[m.cursor].ID == "deploy"
}

func (m wizardModel) openDashboard() bool {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return false
	}
	return m.items[m.cursor].Kind == itemAction && m.items[m.cursor].ID == "dashboard"
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m wizardModel) View(width int) string {
	var b strings.Builder
	cw := width - 6
	if cw < 72 {
		cw = 72
	}

	// Header
	jur := wizardJurisdiction(m.items)
	b.WriteString(styleBrand.Render("CLAW1") + "  " +
		styleHeader.Render("L1 DEPLOYMENT WIZARD") + "  " +
		statusPill(jur, blue) + "\n")
	b.WriteString(styleKicker.Render("  Configure your private Avalanche L1 with compliance built in.") + "\n")
	b.WriteString(styleDim.Render("  Like OpenZeppelin Wizard — but for regulated blockchain infrastructure.") + "\n")

	// Status
	if snap := m.cache.snap; snap.net != nil {
		bh := m.cache.blockHeight
		if bh == "" {
			bh = "..."
		}
		b.WriteString("\n  " + dot(green) + " " +
			styleGreen.Render("L1 active") + "  " +
			styleDim.Render(snap.net.Name+" · chain "+fmt.Sprintf("%d", snap.net.ChainID)+" · block #"+bh) + "\n")
	}

	// Items
	for i, it := range m.items {
		b.WriteString(m.renderItem(it, i == m.cursor, cw))
	}

	// Error/Action
	if m.err != "" {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err) + "\n")
	}
	if m.action != "" {
		b.WriteString("\n" + styleYellow.Render("  "+m.action) + "\n")
	}

	// Preview
	if m.showPreview {
		b.WriteString("\n" + styleSectionTitle.Render("  TERRAFORM PREVIEW") + "\n")
		b.WriteString(rule(cw) + "\n")
		for _, line := range strings.Split(terraformPreview(m.items), "\n") {
			b.WriteString(styleDim.Render("  "+line) + "\n")
		}
		b.WriteString(rule(cw) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [↑/↓] navigate   [Space/Enter] select   [Tab] next section   [Q] quit"))

	return styleBox.Width(width - 4).Render(b.String())
}

func (m wizardModel) renderItem(it wizardItem, focused bool, width int) string {
	pfx := "  "
	if focused {
		pfx = styleGreen.Render("› ")
	}

	switch it.Kind {
	case itemHeading:
		return "\n  " + styleSectionTitle.Render(it.Label) + "\n"

	case itemDivider:
		pad := width - lipgloss.Width(it.Label) - 6
		if pad < 4 {
			pad = 4
		}
		l := pad / 2
		r := pad - l
		return "\n  " + styleDim.Render(strings.Repeat("═", l)) +
			" " + styleHeader.Render(it.Label) + " " +
			styleDim.Render(strings.Repeat("═", r)) + "\n"

	case itemRadio:
		mk := styleDim.Render("○")
		if it.On {
			mk = styleGreen.Render("●")
		}
		lbl := styleValue.Render(it.Label)
		if focused {
			lbl = styleGreen.Render(it.Label)
		}
		return pfx + mk + " " + fmt.Sprintf("%-36s", lbl) + styleDim.Render(it.Desc) + "\n"

	case itemToggle:
		mk := styleDim.Render("○")
		if it.On {
			mk = styleGreen.Render("✓")
		}
		if it.Locked {
			if it.On {
				mk = styleBlue.Render("✓")
			} else {
				mk = styleDim.Render("○")
			}
		}
		lbl := styleValue.Render(it.Label)
		if focused {
			lbl = styleGreen.Render(it.Label)
		}
		extra := ""
		if it.Locked {
			extra = styleDim.Render(" [required]")
		}
		if it.Warn != "" {
			extra += "  " + styleRed.Render("⚠ "+it.Warn)
		}
		return pfx + mk + " " + fmt.Sprintf("%-36s", lbl) + styleDim.Render(it.Desc) + extra + "\n"

	case itemText:
		lbl := styleLabel.Render(fmt.Sprintf("%-16s", it.Label))
		val := styleGreen.Render("[" + it.Value + "]")
		if focused && m.editing {
			val = styleYellow.Render("[" + m.editInput.View() + "]")
		} else if focused {
			val = styleButtonActive.Render("[" + it.Value + "]")
		}
		return pfx + lbl + " " + val + "  " + styleDim.Render(it.Desc) + "\n"

	case itemAction:
		body := styleButton.Render("  " + it.Label + "  ")
		if focused {
			body = styleButtonActive.Render("  " + it.Label + "  ")
		}
		return pfx + body + "  " + styleDim.Render(it.Desc) + "\n"

	case itemInfo:
		return "  " + styleDim.Render(it.Label) + "\n"
	}
	return ""
}

// ── Helpers kept for other files ──────────────────────────────────────────────

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
