// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test} from "forge-std/Test.sol";
import {DividendDistributor} from "../src/DividendDistributor.sol";

contract DividendDistributorTest is Test {
    DividendDistributor distributor;

    address owner = address(this);
    address alice = makeAddr("alice");
    address bob   = makeAddr("bob");
    address carol = makeAddr("carol");

    function setUp() public {
        distributor = new DividendDistributor(address(0), 0);
    }

    function test_register_and_distribute() public {
        // 3 shareholders: 30% / 30% / 40%
        distributor.registerShareholder(alice, "Alice Morales", 3000);
        distributor.registerShareholder(bob,   "Bob Ramirez",   3000);
        distributor.registerShareholder(carol, "Carol Vega",    4000);

        assertEq(distributor.getShareholderCount(), 3);
        assertEq(distributor.totalBps(), 10000);

        uint256 amount = 1 ether;
        vm.deal(owner, amount);
        distributor.distribute{value: amount}();

        assertEq(alice.balance, 0.30 ether);
        assertEq(bob.balance,   0.30 ether);
        assertEq(carol.balance, 0.40 ether);
    }

    function test_distribute_without_shareholders() public {
        vm.deal(owner, 1 ether);
        vm.expectRevert("No shareholders");
        distributor.distribute{value: 1 ether}();
    }

    function test_non_owner_revert() public {
        distributor.registerShareholder(alice, "Alice Morales", 10000);

        address attacker = makeAddr("attacker");
        vm.deal(attacker, 1 ether);
        vm.prank(attacker);
        vm.expectRevert("Not owner");
        distributor.distribute{value: 1 ether}();
    }

    function test_distribute_no_value_revert() public {
        distributor.registerShareholder(alice, "Alice Morales", 10000);
        vm.expectRevert("No value sent");
        distributor.distribute{value: 0}();
    }

    function test_bps_not_10000_revert() public {
        // Partial allocation: 90% total — distribute must revert
        distributor.registerShareholder(alice, "Alice Morales", 6000);
        distributor.registerShareholder(bob,   "Bob Ramirez",   3000);
        assertEq(distributor.totalBps(), 9000);

        vm.deal(owner, 1 ether);
        vm.expectRevert("Shares must sum to 100%");
        distributor.distribute{value: 1 ether}();
    }

    function test_update_existing_shareholder() public {
        // First registration
        distributor.registerShareholder(alice, "Alice Morales", 5000);
        distributor.registerShareholder(bob,   "Bob Ramirez",   5000);
        assertEq(distributor.totalBps(), 10000);
        assertEq(distributor.getShareholderCount(), 2);

        // Update alice's share — list length must not grow
        distributor.registerShareholder(alice, "Alice Morales Updated", 4000);
        assertEq(distributor.getShareholderCount(), 2, "No duplicate added");
        assertEq(distributor.totalBps(), 9000, "totalBps adjusts correctly");

        // Fix bob to make whole again
        distributor.registerShareholder(bob, "Bob Ramirez", 6000);
        assertEq(distributor.totalBps(), 10000);

        vm.deal(owner, 1 ether);
        distributor.distribute{value: 1 ether}();
        assertEq(alice.balance, 0.40 ether);
        assertEq(bob.balance,   0.60 ether);
    }

    function test_distribute_twice() public {
        distributor.registerShareholder(alice, "Alice Morales", 5000);
        distributor.registerShareholder(bob,   "Bob Ramirez",   5000);

        // First distribution
        vm.deal(owner, 2 ether);
        distributor.distribute{value: 1 ether}();
        assertEq(alice.balance, 0.50 ether);
        assertEq(bob.balance,   0.50 ether);

        // Second distribution — state resets correctly
        distributor.distribute{value: 1 ether}();
        assertEq(alice.balance, 1.00 ether);
        assertEq(bob.balance,   1.00 ether);
    }
}
