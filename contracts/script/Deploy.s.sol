// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, console} from "forge-std/Script.sol";
import {DividendDistributor} from "../src/DividendDistributor.sol";

contract Deploy is Script {
    function run() external returns (DividendDistributor) {
        vm.startBroadcast();
        DividendDistributor distributor = new DividendDistributor(address(0), 0);
        console.log("DividendDistributor deployed to:", address(distributor));
        vm.stopBroadcast();
        return distributor;
    }
}
