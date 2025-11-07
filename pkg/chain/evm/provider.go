package evm

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	x402types "github.com/x402-rs/x402-go/pkg/types"
)

// Provider handles EVM-based payment verification and settlement
type Provider struct {
	client          *ethclient.Client
	chainID         *big.Int
	signers         []*ecdsa.PrivateKey
	signerAddresses []common.Address
	signerIndex     atomic.Uint64
	usdcABI         abi.ABI
	validatorABI    abi.ABI
	network         x402types.Network
	nonceStore      *NonceStore // Tracks used ERC-3009 nonces to prevent replay
}

// NewProvider creates a new EVM provider
func NewProvider(rpcURL string, chainID *big.Int, network x402types.Network, privateKeys []string) (*Provider, error) {
	// Create RPC client with timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second, // Prevent indefinite hangs on RPC calls
	}
	rpcClient, err := rpc.DialHTTPWithClient(rpcURL, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	client := ethclient.NewClient(rpcClient)

	// Parse private keys
	var signers []*ecdsa.PrivateKey
	var addresses []common.Address
	for _, keyHex := range privateKeys {
		keyHex = strings.TrimPrefix(keyHex, "0x")
		privateKey, err := crypto.HexToECDSA(keyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
		signers = append(signers, privateKey)

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("error casting public key to ECDSA")
		}
		address := crypto.PubkeyToAddress(*publicKeyECDSA)
		addresses = append(addresses, address)
	}

	// Load ABIs (embedded as strings for simplicity, or load from file)
	usdcABI, err := loadUSDABI()
	if err != nil {
		return nil, fmt.Errorf("failed to load USDC ABI: %w", err)
	}

	validatorABI, err := loadValidatorABI()
	if err != nil {
		return nil, fmt.Errorf("failed to load Validator ABI: %w", err)
	}

	return &Provider{
		client:          client,
		chainID:         chainID,
		signers:         signers,
		signerAddresses: addresses,
		usdcABI:         usdcABI,
		validatorABI:    validatorABI,
		network:         network,
		nonceStore:      NewNonceStore(),
	}, nil
}

