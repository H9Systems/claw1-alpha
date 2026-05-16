// SPDX-License-Identifier: MIT
pragma solidity 0.8.17;

import {Script, console} from "forge-std/Script.sol";

import {ClaimTopicsRegistry} from "@T-REX/registry/implementation/ClaimTopicsRegistry.sol";
import {TrustedIssuersRegistry} from "@T-REX/registry/implementation/TrustedIssuersRegistry.sol";
import {IdentityRegistryStorage} from "@T-REX/registry/implementation/IdentityRegistryStorage.sol";
import {IdentityRegistry} from "@T-REX/registry/implementation/IdentityRegistry.sol";
import {ModularCompliance} from "@T-REX/compliance/modular/ModularCompliance.sol";
import {Token} from "@T-REX/token/Token.sol";

import {ClaimIssuer} from "@onchain-id/solidity/contracts/ClaimIssuer.sol";
import {Identity} from "@onchain-id/solidity/contracts/Identity.sol";

contract DeployERC3643 is Script {
    uint256 constant KYC_CLAIM_TOPIC = 1;
    uint16 constant MEXICO_COUNTRY_CODE = 484;

    // Deployed addresses — populated during run() and used in helpers
    ClaimTopicsRegistry public ctr;
    TrustedIssuersRegistry public tir;
    IdentityRegistryStorage public irs;
    IdentityRegistry public ir;
    ModularCompliance public compliance;
    Token public token;
    ClaimIssuer public issuerIdentity;

    function run() external {
        uint256 deployerKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        address deployer = vm.addr(deployerKey);

        vm.startBroadcast(deployerKey);
        _deployCoreContracts();
        _wireContracts(deployer);
        _setupKYC(deployerKey, deployer);
        vm.stopBroadcast();

        _printSummary(deployer);
    }

    function _deployCoreContracts() internal {
        ctr = new ClaimTopicsRegistry();
        ctr.init();

        tir = new TrustedIssuersRegistry();
        tir.init();

        irs = new IdentityRegistryStorage();
        irs.init();

        ir = new IdentityRegistry();
        ir.init(address(tir), address(ctr), address(irs));

        compliance = new ModularCompliance();
        compliance.init();

        token = new Token();
        token.init(address(ir), address(compliance), "Claw1 Equity Token", "CEQ", 18, address(0));
    }

    function _wireContracts(address deployer) internal {
        irs.bindIdentityRegistry(address(ir));
        compliance.bindToken(address(token));
        token.addAgent(deployer);
        ir.addAgent(deployer);
    }

    function _setupKYC(uint256 deployerKey, address deployer) internal {
        ctr.addClaimTopic(KYC_CLAIM_TOPIC);

        issuerIdentity = new ClaimIssuer(deployer);
        uint256[] memory topics = new uint256[](1);
        topics[0] = KYC_CLAIM_TOPIC;
        tir.addTrustedIssuer(issuerIdentity, topics);

        _registerDemoInvestor(deployerKey, deployer);
    }

    function _registerDemoInvestor(uint256 deployerKey, address deployer) internal {
        address investor = vm.envOr("DEMO_INVESTOR_ADDRESS", deployer);

        Identity investorIdentity = new Identity(investor, false);

        // ONCHAINID isClaimValid computes: keccak256(abi.encode(identityAddr, topic, data))
        // then prefixes it for eth_sign. We must sign the same hash.
        bytes memory claimData = "";
        bytes32 dataHash = keccak256(abi.encode(address(investorIdentity), KYC_CLAIM_TOPIC, claimData));
        bytes32 prefixedHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", dataHash));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(deployerKey, prefixedHash);
        investorIdentity.addClaim(KYC_CLAIM_TOPIC, 1, address(issuerIdentity), abi.encodePacked(r, s, v), claimData, "");

        ir.registerIdentity(investor, investorIdentity, MEXICO_COUNTRY_CODE);
        token.mint(investor, 1_000_000 * 10 ** 18);
    }

    function _printSummary(address deployer) internal view {
        console.log("=== ERC-3643 Deployment Complete ===");
        console.log("ClaimTopicsRegistry:    ", address(ctr));
        console.log("TrustedIssuersRegistry: ", address(tir));
        console.log("IdentityRegistryStorage:", address(irs));
        console.log("IdentityRegistry:       ", address(ir));
        console.log("ModularCompliance:      ", address(compliance));
        console.log("Token (CEQ):            ", address(token));
        console.log("ClaimIssuer (KYC auth): ", address(issuerIdentity));
        console.log("Deployer / investor:    ", deployer);
        console.log("Minted:                  1,000,000 CEQ");
        console.log("");
        console.log("Verify KYC gate (should return false for non-KYC addr):");
        console.log("  cast call <IdentityRegistry> \"isVerified(address)\" <addr> --rpc-url $L1_RPC_URL");
        console.log("");
        console.log("Verify balance:");
        console.log("  cast call <Token> \"balanceOf(address)\" <investor> --rpc-url $L1_RPC_URL");
    }
}
