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
}

type wizardModel struct {
	target   deployTarget
	focus    int
	inputs   []textinput.Model
	err      string
	repoRoot string
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
		return deployConfig{target: targetLocal, repoRoot: m.repoRoot}, nil
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
	}, nil
}

func (m wizardModel) View(width int) string {
	var b strings.Builder

	// Title
	b.WriteString(styleHeader.Render("CLAW1") + "  " +
		styleDim.Render("Compliance Deploy Wizard") + "\n\n")

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
	} else {
		b.WriteString(styleGreen.Render("  No credentials needed — deploys a local Avalanche devnet.\n"))
		b.WriteString(styleValue.Render("  Requires: forge, avalanche-cli, terraform, docker, jq\n"))
	}

	if m.err != "" {
		b.WriteString("\n" + styleRed.Render("  ✗ "+m.err) + "\n")
	}

	b.WriteString(styleKeys.Render("\n  [Tab] next field   [D] deploy   [Q] quit"))

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
