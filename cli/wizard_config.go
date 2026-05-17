package main

import (
	"fmt"
	"strings"
)

// ── Item kinds ────────────────────────────────────────────────────────────────

const (
	itemHeading = "heading"
	itemDivider = "divider"
	itemRadio   = "radio"
	itemToggle  = "toggle"
	itemText    = "text"
	itemAction  = "action"
	itemInfo    = "info"
)

// wizardItem is a single element in the wizard configurator.
type wizardItem struct {
	ID     string
	Kind   string // heading, divider, radio, toggle, text, action, info
	Group  string // radio group name (only for radios)
	Label  string
	Desc   string
	Warn   string // warning shown next to the item
	On     bool   // state for toggles and radios
	Value  string // state for text inputs
	Locked bool   // cannot be changed (required for compliance)
}

// navigable returns true if the cursor can land on this item.
func (it wizardItem) navigable() bool {
	switch it.Kind {
	case itemRadio, itemToggle, itemText, itemAction:
		return true
	}
	return false
}

// ── Default wizard items ──────────────────────────────────────────────────────
// Maps the full terraform-provider-claw1 l1_resource schema + compliance layer.

func defaultWizardItems() []wizardItem {
	return []wizardItem{
		// ── BLOCKCHAIN ─────────────────────────────────────────────────────
		{ID: "h_chain", Kind: itemHeading, Label: "BLOCKCHAIN"},
		{ID: "name", Kind: itemText, Label: "Name", Value: "claw1demobank", Desc: "avalanche-cli blockchain identifier"},
		{ID: "chain_id", Kind: itemText, Label: "Chain ID", Value: "432260", Desc: "EVM chain ID in genesis block"},
		{ID: "token", Kind: itemText, Label: "Token", Value: "CLAW", Desc: "native token symbol (--evm-token)"},
		{ID: "vm_version", Kind: itemText, Label: "VM version", Value: "latest", Desc: "subnet-evm release (--vm-version)"},
		{ID: "prod_defaults", Kind: itemToggle, Label: "Production defaults", On: false, Desc: "--production-defaults (vs --test-defaults)"},

		// ── CONSENSUS ──────────────────────────────────────────────────────
		{ID: "h_consensus", Kind: itemHeading, Label: "CONSENSUS"},
		{ID: "poa", Kind: itemRadio, Group: "consensus", Label: "Proof of Authority", On: true, Desc: "admin-managed validators (default)"},
		{ID: "pos_native", Kind: itemRadio, Group: "consensus", Label: "Proof of Stake — native AVAX", Desc: "AVAX staking for validator selection"},
		{ID: "pos_erc20", Kind: itemRadio, Group: "consensus", Label: "Proof of Stake — ERC-20 token", Desc: "custom staking token"},
		{ID: "sovereign", Kind: itemToggle, Label: "Sovereign L1", On: true, Desc: "subnet-only validators (--sovereign)"},

		// ── PRECOMPILES ────────────────────────────────────────────────────
		{ID: "h_precompiles", Kind: itemHeading, Label: "PRECOMPILES"},
		{ID: "tx_allowlist", Kind: itemToggle, Label: "TxAllowList", On: true, Locked: true, Desc: "network-level tx whitelist — required for compliance"},
		{ID: "warp", Kind: itemToggle, Label: "Warp Messaging", On: true, Desc: "Avalanche Warp cross-chain messaging"},
		{ID: "icm", Kind: itemToggle, Label: "ICM Registry", On: true, Desc: "interchain messaging at genesis (--icm)"},
		{ID: "debug", Kind: itemToggle, Label: "Debug API", On: false, Desc: "enable blockchain debugging (--debug)"},

		// ═══════════════ COMPLIANCE & REGULATION ═══════════════════════════
		{ID: "d_compliance", Kind: itemDivider, Label: "COMPLIANCE & REGULATION"},

		// ── JURISDICTION ───────────────────────────────────────────────────
		{ID: "h_jurisdiction", Kind: itemHeading, Label: "JURISDICTION"},
		{ID: "j_mx", Kind: itemRadio, Group: "jurisdiction", Label: "CNBV — México", On: true, Desc: "Ley Fintech 2018 · FATF presidency 2025-26"},
		{ID: "j_pa", Kind: itemRadio, Group: "jurisdiction", Label: "SMV — Panamá", Desc: "Draft Bill 326 pending · FATF/GAFILAT member"},
		{ID: "j_co", Kind: itemRadio, Group: "jurisdiction", Label: "SFC — Colombia", Desc: "CE 027/2021 · SARLAFT AML compliance"},
		{ID: "j_br", Kind: itemRadio, Group: "jurisdiction", Label: "CVM — Brasil", Desc: "Lei 14.478/2022 · dual CVM+BCB oversight"},
		{ID: "j_custom", Kind: itemRadio, Group: "jurisdiction", Label: "Custom jurisdiction", Desc: "manual configuration"},

		// ── ENFORCEMENT ────────────────────────────────────────────────────
		{ID: "h_enforcement", Kind: itemHeading, Label: "ENFORCEMENT"},
		{ID: "kyc_token", Kind: itemToggle, Label: "KYC at token layer (ERC-3643)", On: true, Desc: "identity-restricted transfers via ONCHAINID"},
		{ID: "kyc_network", Kind: itemToggle, Label: "KYC at network layer (TxAllowList)", On: true, Locked: true, Desc: "only whitelisted addresses can transact"},
		{ID: "evidence", Kind: itemToggle, Label: "On-chain compliance evidence", On: true, Desc: "immutable audit trail via ComplianceRegistry"},
		{ID: "ictt", Kind: itemToggle, Label: "ICTT bridge (L1 → C-Chain)", On: false, Warn: "blocked: FATF Travel Rule", Desc: "outbound bridge to public chain — regulatory risk"},

		// ── CONTRACT TEMPLATES ─────────────────────────────────────────────
		{ID: "h_contracts", Kind: itemHeading, Label: "CONTRACT TEMPLATES"},
		{ID: "c_compliance", Kind: itemToggle, Label: "ComplianceRegistry", On: true, Desc: "jurisdiction + KYC verifier + evidence trail"},
		{ID: "c_trex", Kind: itemToggle, Label: "ERC-3643 / T-REX Token (CEQ)", On: true, Desc: "SEC/MAS-referenced identity-restricted token standard"},
		{ID: "c_dividend", Kind: itemToggle, Label: "DividendDistributor", On: true, Desc: "KYC-verified dividend distribution"},
		{ID: "c_trex_info", Kind: itemInfo, Label: "  T-REX referenced by SEC Chair Atkins (Jul 2025) as compliance model"},

		// ═══════════════ INFRASTRUCTURE ════════════════════════════════════
		{ID: "d_infra", Kind: itemDivider, Label: "INFRASTRUCTURE"},

		// ── DEPLOY TARGET ──────────────────────────────────────────────────
		{ID: "h_target", Kind: itemHeading, Label: "DEPLOY TARGET"},
		{ID: "t_local", Kind: itemRadio, Group: "target", Label: "Local — on-premise devnet", On: true, Desc: "Avalanche network on this machine"},
		{ID: "t_oci", Kind: itemRadio, Group: "target", Label: "OCI — Oracle Cloud VM", Desc: "remote VM in your OCI tenancy"},

		// ── VALIDATORS ─────────────────────────────────────────────────────
		{ID: "h_validators", Kind: itemHeading, Label: "VALIDATORS"},
		{ID: "num_nodes", Kind: itemText, Label: "Nodes", Value: "5", Desc: "local network nodes (--num-nodes)"},
		{ID: "num_bootstrap", Kind: itemText, Label: "Bootstrap", Value: "5", Desc: "bootstrap validators (--num-bootstrap-validators)"},
		{ID: "balance", Kind: itemText, Label: "Balance (AVAX)", Value: "1.0", Desc: "AVAX per validator for P-Chain fees"},

		// ── ACTIONS ────────────────────────────────────────────────────────
		{ID: "d_actions", Kind: itemDivider, Label: "ACTIONS"},
		{ID: "deploy", Kind: itemAction, Label: "▶ Deploy L1", Desc: "terraform init + apply + contract deployment"},
		{ID: "destroy", Kind: itemAction, Label: "✕ Destroy infrastructure", Desc: "terraform destroy + cleanup evidence"},
		{ID: "preview", Kind: itemAction, Label: "◎ Toggle Terraform preview", Desc: "show generated HCL configuration"},
		{ID: "dashboard", Kind: itemAction, Label: "◉ Open operations dashboard", Desc: "explorer, T-REX wallets, contracts, evidence"},
	}
}