// Verify validates an EVM payment without submitting a transaction
func (p *Provider) Verify(ctx context.Context, request *x402types.VerifyRequest) (*x402types.VerifyResponse, error) {
	payload := request.PaymentPayload.Payload
	requirements := &request.PaymentRequirements

	// Parse authorization
	auth := &payload.Authorization

	// Validate receiver address
	expectedReceiver := requirements.PayTo
	actualReceiver := auth.To.Hex()
	if !strings.EqualFold(expectedReceiver, actualReceiver) {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewReceiverMismatchError(expectedReceiver, actualReceiver, payer)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// Validate asset is whitelisted USDC (mainnet only)
	// Whitelist of accepted USDC mainnet addresses (case-insensitive)
	whitelistedAssets := map[string]bool{
		"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": true, // USDC on Base mainnet
		// Add more mainnet USDC addresses here as needed (use lowercase):
		// "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": true, // USDC on Ethereum mainnet
		// "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359": true, // USDC on Polygon mainnet
		// "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e": true, // USDC on Avalanche mainnet
	}

	assetAddr := strings.ToLower(requirements.Asset.Hex())
	if !whitelistedAssets[assetAddr] {
		payer := x402types.NewEvmAddress(auth.From)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("unsupported asset: %s (only whitelisted USDC contracts are accepted)", requirements.Asset.Hex()),
			Payer:   &payer,
		}, nil
	}

	// Validate timing
	now := x402types.UnixTimestamp()
	validAfter, err := strconv.ParseUint(auth.ValidAfter, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validAfter: %w", err)
	}
	validBefore, err := strconv.ParseUint(auth.ValidBefore, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid validBefore: %w", err)
	}

	// Validate validBefore > validAfter (prevents integer underflow)
	if validBefore <= validAfter {
		payer := x402types.NewEvmAddress(auth.From)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("invalid validity window: validBefore (%d) must be greater than validAfter (%d)", validBefore, validAfter),
			Payer:   &payer,
		}, nil
	}

	if now < validAfter {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewInvalidTimingError(payer, fmt.Sprintf("payment not yet valid (validAfter: %s, now: %d)", auth.ValidAfter, now))
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}
	if now >= validBefore {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewInvalidTimingError(payer, fmt.Sprintf("payment expired (validBefore: %s, now: %d)", auth.ValidBefore, now))
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// Validate timeout window doesn't exceed MaxTimeoutSeconds
	if requirements.MaxTimeoutSeconds > 0 {
		timeoutWindow := validBefore - validAfter
		maxTimeout := uint64(requirements.MaxTimeoutSeconds)
		if timeoutWindow > maxTimeout {
			payer := x402types.NewEvmAddress(auth.From)
			return &x402types.VerifyResponse{
				IsValid: false,
				Reason:  fmt.Sprintf("payment validity window too long: %d seconds (max allowed: %d seconds)", timeoutWindow, maxTimeout),
				Payer:   &payer,
			}, nil
		}
	}

	// Check for nonce replay
	fromAddress := auth.From.Hex()
	if p.nonceStore.IsNonceUsed(fromAddress, auth.Nonce) {
		payer := x402types.NewEvmAddress(auth.From)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  "nonce already used (replay attack detected)",
			Payer:   &payer,
		}, nil
	}

	// Parse amount
	value := new(big.Int)
	value, ok := value.SetString(auth.Value, 10)
	if !ok {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewDecodingError("invalid value format")
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// Parse required amount
	requiredAmount := new(big.Int)
	requiredAmount, ok = requiredAmount.SetString(requirements.MaxAmountRequired, 10)
	if !ok {
		return nil, x402types.NewDecodingError("invalid required amount")
	}

	// Check amount sufficiency
	if value.Cmp(requiredAmount) < 0 {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewInsufficientValueError(payer)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// Verify EIP-712 signature
	valid, err := p.verifySignature(ctx, auth, payload.Signature, requirements.Asset.Hex())
	if err != nil {
		payer := x402types.NewEvmAddress(auth.From)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("signature verification failed: %v", err),
			Payer:   &payer,
		}, nil
	}
	if !valid {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewInvalidSignatureError(payer, "signature verification failed")
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// Check balance
	tokenAddr := requirements.Asset
	balance, err := p.getBalance(ctx, tokenAddr, auth.From)
	if err != nil {
		log.Printf("evm.Verify: balance check failed err=%v", err)
		payer := x402types.NewEvmAddress(auth.From)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  fmt.Sprintf("balance check failed: %v", err),
			Payer:   &payer,
		}, nil
	}

	if balance.Cmp(value) < 0 {
		payer := x402types.NewEvmAddress(auth.From)
		err := x402types.NewInsufficientFundsError(payer)
		return &x402types.VerifyResponse{
			IsValid: false,
			Reason:  err.Message,
			Payer:   &payer,
		}, nil
	}

	// All checks passed
	payer := x402types.NewEvmAddress(auth.From)
	return &x402types.VerifyResponse{
		IsValid: true,
		Payer:   &payer,
	}, nil
}

