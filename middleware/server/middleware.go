package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/x402-rs/x402-go/pkg/types"
)

// X402Middleware provides payment protection for HTTP handlers
type X402Middleware struct {
	facilitatorURL string
	client         *http.Client
}

// NewX402Middleware creates a new middleware instance
func NewX402Middleware(facilitatorURL string) *X402Middleware {
	return &X402Middleware{
		facilitatorURL: strings.TrimSuffix(facilitatorURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second, // Prevent indefinite hangs
		},
	}
}

// PriceTag represents payment requirements for a route
type PriceTag struct {
	Requirements types.PaymentRequirements
}

// NewPriceTag creates a new price tag
func NewPriceTag(network types.Network, amount, tokenSymbol string, payTo, token types.MixedAddress, resource, description, mimeType string, maxTimeoutSeconds int, asset types.MixedAddress, outputSchema json.RawMessage) *PriceTag {
	return &PriceTag{
		Requirements: types.PaymentRequirements{
			Version:           types.X402VersionV1,
			Scheme:            types.SchemeExact,
			Network:           network,
			PayTo:             payTo.Address,
			MaxAmountRequired: amount,
			Resource:          resource,
			Description:       description,
			MimeType:          mimeType,
			MaxTimeoutSeconds: maxTimeoutSeconds,
			Asset:             common.HexToAddress(asset.Address),
			OutputSchema:      outputSchema,
		},
	}
}

// Protect wraps an HTTP handler with payment verification
func (m *X402Middleware) Protect(next http.Handler, priceTag *PriceTag) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for payment header
		paymentHeader := r.Header.Get("X-Payment-Payload")
		if paymentHeader == "" {
			// No payment provided, return 402 Payment Required
			m.send402(w, &priceTag.Requirements)
			return
		}

		// Parse payment payload
		var payload types.PaymentPayload
		if err := json.Unmarshal([]byte(paymentHeader), &payload); err != nil {
			http.Error(w, fmt.Sprintf("invalid payment payload: %v", err), http.StatusBadRequest)
			return
		}

		// Verify payment with facilitator
		verifyReq := types.VerifyRequest{
			PaymentPayload:      payload,
			PaymentRequirements: priceTag.Requirements,
		}

		verifyResp, err := m.verifyPayment(&verifyReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("payment verification failed: %v", err), http.StatusInternalServerError)
			return
		}

		if !verifyResp.IsValid {
			// Payment invalid, return 402 with reason
			m.send402WithReason(w, &priceTag.Requirements, verifyResp.Reason)
			return
		}

		// Payment valid, call next handler
		next.ServeHTTP(w, r)
	})
}

// verifyPayment calls the facilitator to verify a payment
func (m *X402Middleware) verifyPayment(req *types.VerifyRequest) (*types.VerifyResponse, error) {
	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call facilitator
	resp, err := m.client.Post(
		m.facilitatorURL+"/verify",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("facilitator request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var verifyResp types.VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &verifyResp, nil
}

// send402 sends a 402 Payment Required response
func (m *X402Middleware) send402(w http.ResponseWriter, requirements *types.PaymentRequirements) {
	m.send402WithReason(w, requirements, "")
}

// send402WithReason sends a 402 Payment Required response with a reason
func (m *X402Middleware) send402WithReason(w http.ResponseWriter, requirements *types.PaymentRequirements, reason string) {
	// Marshal requirements
	reqJSON, _ := json.Marshal(requirements)

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Payment-Required", string(reqJSON))
	w.WriteHeader(http.StatusPaymentRequired)

	// Response body
	response := map[string]interface{}{
		"error":                "payment required",
		"payment_requirements": requirements,
	}
	if reason != "" {
		response["reason"] = reason
	}

	json.NewEncoder(w).Encode(response)
}

// Helper builder for creating price tags

// PriceTagBuilder provides a fluent API for creating price tags
type PriceTagBuilder struct {
	network           types.Network
	amount            string
	tokenSymbol       string
	payTo             types.MixedAddress
	token             types.MixedAddress
	resource          string
	description       string
	mimeType          string
	maxTimeoutSeconds int
	asset             types.MixedAddress
	extra             json.RawMessage
}

// NewPriceTagBuilder creates a new builder
func NewPriceTagBuilder() *PriceTagBuilder {
	return &PriceTagBuilder{}
}

// Network sets the blockchain network
func (b *PriceTagBuilder) Network(network types.Network) *PriceTagBuilder {
	b.network = network
	return b
}

// Amount sets the payment amount
func (b *PriceTagBuilder) Amount(amount string) *PriceTagBuilder {
	b.amount = amount
	return b
}

// TokenSymbol sets the token symbol
func (b *PriceTagBuilder) TokenSymbol(symbol string) *PriceTagBuilder {
	b.tokenSymbol = symbol
	return b
}

// PayTo sets the recipient address
func (b *PriceTagBuilder) PayTo(addr types.MixedAddress) *PriceTagBuilder {
	b.payTo = addr
	return b
}

// Token sets the token address
func (b *PriceTagBuilder) Token(addr types.MixedAddress) *PriceTagBuilder {
	b.token = addr
	return b
}

// Build creates the price tag
func (b *PriceTagBuilder) Build() *PriceTag {
	return NewPriceTag(b.network, b.amount, b.tokenSymbol, b.payTo, b.token, b.resource, b.description, b.mimeType, b.maxTimeoutSeconds, b.asset, b.extra)
}
