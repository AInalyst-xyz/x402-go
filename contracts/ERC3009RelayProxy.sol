// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title ERC3009 Relay Proxy
 * @notice Relay proxy for USDC transferWithAuthorization with fee collection
 * @dev Sits between relayer and USDC contract to add business logic
 */

interface IERC20WithAuth {
    function transferWithAuthorization(
        address from,
        address to,
        uint256 value,
        uint256 validAfter,
        uint256 validBefore,
        bytes32 nonce,
        bytes calldata signature
    ) external;

    function transfer(address to, uint256 value) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

contract ERC3009RelayProxy {
    // USDC contract address (Base mainnet)
    IERC20WithAuth public immutable usdc;

    // Fee configuration
    address public feeCollector;
    uint256 public relayFeeBps; // Fee in basis points (100 = 1%)

    // Access control
    mapping(address => bool) public authorizedRelayers;
    address public owner;

    // Nonce tracking (redundant with USDC, but useful for events)
    mapping(address => mapping(bytes32 => bool)) public processedNonces;

    // Events
    event RelayExecuted(
        address indexed from,
        address indexed to,
        uint256 value,
        uint256 fee,
        bytes32 indexed nonce,
        address relayer
    );

    event FeeCollectorUpdated(address indexed oldCollector, address indexed newCollector);
    event RelayFeeUpdated(uint256 oldFee, uint256 newFee);
    event RelayerAuthorized(address indexed relayer, bool authorized);

    // Errors
    error UnauthorizedRelayer();
    error NonceAlreadyProcessed();
    error InvalidFeeCollector();
    error InvalidFeeBps();
    error Unauthorized();

    modifier onlyOwner() {
        if (msg.sender != owner) revert Unauthorized();
        _;
    }

    modifier onlyAuthorizedRelayer() {
        if (!authorizedRelayers[msg.sender]) revert UnauthorizedRelayer();
        _;
    }

    constructor(
        address _usdc,
        address _feeCollector,
        uint256 _relayFeeBps
    ) {
        require(_usdc != address(0), "Invalid USDC address");
        require(_feeCollector != address(0), "Invalid fee collector");
        require(_relayFeeBps <= 1000, "Fee too high"); // Max 10%

        usdc = IERC20WithAuth(_usdc);
        feeCollector = _feeCollector;
        relayFeeBps = _relayFeeBps;
        owner = msg.sender;

        // Authorize deployer as first relayer
        authorizedRelayers[msg.sender] = true;
    }

    /**
     * @notice Relay a transfer with fee deduction
     * @dev Only authorized relayers can call this
     */
    function relay(
        address from,
        address to,
        uint256 value,
        uint256 validAfter,
        uint256 validBefore,
        bytes32 nonce,
        bytes calldata signature
    ) external onlyAuthorizedRelayer returns (bool) {
        // Check nonce not already processed (redundant but safer)
        if (processedNonces[from][nonce]) revert NonceAlreadyProcessed();

        // Calculate fee
        uint256 fee = (value * relayFeeBps) / 10000;
        uint256 netAmount = value - fee;

        // Call USDC transferWithAuthorization
        // This transfers full amount to this contract
        usdc.transferWithAuthorization(
            from,
            address(this), // Proxy receives funds
            value,
            validAfter,
            validBefore,
            nonce,
            signature
        );

        // Split payment: merchant + fee
        usdc.transfer(to, netAmount);
        if (fee > 0) {
            usdc.transfer(feeCollector, fee);
        }

        // Mark nonce as processed
        processedNonces[from][nonce] = true;

        // Emit event
        emit RelayExecuted(from, to, value, fee, nonce, msg.sender);

        return true;
    }

    /**
     * @notice Batch relay multiple transfers
     * @dev Saves gas by batching multiple transfers
     */
    function batchRelay(
        address[] calldata froms,
        address[] calldata tos,
        uint256[] calldata values,
        uint256[] calldata validAfters,
        uint256[] calldata validBefores,
        bytes32[] calldata nonces,
        bytes[] calldata signatures
    ) external onlyAuthorizedRelayer returns (bool) {
        require(froms.length == tos.length, "Length mismatch");
        require(froms.length == values.length, "Length mismatch");
        require(froms.length == nonces.length, "Length mismatch");
        require(froms.length == signatures.length, "Length mismatch");

        for (uint256 i = 0; i < froms.length; i++) {
            relay(
                froms[i],
                tos[i],
                values[i],
                validAfters[i],
                validBefores[i],
                nonces[i],
                signatures[i]
            );
        }

        return true;
    }

    /**
     * @notice Update fee collector address
     */
    function setFeeCollector(address _feeCollector) external onlyOwner {
        if (_feeCollector == address(0)) revert InvalidFeeCollector();

        address oldCollector = feeCollector;
        feeCollector = _feeCollector;

        emit FeeCollectorUpdated(oldCollector, _feeCollector);
    }

    /**
     * @notice Update relay fee
     */
    function setRelayFee(uint256 _relayFeeBps) external onlyOwner {
        if (_relayFeeBps > 1000) revert InvalidFeeBps(); // Max 10%

        uint256 oldFee = relayFeeBps;
        relayFeeBps = _relayFeeBps;

        emit RelayFeeUpdated(oldFee, _relayFeeBps);
    }

    /**
     * @notice Authorize or deauthorize a relayer
     */
    function setRelayerAuthorization(address relayer, bool authorized) external onlyOwner {
        authorizedRelayers[relayer] = authorized;
        emit RelayerAuthorized(relayer, authorized);
    }

    /**
     * @notice Transfer ownership
     */
    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "Invalid owner");
        owner = newOwner;
    }

    /**
     * @notice Emergency withdrawal (in case funds get stuck)
     */
    function emergencyWithdraw(address token, address to, uint256 amount) external onlyOwner {
        IERC20WithAuth(token).transfer(to, amount);
    }
}