// Settle executes an EVM payment on-chain
func (p *Provider) Settle(ctx context.Context, request *x402types.SettleRequest) (*x402types.SettleResponse, error) {
	// First verify
	verifyReq := &x402types.VerifyRequest{
		PaymentPayload:      request.PaymentPayload,
		PaymentRequirements: request.PaymentRequirements,
	}
	verifyResp, err := p.Verify(ctx, verifyReq)
	if err != nil {
		return &x402types.SettleResponse{
			Success: false,
			Error:   fmt.Sprintf("verification failed: %v", err),
		}, nil
	}
	if !verifyResp.IsValid {
		return &x402types.SettleResponse{
			Success: false,
			Error:   verifyResp.Reason,
		}, nil
	}

	// Get payload
	payload := request.PaymentPayload.Payload
	auth := &payload.Authorization

	// Select signer (round-robin)
	signerIdx := int(p.signerIndex.Add(1) % uint64(len(p.signers)))
	signer := p.signers[signerIdx]

	// Create transaction
	tokenAddr := request.PaymentRequirements.Asset

	// Parse nonce
	nonceHex := strings.TrimPrefix(auth.Nonce, "0x")
	nonceBytes, err := hex.DecodeString(nonceHex)
	if err != nil {
		return &x402types.SettleResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid nonce: %v", err),
		}, nil
	}
	var nonce32 [32]byte
	copy(nonce32[:], nonceBytes)

	// Parse signature
	sigHex := strings.TrimPrefix(payload.Signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return &x402types.SettleResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid signature: %v", err),
		}, nil
	}

	// Parse value
	value := new(big.Int)
	value, ok := value.SetString(auth.Value, 10)
	if !ok {
		return &x402types.SettleResponse{
			Success: false,
			Error:   "invalid value",
		}, nil
	}

	validAfter, ok := new(big.Int).SetString(auth.ValidAfter, 10)
	if !ok {
		return &x402types.SettleResponse{
			Success: false,
			Error:   "invalid validAfter",
		}, nil
	}
	validBefore, ok := new(big.Int).SetString(auth.ValidBefore, 10)
	if !ok {
		return &x402types.SettleResponse{
			Success: false,
			Error:   "invalid validBefore",
		}, nil
	}

	// Call transferWithAuthorization
	tx, err := p.transferWithAuthorization(
		ctx,
		signer,
		tokenAddr,
		auth.From,
		auth.To,
		value,
		validAfter,
		validBefore,
		nonce32,
		sigBytes,
	)
	if err != nil {
		return &x402types.SettleResponse{
			Success: false,
			Error:   fmt.Sprintf("transaction failed: %v", err),
		}, nil
	}

	// Wait for receipt
	receipt, err := bind.WaitMined(ctx, p.client, tx)
	if err != nil {
		return &x402types.SettleResponse{
			Success: false,
			Error:   fmt.Sprintf("waiting for tx failed: %v", err),
		}, nil
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return &x402types.SettleResponse{
			Success: false,
			Error:   "transaction reverted",
		}, nil
	}

	// Mark nonce as used after successful settlement
	fromAddress := auth.From.Hex()
	p.nonceStore.MarkNonceUsed(fromAddress, auth.Nonce, validBefore.Int64())

	return &x402types.SettleResponse{
		Success: true,
		TransactionHash: &x402types.TransactionHash{
			Type: "evm",
			Hash: tx.Hash().Hex(),
		},
	}, nil
}

// verifySignature validates the EIP-712 signature
func (p *Provider) verifySignature(ctx context.Context, auth *x402types.ExactEvmPayloadAuthorization, signature, tokenAddress string) (bool, error) {
	// Create EIP-712 typed data
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"TransferWithAuthorization": []apitypes.Type{
				{Name: "from", Type: "address"},
				{Name: "to", Type: "address"},
				{Name: "value", Type: "uint256"},
				{Name: "validAfter", Type: "uint256"},
				{Name: "validBefore", Type: "uint256"},
				{Name: "nonce", Type: "bytes32"},
			},
		},
		PrimaryType: "TransferWithAuthorization",
		Domain: apitypes.TypedDataDomain{
			Name:              "USD Coin",
			Version:           "2",
			ChainId:           (*math.HexOrDecimal256)(p.chainID),
			VerifyingContract: tokenAddress,
		},
		Message: apitypes.TypedDataMessage{
			"from":        auth.From.Hex(),
			"to":          auth.To.Hex(),
			"value":       auth.Value,
			"validAfter":  auth.ValidAfter,
			"validBefore": auth.ValidBefore,
			"nonce":       auth.Nonce,
		},
	}

	// Hash the typed data
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return false, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return false, fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256Hash(rawData)

	// Parse signature
	sigHex := strings.TrimPrefix(signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %w", err)
	}

	if len(sigBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(sigBytes))
	}

	// Adjust V value
	if sigBytes[64] >= 27 {
		sigBytes[64] -= 27
	}

	// Recover public key
	pubKey, err := crypto.SigToPub(hash.Bytes(), sigBytes)
	if err != nil {
		return false, fmt.Errorf("failed to recover pubkey: %w", err)
	}

	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// Check if recovered address matches expected from address
	return strings.EqualFold(recoveredAddr.Hex(), auth.From.Hex()), nil
}