// ── Item helpers ──────────────────────────────────────────────────────────────

func wizardGetValue(items []wizardItem, id string) string {
	for _, it := range items {
		if it.ID == id {
			return it.Value
		}
	}
	return ""
}

func wizardIsOn(items []wizardItem, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return it.On
		}
	}
	return false
}

func wizardRadioSelected(items []wizardItem, group string) string {
	for _, it := range items {
		if it.Group == group && it.On {
			return it.ID
		}
	}
	return ""
}

func wizardSelectRadio(items []wizardItem, group, selectedID string) {
	for i := range items {
		if items[i].Group == group {
			items[i].On = items[i].ID == selectedID
		}
	}
}

func wizardJurisdiction(items []wizardItem) string {
	switch wizardRadioSelected(items, "jurisdiction") {
	case "j_mx":
		return "CNBV-MX"
	case "j_pa":
		return "SMV-PA"
	case "j_co":
		return "SFC-CO"
	case "j_br":
		return "CVM-BR"
	default:
		return "CUSTOM"
	}
}

func wizardConsensus(items []wizardItem) string {
	switch wizardRadioSelected(items, "consensus") {
	case "pos_native":
		return "pos-native"
	case "pos_erc20":
		return "pos-erc20"
	default:
		return "poa"
	}
}

