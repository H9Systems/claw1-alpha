// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract ComplianceRegistry {
    struct Config {
        uint256 chainId;
        address txAllowListAdmin;
        address kycVerifier;
        uint256 kycClaimId;
        string jurisdiction;
        uint256 configuredAt;
    }

    Config private _config;
    address public owner;

    event ConfigRecorded(
        uint256 indexed chainId,
        address txAllowListAdmin,
        address kycVerifier,
        string jurisdiction,
        uint256 timestamp
    );

    event AllowlistChanged(
        address indexed who,
        uint8 role,
        address indexed changedBy,
        uint256 timestamp
    );

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    constructor(
        uint256 chainId,
        address txAllowListAdmin,
        address kycVerifier,
        uint256 kycClaimId,
        string memory jurisdiction
    ) {
        owner = msg.sender;
        _config = Config({
            chainId: chainId,
            txAllowListAdmin: txAllowListAdmin,
            kycVerifier: kycVerifier,
            kycClaimId: kycClaimId,
            jurisdiction: jurisdiction,
            configuredAt: block.timestamp
        });
        emit ConfigRecorded(chainId, txAllowListAdmin, kycVerifier, jurisdiction, block.timestamp);
    }

    function recordAllowlistChange(address who, uint8 role) external onlyOwner {
        emit AllowlistChanged(who, role, msg.sender, block.timestamp);
    }

    function getConfig() external view returns (Config memory) {
        return _config;
    }
}
