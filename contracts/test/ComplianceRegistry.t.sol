// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {ComplianceRegistry} from "../src/ComplianceRegistry.sol";

contract ComplianceRegistryTest is Test {
    ComplianceRegistry registry;

    uint256 constant CHAIN_ID = 99999;
    address constant TX_ALLOWLIST_ADMIN = 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC;
    address constant KYC_VERIFIER = address(0);
    uint256 constant KYC_CLAIM_ID = 0;
    string constant JURISDICTION = "demo";

    function setUp() public {
        registry = new ComplianceRegistry(
            CHAIN_ID,
            TX_ALLOWLIST_ADMIN,
            KYC_VERIFIER,
            KYC_CLAIM_ID,
            JURISDICTION
        );
    }

    function test_constructor_stores_all_values() public view {
        ComplianceRegistry.Config memory cfg = registry.getConfig();
        assertEq(cfg.chainId, CHAIN_ID);
        assertEq(cfg.txAllowListAdmin, TX_ALLOWLIST_ADMIN);
        assertEq(cfg.kycVerifier, KYC_VERIFIER);
        assertEq(cfg.kycClaimId, KYC_CLAIM_ID);
        assertEq(cfg.jurisdiction, JURISDICTION);
        assertGt(cfg.configuredAt, 0);
    }

    function test_config_recorded_event_emitted() public {
        vm.expectEmit(true, false, false, true);
        emit ComplianceRegistry.ConfigRecorded(
            CHAIN_ID,
            TX_ALLOWLIST_ADMIN,
            KYC_VERIFIER,
            JURISDICTION,
            block.timestamp
        );
        new ComplianceRegistry(CHAIN_ID, TX_ALLOWLIST_ADMIN, KYC_VERIFIER, KYC_CLAIM_ID, JURISDICTION);
    }

    function test_record_allowlist_change_emits_event() public {
        address shareholder = makeAddr("shareholder");
        uint8 role = 1;

        vm.expectEmit(true, true, false, true);
        emit ComplianceRegistry.AllowlistChanged(shareholder, role, address(this), block.timestamp);
        registry.recordAllowlistChange(shareholder, role);
    }

    function test_non_owner_record_reverts() public {
        address attacker = makeAddr("attacker");
        vm.prank(attacker);
        vm.expectRevert("Not owner");
        registry.recordAllowlistChange(makeAddr("target"), 1);
    }
}
