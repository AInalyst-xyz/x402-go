package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// X402Version represents the protocol version
type X402Version string

const (
	X402VersionV1 X402Version = "1"
)

// Scheme represents the payment scheme
type Scheme string

const (
	SchemeExact Scheme = "exact"
)

// Network represents supported blockchain networks
type Network string

const (
	NetworkBaseSepolia   Network = "base-sepolia"
	NetworkBase          Network = "base"
	NetworkAvalancheFuji Network = "avalanche-fuji"
	NetworkAvalanche     Network = "avalanche"
	NetworkPolygonAmoy   Network = "polygon-amoy"
	NetworkPolygon       Network = "polygon"
	NetworkSei           Network = "sei"
	NetworkSeiTestnet    Network = "sei-testnet"
	NetworkXDC           Network = "xdc"
	NetworkSolana        Network = "solana"
	NetworkSolanaDevnet  Network = "solana-devnet"
)

// MixedAddress represents an address on any supported chain
type MixedAddress struct {
	Type    string `json:"type"` // "evm", "solana", "offchain"
	Address string `json:"address"`
}

// NewEvmAddress creates a new EVM address
func NewEvmAddress(addr common.Address) MixedAddress {
	return MixedAddress{
		Type:    "evm",
		Address: addr.Hex(),
	}
}

// NewSolanaAddress creates a new Solana address
func NewSolanaAddress(addr string) MixedAddress {
	return MixedAddress{
		Type:    "solana",
		Address: addr,
	}
}

// NewOffchainAddress creates a new off-chain address
func NewOffchainAddress(addr string) MixedAddress {
	return MixedAddress{
		Type:    "offchain",
		Address: addr,
	}
}

// PaymentRequirements specifies what payment is required
type PaymentRequirements struct {
	Version           X402Version     `json:"version"`
	Scheme            Scheme          `json:"scheme"`
	Network           Network         `json:"network"`
	PayTo             string          `json:"payTo"`
	MaxAmountRequired string          `json:"maxAmountRequired"`
	Resource          string          `json:"resource"`
	Description       string          `json:"description"`
	MimeType          string          `json:"mimeType"`
	MaxTimeoutSeconds int             `json:"maxTimeoutSeconds"`
	Asset             common.Address  `json:"asset"`
	OutputSchema      json.RawMessage `json:"outputSchema"`
	Extra             json.RawMessage `json:"extra"`
}

// ExactEvmPayloadAuthorization represents EIP-712 transfer authorization data
type ExactEvmPayloadAuthorization struct {
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	Value       string         `json:"value"`
	ValidAfter  string         `json:"validAfter"`
	ValidBefore string         `json:"validBefore"`
	Nonce       string         `json:"nonce"` // hex-encoded
}

// ExactEvmPayload contains the EVM payment payload
type ExactEvmPayload struct {
	Signature     string                       `json:"signature"` // hex-encoded
	Authorization ExactEvmPayloadAuthorization `json:"authorization"`
}

// ExactSolanaPayload contains the Solana payment payload
type ExactSolanaPayload struct {
	Transaction string `json:"transaction"` // base64-encoded versioned transaction
}

// ExactPaymentPayload is a union of EVM and Solana payloads
type ExactPaymentPayload struct {
	Evm    *ExactEvmPayload    `json:"evm,omitempty"`
	Solana *ExactSolanaPayload `json:"solana,omitempty"`
}

// PaymentPayload contains the complete payment information
type PaymentPayload struct {
	X402Version int             `json:"x402Version"`
	Scheme      Scheme          `json:"scheme"`
	Network     Network         `json:"network"`
	Payload     ExactEvmPayload `json:"payload"`
}

// VerifyRequest is the request to verify a payment
type VerifyRequest struct {
	X402Version         int                 `json:"x402Version"`
	PaymentPayload      PaymentPayload      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements `json:"paymentRequirements"`
}

// SettleRequest is the request to settle a payment
type SettleRequest struct {
	PaymentPayload      PaymentPayload      `json:"paymentPayload"`
	PaymentRequirements PaymentRequirements `json:"paymentRequirements"`
}

// VerifyResponse is the response from payment verification
type VerifyResponse struct {
	IsValid  bool          `json:"isValid"`
	Payer  *MixedAddress `json:"payer,omitempty"`
	Reason string        `json:"reason,omitempty"`
}