// ── Terraform HCL preview ─────────────────────────────────────────────────────

func terraformPreview(items []wizardItem) string {
	val := func(id string) string { return wizardGetValue(items, id) }
	on := func(id string) bool { return wizardIsOn(items, id) }
	consensus := wizardConsensus(items)
	jur := wizardJurisdiction(items)

	var b strings.Builder
	b.WriteString("# Generated by claw1 wizard — " + jur + "\n\n")

	// L1 resource
	b.WriteString("resource \"claw1_l1\" \"main\" {\n")
	b.WriteString(fmt.Sprintf("  name       = %q\n", val("name")))
	b.WriteString(fmt.Sprintf("  chain_id   = %s\n", val("chain_id")))
	if v := val("token"); v != "" && v != "CLAW" {
		b.WriteString(fmt.Sprintf("  token_symbol = %q\n", v))
	}
	b.WriteString(fmt.Sprintf("  consensus  = %q\n", consensus))
	if on("sovereign") {
		b.WriteString("  sovereign  = true\n")
	}
	if on("warp") {
		b.WriteString("  enable_warp = true\n")
	}
	if on("icm") {
		b.WriteString("  enable_icm  = true\n")
	}
	if on("debug") {
		b.WriteString("  enable_debug = true\n")
	}
	if on("prod_defaults") {
		b.WriteString("  production_defaults = true\n")
	}
	if v := val("vm_version"); v != "" && v != "latest" {
		b.WriteString(fmt.Sprintf("  vm_version = %q\n", v))
	}
	b.WriteString("}\n")

	// ComplianceRegistry
	if on("c_compliance") {
		b.WriteString("\nresource \"claw1_contract\" \"compliance\" {\n")
		b.WriteString("  name   = \"ComplianceRegistry\"\n")
		b.WriteString("  source = \"contracts/src/ComplianceRegistry.sol\"\n")
		b.WriteString("  constructor_args = [\n")
		b.WriteString("    claw1_l1.main.chain_id,\n")
		b.WriteString("    \"0x8db97...ewoq\",    # TxAllowList admin\n")
		b.WriteString("    \"0x0000...0000\",     # KYC verifier (pluggable)\n")
		b.WriteString("    \"0\",                 # KYC claim ID\n")
		b.WriteString(fmt.Sprintf("    %q,    # jurisdiction\n", jur))
		b.WriteString("  ]\n")
		b.WriteString("}\n")
	}

	// DividendDistributor
	if on("c_dividend") {
		b.WriteString("\nresource \"claw1_contract\" \"dividends\" {\n")
		b.WriteString("  name   = \"DividendDistributor\"\n")
		b.WriteString("  source = \"contracts/src/DividendDistributor.sol\"\n")
		b.WriteString("  constructor_args = [\n")
		b.WriteString("    \"0x0000...0000\",  # KYC verifier\n")
		b.WriteString("    \"0\",              # KYC claim ID\n")
		b.WriteString("  ]\n")
		b.WriteString("}\n")
	}

	// ERC-3643
	if on("c_trex") {
		b.WriteString("\n# ERC-3643 / T-REX suite deployed via forge script\n")
		b.WriteString("# → IdentityRegistry + ClaimIssuer + CEQ Token\n")
	}

	return b.String()
}
