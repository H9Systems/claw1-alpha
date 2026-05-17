// SPDX-License-Identifier: MIT
// DEPENDENCY: run scripts/ictt-setup.sh before compiling this script.
//
// DeployICTT — deploys ICTT bridge contracts for an on-prem Avalanche devnet.
//
// Due to Foundry's inability to simulate Avalanche Warp precompiles
// (IWarpMessenger at 0x05), this script CANNOT be run with `forge script --broadcast`.
// Instead, the Go orchestrator (run.sh / claw1 TUI) uses `forge create` for each
// contract individually, which bypasses Foundry's local EVM simulation.
//
// This Solidity file exists purely for compilation verification and ABI export.
// The actual deployment flow is:
//   1. forge create TeleporterRegistry (on each chain)
//   2. forge create DemoUSDC (on C-chain, optional)
//   3. forge create ERC20TokenHome (on C-chain)
//   4. forge create ERC20TokenRemote (on L1)
pragma solidity 0.8.18;

// ICTT v1.0.0 imports — available after scripts/ictt-setup.sh
import {ERC20TokenHome} from "@ictt/TokenHome/ERC20TokenHome.sol";
import {ERC20TokenRemote} from "@ictt/TokenRemote/ERC20TokenRemote.sol";
import {TokenRemoteSettings} from "@ictt/TokenRemote/interfaces/ITokenRemote.sol";
import {TeleporterRegistry, ProtocolRegistryEntry} from "@teleporter/upgrades/TeleporterRegistry.sol";

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

// DeployICTT is a placeholder that simply verifies compilation.
// The actual deployment is orchestrated by run.sh / claw1 TUI using forge create.
contract DeployICTT {
    // Intentionally empty — see script comments above.
}