// NewValidResponse creates a successful verification response
func NewValidResponse(payer MixedAddress) VerifyResponse {
	return VerifyResponse{
		IsValid: true,
		Payer: &payer,
	}
}

// NewInvalidResponse creates a failed verification response
func NewInvalidResponse(reason string, payer *MixedAddress) VerifyResponse {
	return VerifyResponse{
		IsValid:  false,
		Reason: reason,
		Payer:  payer,
	}
}

// TransactionHash represents a transaction hash on any chain
type TransactionHash struct {
	Type string `json:"type"` // "evm" or "solana"
	Hash string `json:"hash"`
}

// SettleResponse is the response from payment settlement
type SettleResponse struct {
	Success         bool             `json:"success"`
	TransactionHash *TransactionHash `json:"transaction_hash,omitempty"`
	Error           string           `json:"error,omitempty"`
}

// SupportedPaymentKind represents a supported payment type
type SupportedPaymentKind struct {
	Version     X402Version  `json:"version"`
	Scheme      Scheme       `json:"scheme"`
	Network     Network      `json:"network"`
	Token       MixedAddress `json:"token"`
	TokenSymbol string       `json:"token_symbol"`
}

// SupportedPaymentKindsResponse lists all supported payment kinds
type SupportedPaymentKindsResponse struct {
	Kinds []SupportedPaymentKind `json:"kinds"`
}

// Error types

// FacilitatorError represents errors that can occur during facilitation
type FacilitatorError struct {
	Type    string
	Message string
	Payer   *MixedAddress
}

func (e *FacilitatorError) Error() string {
	if e.Payer != nil {
		return fmt.Sprintf("%s: %s (payer: %s)", e.Type, e.Message, e.Payer.Address)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Error constructors

func NewUnsupportedNetworkError(payer *MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "UnsupportedNetwork",
		Message: "network not supported by this facilitator",
		Payer:   payer,
	}
}

func NewNetworkMismatchError(expected, actual Network, payer *MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "NetworkMismatch",
		Message: fmt.Sprintf("expected %s, got %s", expected, actual),
		Payer:   payer,
	}
}

func NewSchemeMismatchError(expected, actual Scheme, payer *MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "SchemeMismatch",
		Message: fmt.Sprintf("expected %s, got %s", expected, actual),
		Payer:   payer,
	}
}

func NewReceiverMismatchError(expected, actual string, payer MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "ReceiverMismatch",
		Message: fmt.Sprintf("expected %s, got %s", expected, actual),
		Payer:   &payer,
	}
}

func NewInvalidTimingError(payer MixedAddress, message string) *FacilitatorError {
	return &FacilitatorError{
		Type:    "InvalidTiming",
		Message: message,
		Payer:   &payer,
	}
}

func NewInsufficientFundsError(payer MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "InsufficientFunds",
		Message: "payer has insufficient balance",
		Payer:   &payer,
	}
}

func NewInsufficientValueError(payer MixedAddress) *FacilitatorError {
	return &FacilitatorError{
		Type:    "InsufficientValue",
		Message: "payment amount less than required",
		Payer:   &payer,
	}
}

func NewInvalidSignatureError(payer MixedAddress, message string) *FacilitatorError {
	return &FacilitatorError{
		Type:    "InvalidSignature",
		Message: message,
		Payer:   &payer,
	}
}

func NewDecodingError(message string) *FacilitatorError {
	return &FacilitatorError{
		Type:    "DecodingError",
		Message: message,
	}
}

func NewContractCallError(message string) *FacilitatorError {
	return &FacilitatorError{
		Type:    "ContractCallError",
		Message: message,
	}
}

// Helper functions

// UnixTimestamp returns the current Unix timestamp in seconds
func UnixTimestamp() uint64 {
	return uint64(time.Now().Unix())
}

// MarshalJSON custom marshaling for better compatibility
func (p *PaymentPayload) MarshalJSON() ([]byte, error) {
	type Alias PaymentPayload
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(p),
	})
}

// IsEVM returns true if the network is EVM-compatible
func (n Network) IsEVM() bool {
	switch n {
	case NetworkBaseSepolia, NetworkBase, NetworkAvalancheFuji, NetworkAvalanche,
		NetworkPolygonAmoy, NetworkPolygon, NetworkSei, NetworkSeiTestnet, NetworkXDC:
		return true
	default:
		return false
	}
}

// IsSolana returns true if the network is Solana-based
func (n Network) IsSolana() bool {
	return n == NetworkSolana || n == NetworkSolanaDevnet
}
