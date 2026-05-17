package main

import (
	"fmt"
	"os"
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
	focus        int
	inputs       []textinput.Model
	err          string
	repoRoot     string
	enableICTT   bool
}

const (
	tabDeploy = iota
	tabCompliance
	tabOperations
	tabOCI
	numTabs
)

const (
	deployCursorOCI = iota
	deployCursorLocal
	deployCursorStart
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
		target:   targetLocal,
		focus:    inTenancy,
		inputs:   inputs,
		repoRoot: repoRoot,
	}
}

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % numTabs
			m.syncFocus()
		case "shift+tab", "left":
			m.activeTab = (m.activeTab - 1 + numTabs) % numTabs
			m.syncFocus()
		case "down":
			switch {
			case m.activeTab == tabDeploy:
				m.deployCursor = (m.deployCursor + 1) % numDeployCursors
			case m.activeTab == tabOCI && m.target == targetOCI:
				m.focus = (m.focus + 1) % numInputs
			}
			m.syncFocus()
		case "up":
			switch {
			case m.activeTab == tabDeploy:
				m.deployCursor = (m.deployCursor - 1 + numDeployCursors) % numDeployCursors
			case m.activeTab == tabOCI && m.target == targetOCI:
				m.focus = (m.focus - 1 + numInputs) % numInputs
			}
			m.syncFocus()
		}
	}

	if m.activeTab == tabOCI && m.target == targetOCI {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *wizardModel) syncFocus() {
	for i := range m.inputs {
		if m.activeTab == tabOCI && m.target == targetOCI && i == m.focus {
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
	case deployCursorStart:
		return true
	}
	return false
}

func (m wizardModel) View(width int) string {
	var b strings.Builder
	contentWidth := width - 8
	if contentWidth < 72 {
		contentWidth = 72
	}

	// Title
	b.WriteString(styleHeader.Render("CLAW1") + "  " +
		styleDim.Render("Regulated asset appliance") + "\n")
	b.WriteString(styleValue.Render("  Deploy a permissioned L1 with ERC-3643 transfer controls.") + "\n\n")
	b.WriteString(m.tabs() + "\n\n")

	switch m.activeTab {
	case tabDeploy:
		b.WriteString(m.deployTab(contentWidth))
	case tabCompliance:
		b.WriteString(m.complianceTab(contentWidth))
	case tabOperations:
		b.WriteString(m.operationsTab())
	case tabOCI:
		b.WriteString(m.ociTab(contentWidth))
	}

	if m.err != "" {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [←/→] tabs   [↑/↓] select   [Enter] activate   [Q] quit"))

	inner := b.String()
	return styleBox.Width(width - 4).Render(inner)
}

func (m wizardModel) tabs() string {
	names := []string{"Deploy", "Compliance", "Operations", "OCI Config"}
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

func (m wizardModel) deployTab(contentWidth int) string {
	var b strings.Builder
	b.WriteString(styleSectionTitle.Render("DEPLOY TARGET") + "\n")
	b.WriteString(m.optionRow(deployCursorOCI, m.target == targetOCI, "Oracle Cloud Infrastructure", "cloud L1 with OCI VM") + "\n")
	b.WriteString(m.optionRow(deployCursorLocal, m.target == targetLocal, "Local devnet", "single-machine demo appliance") + "\n\n")

	if m.target == targetOCI {
		b.WriteString(featureRow("Selected path", "OCI VM + Avalanche L1 + T-REX", contentWidth))
		b.WriteString(featureRow("Before deploy", "fill the OCI Config tab", contentWidth))
	} else {
		b.WriteString(featureRow("Selected path", "Avalanche devnet + custom L1", contentWidth))
		b.WriteString(featureRow("OCI adds", "multi-node infra, hardened keys, RBAC", contentWidth))
	}

	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("WHAT DEPLOY DOES") + "\n")
	b.WriteString(featureRow("1. Provision L1", "Terraform creates the Avalanche L1", contentWidth))
	b.WriteString(featureRow("2. Deploy T-REX", "token, registry, KYC issuer", contentWidth))
	b.WriteString(featureRow("3. Prove KYC gate", "verified succeeds, unknown must revert", contentWidth))
	b.WriteString(featureRow("4. Evidence", "addresses and tx hashes stay local", contentWidth))
	b.WriteString("\n")
	b.WriteString(m.primaryAction())
	return b.String()
}

func (m wizardModel) complianceTab(contentWidth int) string {
	var b strings.Builder
	b.WriteString(styleSectionTitle.Render("REGULATORY PRESET") + "\n")
	b.WriteString(featureRow("CNBV-style asset", "TxAllowList + KYC claim + ERC-3643", contentWidth))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("COMPLIANCE SUITE") + "\n")
	b.WriteString(featureRow("ERC-3643 T-REX", "identity-bound token, KYC claim", contentWidth))
	b.WriteString(featureRow("IdentityRegistry", "verified wallets can receive tokens", contentWidth))
	b.WriteString(featureRow("ClaimIssuer", "demo KYC authority for investor claims", contentWidth))
	b.WriteString(featureRow("ComplianceRegistry", "KYC and jurisdiction evidence", contentWidth))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("DEMO CHECK") + "\n")
	b.WriteString(featureRow("Expected success", "issuer sends tokens to verified wallet", contentWidth))
	b.WriteString(featureRow("Expected revert", "issuer sends tokens to unknown wallet", contentWidth))
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

func (m wizardModel) operationsTab() string {
	var b strings.Builder
	b.WriteString(styleSectionTitle.Render("SCRIPTABLE OPERATIONS") + "\n")
	b.WriteString(styleDim.Render("  local:    claw1 deploy --local [--json]") + "\n")
	b.WriteString(styleDim.Render("            claw1 destroy --local [--json]") + "\n")
	b.WriteString(styleDim.Render("  oci:      claw1 deploy --oci --yes [--json]") + "\n")
	b.WriteString(styleDim.Render("            claw1 destroy --oci --dry-run") + "\n")
	b.WriteString(styleDim.Render("  inspect:  claw1 inspect --local [--json]") + "\n")
	b.WriteString(styleDim.Render("  wallets:  claw1 wallet list [--json]") + "\n")
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("DEMO RESET") + "\n")
	b.WriteString(styleDim.Render("  scripts/reset.sh") + "\n")
	b.WriteString(styleDim.Render("  Demo state stays in ~/.claw1/{name}/network.json") + "\n")
	return b.String()
}

