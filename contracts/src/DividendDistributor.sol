// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IKYCVerifier {
    function ifVerified(address claimer, uint256 claimId) external view returns (bool);
}

contract DividendDistributor {
    address public owner;
    uint16 public totalBps;
    IKYCVerifier public immutable kycVerifier;
    uint256 public immutable kycClaimId;

    struct Shareholder {
        string name;
        uint16 bps; // basis points: 10000 = 100%
    }

    mapping(address => Shareholder) public shareholders;
    address[] public shareholderList;

    event ShareholderRegistered(address indexed addr, string name, uint16 bps);
    event DividendDistributed(address indexed to, uint256 amount, string shareholderName);
    event DistributionCompleted(uint256 totalAmount, uint256 shareholderCount);

    modifier onlyOwner() {
        require(msg.sender == owner, "Not owner");
        _;
    }

    constructor(address _kycVerifier, uint256 _kycClaimId) {
        owner = msg.sender;
        kycVerifier = IKYCVerifier(_kycVerifier);
        kycClaimId = _kycClaimId;
    }

    function registerShareholder(address addr, string calldata name, uint16 bps) external onlyOwner {
        if (address(kycVerifier) != address(0)) {
            require(kycVerifier.ifVerified(addr, kycClaimId), "KYC not verified");
        }
        uint16 oldBps = shareholders[addr].bps;
        if (oldBps == 0) {
            shareholderList.push(addr);
        }
        totalBps = totalBps - oldBps + bps;
        shareholders[addr] = Shareholder(name, bps);
        emit ShareholderRegistered(addr, name, bps);
    }

    function distribute() external payable onlyOwner {
        require(msg.value > 0, "No value sent");
        require(shareholderList.length > 0, "No shareholders");
        require(totalBps == 10000, "Shares must sum to 100%");

        uint256 total = msg.value;
        for (uint256 i = 0; i < shareholderList.length; i++) {
            address addr = shareholderList[i];
            Shareholder memory s = shareholders[addr];
            uint256 payout = (total * s.bps) / 10000;
            if (payout > 0) {
                payable(addr).transfer(payout);
                emit DividendDistributed(addr, payout, s.name);
            }
        }
        emit DistributionCompleted(total, shareholderList.length);
    }

    function getShareholderCount() external view returns (uint256) {
        return shareholderList.length;
    }
}