// getBalance queries the token balance of an address
func (p *Provider) getBalance(ctx context.Context, token, account common.Address) (*big.Int, error) {
	// Pack balanceOf call
	data, err := p.usdcABI.Pack("balanceOf", account)
	if err != nil {
		return nil, fmt.Errorf("failed to pack balanceOf: %w", err)
	}

	// Call contract
	msg := ethereum.CallMsg{
		To:   &token,
		Data: data,
	}
	result, err := p.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, fmt.Errorf("balanceOf call failed: %w", err)
	}

	// Unpack result
	var balance *big.Int
	err = p.usdcABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack balanceOf result: %w", err)
	}

	return balance, nil
}

// transferWithAuthorization submits a transferWithAuthorization transaction
func (p *Provider) transferWithAuthorization(
	ctx context.Context,
	signer *ecdsa.PrivateKey,
	token, from, to common.Address,
	value, validAfter, validBefore *big.Int,
	nonce [32]byte,
	signature []byte,
) (*types.Transaction, error) {
	// Create auth
	auth, err := bind.NewKeyedTransactorWithChainID(signer, p.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	// Get nonce
	signerAddr := crypto.PubkeyToAddress(*signer.Public().(*ecdsa.PublicKey))
	nonceVal, err := p.client.PendingNonceAt(ctx, signerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonceVal))

	// Set gas limit for transferWithAuthorization
	// Fixed at 100,000 (typical usage: ~50-70k, provides safe buffer)
	auth.GasLimit = 100000

	// Get gas price
	gasPrice, err := p.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}
	auth.GasPrice = gasPrice

	// Pack the function call
	data, err := p.usdcABI.Pack(
		"transferWithAuthorization",
		from,
		to,
		value,
		validAfter,
		validBefore,
		nonce,
		signature,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack transferWithAuthorization: %w", err)
	}

	// Create raw transaction
	tx := types.NewTransaction(
		auth.Nonce.Uint64(),
		token,
		big.NewInt(0), // value
		auth.GasLimit,
		auth.GasPrice,
		data,
	)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(p.chainID), signer)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}

	// Send transaction
	err = p.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send tx: %w", err)
	}

	return signedTx, nil
}

// loadUSDABI loads the USDC ABI
func loadUSDABI() (abi.ABI, error) {
	// Simplified - in production, load from file or embed
	const usdcABIJSON = `[{"inputs":[{"internalType":"address","name":"account","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"},{"internalType":"uint256","name":"validAfter","type":"uint256"},{"internalType":"uint256","name":"validBefore","type":"uint256"},{"internalType":"bytes32","name":"nonce","type":"bytes32"},{"internalType":"bytes","name":"signature","type":"bytes"}],"name":"transferWithAuthorization","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	return abi.JSON(strings.NewReader(usdcABIJSON))
}

// loadValidatorABI loads the Validator6492 ABI
func loadValidatorABI() (abi.ABI, error) {
	// Simplified - in production, load from file
	const validatorABIJSON = `[]`
	return abi.JSON(strings.NewReader(validatorABIJSON))
}