func (m wizardModel) ociTab(contentWidth int) string {
	var b strings.Builder
	if m.target != targetOCI {
		b.WriteString(styleSectionTitle.Render("OCI CONFIG") + "\n")
		b.WriteString(styleDim.Render("  Current target is Local. Press [1] to configure OCI deployment.\n"))
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("LOCAL REQUIREMENTS") + "\n")
		b.WriteString(featureRow("forge", "build and deploy contracts", contentWidth))
		b.WriteString(featureRow("avalanche-cli", "start the local network and L1", contentWidth))
		b.WriteString(featureRow("terraform", "declare and apply infrastructure", contentWidth))
		b.WriteString(featureRow("docker + jq", "support scripts and local tooling", contentWidth))
		return b.String()
	}

	b.WriteString(styleSectionTitle.Render("OCI CREDENTIALS") + "\n")
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
	b.WriteString(styleDim.Render("\n  [↑/↓] move field. Secrets are read locally, not written to evidence.\n"))
	return b.String()
}

func featureRow(label, value string, width int) string {
	labelWidth := 22
	valueWidth := width - labelWidth - 8
	if valueWidth < 32 {
		valueWidth = 32
	}
	return "  " + dot(green) + "  " +
		styleValue.Width(labelWidth).Render(label) +
		styleDim.Width(valueWidth).Render(value) + "\n"
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
