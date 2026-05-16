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
	enableICTT  bool // deploy ICTT TokenHome+TokenRemote (OCI only; requires scripts/ictt-setup.sh)
}

type wizardModel struct {
	target     deployTarget
	focus      int
	inputs     []textinput.Model
	err        string
	repoRoot   string
	enableICTT bool
}

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

	inputs[inTenancy].Focus()

	return wizardModel{
		target:   targetOCI,
		focus:    inTenancy,
		inputs:   inputs,
		repoRoot: repoRoot,
	}
}

func (m wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			if m.target == targetOCI {
				m.focus = (m.focus + 1) % numInputs
			}
			m.syncFocus()
		case "shift+tab", "up":
			if m.target == targetOCI {
				m.focus = (m.focus - 1 + numInputs) % numInputs
			}
			m.syncFocus()
		case "1":
			m.target = targetOCI
			m.syncFocus()
		case "2":
			m.target = targetLocal
			for i := range m.inputs {
				m.inputs[i].Blur()
			}
		case "i", "I":
			if m.target == targetOCI {
				m.enableICTT = !m.enableICTT
			}
		}
	}

	if m.target == targetOCI {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *wizardModel) syncFocus() {
	for i := range m.inputs {
		if i == m.focus {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
}

func (m wizardModel) validate() (deployConfig, error) {
	if m.target == targetLocal {
		return deployConfig{target: targetLocal, repoRoot: m.repoRoot, enableICTT: false}, nil
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

func (m wizardModel) View(width int) string {
	var b strings.Builder

	// Title
	b.WriteString(styleHeader.Render("CLAW1") + "  " +
		styleDim.Render("Compliance TUI") + "\n")
	b.WriteString(styleDim.Render("  One tool: provision, inspect, transact, preserve evidence, destroy.") + "\n\n")

	// Target selection
	b.WriteString(styleSectionTitle.Render("DEPLOY TARGET") + "\n")
	if m.target == targetOCI {
		b.WriteString("  " + dot(green) + " Oracle Cloud Infrastructure (OCI)\n")
		b.WriteString("  " + circle() + " " + styleDim.Render("Local (on-prem devnet)") + "\n")
	} else {
		b.WriteString("  " + circle() + " " + styleDim.Render("Oracle Cloud Infrastructure (OCI)") + "\n")
		b.WriteString("  " + dot(green) + " Local (on-prem devnet)\n")
	}
	b.WriteString(styleDim.Render("  [1] OCI   [2] Local") + "\n\n")

	if m.target == targetOCI {
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
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("COMPLIANCE SUITE") + "\n")
		b.WriteString("  " + dot(green) + "  ERC-3643 T-REX    " + styleDim.Render("identity-bound token, KYC claim, ONCHAINID") + "\n")
		b.WriteString("  " + dot(green) + "  ComplianceRegistry " + styleDim.Render("on-chain KYC / jurisdiction enforcement") + "\n")
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("ICTT BRIDGE") + "\n")
		if m.enableICTT {
			b.WriteString("  " + dot(green) + " " + styleGreen.Render("Enabled") +
				styleDim.Render("  C-chain → L1 wrapped-token bridge (Teleporter/Warp)") + "\n")
			b.WriteString(styleDim.Render("  Requires: scripts/ictt-setup.sh  |  Fuji Teleporter Registry wired") + "\n")
		} else {
			b.WriteString("  " + circle() + " " + styleDim.Render("Disabled  (press [I] to enable)") + "\n")
		}
	} else {
		b.WriteString(styleGreen.Render("  No credentials needed — deploys a local Avalanche devnet.\n"))
		b.WriteString(styleValue.Render("  Requires: forge, avalanche-cli, terraform, docker, jq\n"))
		b.WriteString("\n")
		b.WriteString(styleSectionTitle.Render("COMPLIANCE SUITE") + "\n")
		b.WriteString("  " + dot(green) + "  ERC-3643 T-REX    " + styleDim.Render("identity-bound token, KYC claim, ONCHAINID") + "\n")
		b.WriteString("  " + dot(green) + "  ComplianceRegistry " + styleDim.Render("on-chain KYC / jurisdiction enforcement") + "\n")
		b.WriteString(styleDim.Render("  ICTT bridge not available for local devnet (no Fuji C-chain)") + "\n")
	}

	if m.err != "" {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err) + "\n")
	}

	b.WriteString("\n" + styleSectionTitle.Render("SCRIPTABLE OPERATIONS") + "\n")
	b.WriteString(styleDim.Render("  deploy:   claw1 deploy --oci --yes [--json]") + "\n")
	b.WriteString(styleDim.Render("  destroy:  claw1 destroy --oci --dry-run | claw1 destroy --oci --yes [--json]") + "\n")
	b.WriteString(styleDim.Render("  inspect:  claw1 inspect --oci [--json]") + "\n")
	b.WriteString(styleDim.Render("  wallets:  claw1 wallet list [--json]") + "\n")

	keys := "  [Tab] next field   [D] deploy   [Q] quit"
	if m.target == targetOCI {
		keys = "  [Tab] next field   [I] toggle ICTT   [D] deploy   [Q] quit"
	}
	b.WriteString(styleKeys.Render("\n" + keys))

	inner := b.String()
	return styleBox.Width(width - 4).Render(inner)
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
