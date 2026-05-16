// SPDX-License-Identifier: MIT
// DEPENDENCY: run scripts/ictt-setup.sh before compiling this script.
pragma solidity 0.8.18;

import {Script, console} from "forge-std/Script.sol";

// ICTT v2.0.0 imports — available after scripts/ictt-setup.sh
import {ERC20TokenHome} from "@ictt/TokenHome/ERC20TokenHome.sol";
import {ERC20TokenRemote} from "@ictt/TokenRemote/ERC20TokenRemote.sol";
import {TokenHomeSettings} from "@ictt/interfaces/ITokenHome.sol";
import {TokenRemoteSettings} from "@ictt/interfaces/ITokenRemote.sol";

// Minimal ERC20 for demo token deployment when no source token is supplied
import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract DemoUSDC is ERC20 {
    uint8 private _decimals;
    constructor(string memory name, string memory symbol, uint8 dec)
        ERC20(name, symbol)
    {
        _decimals = dec;
        _mint(msg.sender, 10_000_000 * 10 ** dec); // 10M demo tokens
    }
    function decimals() public view override returns (uint8) { return _decimals; }
}

// ─────────────────────────────────────────────────────────────────────────────
// DeployICTT — deploys ERC20TokenHome on C-chain + ERC20TokenRemote on L1.
//
// Required env vars:
//   DEPLOYER_PRIVATE_KEY      hex private key (no 0x prefix)
//   C_CHAIN_RPC_URL           Fuji C-chain RPC (https://api.avax-test.network/ext/bc/C/rpc)
//   C_CHAIN_BLOCKCHAIN_ID     Fuji C-chain blockchainID as bytes32 hex
//   L1_RPC_URL                L1 RPC (from network.json — tunneled for OCI)
//   L1_TELEPORTER_REGISTRY    Teleporter Registry on the L1
//
// Optional env vars:
//   SOURCE_TOKEN_ADDRESS      ERC20 on C-chain to bridge (deploys DemoUSDC if unset)
//   C_CHAIN_TELEPORTER_REGISTRY  defaults to Fuji canonical address
// ─────────────────────────────────────────────────────────────────────────────
contract DeployICTT is Script {
    // Fuji C-chain Teleporter Registry (canonical, v1.0.0+)
    address constant FUJI_TELEPORTER_REGISTRY = 0xF86Cb19Ad8405AEFa7d09C778215D2Cb6eBfB228;

    function run() external {
        uint256 deployerKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        address deployer    = vm.addr(deployerKey);

        string memory cChainRPC    = vm.envString("C_CHAIN_RPC_URL");
        string memory l1RPC        = vm.envString("L1_RPC_URL");
        bytes32 cChainBlockchainID = vm.envBytes32("C_CHAIN_BLOCKCHAIN_ID");

        address cChainRegistry = vm.envOr("C_CHAIN_TELEPORTER_REGISTRY", FUJI_TELEPORTER_REGISTRY);
        address l1Registry     = vm.envAddress("L1_TELEPORTER_REGISTRY");

        // ── Step 1: Deploy TokenHome on C-chain ──────────────────────────────

        uint256 cFork = vm.createFork(cChainRPC);
        vm.selectFork(cFork);

        // Deploy a demo ERC20 on C-chain if no real token address is supplied.
        address sourceToken = vm.envOr("SOURCE_TOKEN_ADDRESS", address(0));
        vm.startBroadcast(deployerKey);
        if (sourceToken == address(0)) {
            DemoUSDC demo = new DemoUSDC("Demo USDC", "dUSDC", 6);
            sourceToken = address(demo);
            console.log("DemoUSDC deployed at:", sourceToken);
        }

        TokenHomeSettings memory homeSettings = TokenHomeSettings({
            teleporterRegistryAddress: cChainRegistry,
            teleporterManager:        deployer,
            minTeleporterVersion:     1
        });
        ERC20TokenHome tokenHome = new ERC20TokenHome(homeSettings, 6);
        vm.stopBroadcast();

        console.log("ICTT_TOKEN_HOME:        ", address(tokenHome));
        console.log("ICTT_SOURCE_TOKEN:      ", sourceToken);

        // ── Step 2: Deploy TokenRemote on L1 ─────────────────────────────────

        uint256 l1Fork = vm.createFork(l1RPC);
        vm.selectFork(l1Fork);

        TokenRemoteSettings memory remoteSettings = TokenRemoteSettings({
            teleporterRegistryAddress: l1Registry,
            teleporterManager:        deployer,
            minTeleporterVersion:     1,
            tokenHomeAddress:         address(tokenHome),
            tokenHomeBlockchainID:    cChainBlockchainID,
            homeTokenDecimals:        6
        });
        vm.startBroadcast(deployerKey);
        ERC20TokenRemote tokenRemote = new ERC20TokenRemote(
            remoteSettings,
            "Bridged USDC",
            "bUSDC",
            6
        );
        vm.stopBroadcast();

        console.log("ICTT_TOKEN_REMOTE:      ", address(tokenRemote));

        // ── Summary ───────────────────────────────────────────────────────────
        console.log("");
        console.log("ICTT bridge wired:");
        console.log("  C-chain source token:", sourceToken);
        console.log("  TokenHome (C-chain): ", address(tokenHome));
        console.log("  TokenRemote (L1):    ", address(tokenRemote));
        console.log("");
        console.log("Next: register the remote with the home via Teleporter send.");
    }
}
