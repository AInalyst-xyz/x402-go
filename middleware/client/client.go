package client

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/x402-rs/x402-go/pkg/types"
)

// PayingClient is an HTTP client that automatically handles x402 payments
type PayingClient struct {
	client     *http.Client
	signer     *ecdsa.PrivateKey
	signerAddr common.Address
}

// NewPayingClient creates a new client with payment capabilities
func NewPayingClient(privateKeyHex string) (*PayingClient, error) {
	// Parse private key
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Get address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return &PayingClient{
		client:     &http.Client{},
		signer:     privateKey,
		signerAddr: address,
	}, nil
}

// Get performs a GET request with automatic payment handling
func (c *PayingClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Post performs a POST request with automatic payment handling
func (c *PayingClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

// Do executes an HTTP request with automatic payment handling
func (c *PayingClient) Do(req *http.Request) (*http.Response, error) {
	// First, try the request without payment
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// If not 402, return response
	if resp.StatusCode != http.StatusPaymentRequired {
		return resp, nil
	}

	// Parse payment requirements from 402 response
	requirements, err := c.parsePaymentRequirements(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payment requirements: %w", err)
	}

	// Generate payment payload
	payload, err := c.generatePaymentPayload(requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate payment: %w", err)
	}

	// Retry request with payment
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment: %w", err)
	}

	// Clone request
	retryReq := req.Clone(req.Context())
	retryReq.Header.Set("X-Payment-Payload", string(payloadJSON))

	// Execute with payment
	return c.client.Do(retryReq)
}

// parsePaymentRequirements extracts payment requirements from a 402 response
func (c *PayingClient) parsePaymentRequirements(resp *http.Response) (*types.PaymentRequirements, error) {
	// Try header first
	reqHeader := resp.Header.Get("X-Payment-Required")
	if reqHeader != "" {
		var requirements types.PaymentRequirements
		if err := json.Unmarshal([]byte(reqHeader), &requirements); err == nil {
			return &requirements, nil
		}
	}

	// Try body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	var response struct {
		PaymentRequirements types.PaymentRequirements `json:"payment_requirements"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response.PaymentRequirements, nil
}

// generatePaymentPayload creates a payment payload for the given requirements
func (c *PayingClient) generatePaymentPayload(requirements *types.PaymentRequirements) (*types.PaymentPayload, error) {
	// Only support EVM for now
	if !requirements.Network.IsEVM() {
		return nil, fmt.Errorf("unsupported network: %s", requirements.Network)
	}

	// Generate nonce
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Set validity window (1 hour)
	now := time.Now().Unix()
	validAfter := uint64(now)
	validBefore := uint64(now + 3600)

	// Parse receiver address
	receiverAddr := requirements.PayTo

	// Create authorization
	auth := types.ExactEvmPayloadAuthorization{
		From:        c.signerAddr,
		To:          common.HexToAddress(receiverAddr),
		Value:       requirements.MaxAmountRequired,
		ValidAfter:  fmt.Sprintf("%d", validAfter),
		ValidBefore: fmt.Sprintf("%d", validBefore),
		Nonce:       "0x" + hex.EncodeToString(nonce),
	}

	// Sign with EIP-712
	signature, err := c.signEIP712(&auth, requirements.Asset.Hex(), requirements.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Create payload
	return &types.PaymentPayload{
		X402Version: 1,
		Scheme:      types.SchemeExact,
		Network:     requirements.Network,
		Payload: types.ExactEvmPayload{
			Signature:     "0x" + hex.EncodeToString(signature),
			Authorization: auth,
		},
	}, nil
}

// signEIP712 signs the authorization with EIP-712
func (c *PayingClient) signEIP712(auth *types.ExactEvmPayloadAuthorization, tokenAddress string, network types.Network) ([]byte, error) {
	// Get chain ID for network
	chainID, err := c.getChainID(network)
	if err != nil {
		return nil, err
	}

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
			ChainId:           (*math.HexOrDecimal256)(chainID),
			VerifyingContract: tokenAddress,
		},
		Message: apitypes.TypedDataMessage{
			"from":        auth.From.Hex(),
			"to":          auth.To.Hex(),
			"value":       auth.Value,
			"validAfter":  fmt.Sprintf("%d", auth.ValidAfter),
			"validBefore": fmt.Sprintf("%d", auth.ValidBefore),
			"nonce":       auth.Nonce,
		},
	}

	// Hash the typed data
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash message: %w", err)
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256Hash(rawData)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), c.signer)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Adjust V value
	if signature[64] < 27 {
		signature[64] += 27
	}

	return signature, nil
}

// getChainID returns the chain ID for a network
func (c *PayingClient) getChainID(network types.Network) (*big.Int, error) {
	// Hardcoded chain IDs for now
	chainIDs := map[types.Network]int64{
		types.NetworkBaseSepolia:   84532,
		types.NetworkBase:          8453,
		types.NetworkAvalancheFuji: 43113,
		types.NetworkAvalanche:     43114,
		types.NetworkPolygonAmoy:   80002,
		types.NetworkPolygon:       137,
		types.NetworkSei:           1329,
		types.NetworkSeiTestnet:    1328,
		types.NetworkXDC:           50,
	}

	chainID, ok := chainIDs[network]
	if !ok {
		return nil, fmt.Errorf("unknown chain ID for network: %s", network)
	}

	return big.NewInt(chainID), nil
}
