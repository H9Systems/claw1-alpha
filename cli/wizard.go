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

	b.WriteString(styleBrand.Render("CLAW1") + "  " +
		styleHeader.Render("PRIVATE L1 CONTROL PLANE") + "  " +
		styleDim.Render("open-core stack for regulated Avalanche deployments") + "\n")
	b.WriteString(styleKicker.Render("  Ship a sovereign chain with compliance, observability, and evidence in one run.") + "\n")
	b.WriteString(styleDim.Render("  Not a block explorer. Not a script pile. The appliance teams expected Alchemy to be for private L1s.") + "\n\n")
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
	names := []string{"Mission", "Compliance", "Operations", "OCI"}
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
	b.WriteString(styleSectionTitle.Render("MISSION") + "\n")
	b.WriteString(featureRow("Use case", "issue regulated debt tokens to verified wallets", contentWidth))
	b.WriteString(featureRow("Why L1", "compliance boundary stays native, liquidity can still route outward", contentWidth))
	b.WriteString(featureRow("Demo proof", "verified transfer passes; unknown wallet is rejected", contentWidth))
	b.WriteString("\n" + styleSectionTitle.Render("DEPLOYMENT RAIL") + "\n")
	b.WriteString(m.optionRow(deployCursorLocal, m.target == targetLocal, "Developer appliance", "local L1, fast repeatable demo") + "\n")
	b.WriteString(m.optionRow(deployCursorOCI, m.target == targetOCI, "Production target", "OCI VM, same Terraform spine") + "\n\n")

	if m.target == targetOCI {
		b.WriteString(featureRow("Selected", "OCI VM + Avalanche L1 + T-REX compliance suite", contentWidth))
		b.WriteString(featureRow("Before deploy", "complete the OCI tab; secrets stay local", contentWidth))
	} else {
		b.WriteString(featureRow("Selected", "Avalanche devnet + custom L1 + T-REX", contentWidth))
		b.WriteString(featureRow("Upgrade path", "OCI adds multi-node infra, hardened keys, RBAC", contentWidth))
	}

	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("RUNBOOK") + "\n")
	b.WriteString(featureRow("1. Provision", "Terraform declares and applies the L1", contentWidth))
	b.WriteString(featureRow("2. Compliance", "ERC-3643, IdentityRegistry, KYC issuer", contentWidth))
	b.WriteString(featureRow("3. Observe", "RPC, contracts, explorer, wallet surface", contentWidth))
	b.WriteString(featureRow("4. Preserve", "local evidence bundle and deploy receipt", contentWidth))
	b.WriteString("\n")
	b.WriteString(m.primaryAction())
	return b.String()
}

func (m wizardModel) complianceTab(contentWidth int) string {
	var b strings.Builder
	b.WriteString(styleSectionTitle.Render("FINANCIAL RAIL") + "\n")
	b.WriteString(featureRow("Status quo", "Hyperledger/Corda can gate wallets, but assets stay trapped", contentWidth))
	b.WriteString(featureRow("Public chain", "liquidity exists, but unrestricted receivers break AML controls", contentWidth))
	b.WriteString(featureRow("Claw1 L1", "verified wallets only, with a path to public liquidity via ICTT", contentWidth))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("DEFAULT TEMPLATE") + "\n")
	b.WriteString(featureRow("ERC-3643 / T-REX", "identity-bound token with transfer restrictions", contentWidth))
	b.WriteString(featureRow("IdentityRegistry", "only verified wallets can receive the asset", contentWidth))
	b.WriteString(featureRow("ClaimIssuer", "demo KYC authority signs investor claims", contentWidth))
	b.WriteString(featureRow("ComplianceRegistry", "jurisdiction and KYC evidence on-chain", contentWidth))
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("DEMO ASSERTION") + "\n")
	b.WriteString(featureRow("Verified wallet", "transfer approved", contentWidth))
	b.WriteString(featureRow("Unknown wallet", "transfer rejected by contract", contentWidth))
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
	b.WriteString(styleSectionTitle.Render("CONTROL SURFACE") + "\n")
	b.WriteString(styleDim.Render("  local:    claw1 deploy --local [--json]") + "\n")
	b.WriteString(styleDim.Render("            claw1 destroy --local [--json]") + "\n")
	b.WriteString(styleDim.Render("  oci:      claw1 deploy --oci --yes [--json]") + "\n")
	b.WriteString(styleDim.Render("            claw1 destroy --oci --dry-run") + "\n")
	b.WriteString(styleDim.Render("  inspect:  claw1 inspect --local [--json]") + "\n")
	b.WriteString(styleDim.Render("  wallets:  claw1 wallet list [--json]") + "\n")
	b.WriteString(styleDim.Render("  explorer: claw1 explorer start | status | open") + "\n")
	b.WriteString("\n")
	b.WriteString(styleSectionTitle.Render("STATE AND EVIDENCE") + "\n")
	b.WriteString(featureRow("State file", "~/.claw1/{name}/network.json", 84))
	b.WriteString(featureRow("Reset", "scripts/reset.sh", 84))
	b.WriteString(featureRow("Destroy posture", "OCI fails closed: dry-run, inventory, verify leftovers", 84))
	return b.String()
}

func (m wizardModel) ociTab(contentWidth int) string {
	var b strings.Builder
	if m.target != targetOCI {
		b.WriteString(styleSectionTitle.Render("OCI CONFIG") + "\n")
		b.WriteString(styleDim.Render("  Current rail is Developer appliance. Select Production target in Mission to edit OCI settings.\n"))
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("LOCAL REQUIREMENTS") + "\n")
		b.WriteString(featureRow("forge", "build and deploy contracts", contentWidth))
		b.WriteString(featureRow("avalanche-cli", "start the local network and L1", contentWidth))
		b.WriteString(featureRow("terraform", "declare and apply infrastructure", contentWidth))
		b.WriteString(featureRow("docker + jq", "support scripts and local tooling", contentWidth))
		return b.String()
	}

	b.WriteString(styleSectionTitle.Render("OCI PRODUCTION TARGET") + "\n")
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
	return "  " + styleKicker.Render("│") + "  " +
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
