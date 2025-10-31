package facilitator

import (
	"context"

	"github.com/x402-rs/x402-go/pkg/types"
)

// Facilitator is the core interface for payment verification and settlement.
//
// A Facilitator handles:
// - Payment verification: Confirming that client-submitted payment payloads
//   match the declared requirements.
// - Payment settlement: Submitting validated payments to the blockchain and
//   monitoring their confirmation.
//
// The Facilitator never holds user funds. It acts solely as a stateless
// verification and execution layer for signed payment payloads.
type Facilitator interface {
	// Verify validates a payment payload against requirements.
	//
	// This checks:
	// - Payload integrity and signature validity
	// - Balance sufficiency
	// - Network compatibility
	// - Compliance with payment requirements
	//
	// Returns a VerifyResponse indicating success or failure.
	// This is an off-chain operation and does not submit transactions.
	Verify(ctx context.Context, request *types.VerifyRequest) (*types.VerifyResponse, error)

	// Settle executes a verified payment on-chain.
	//
	// This:
	// - Re-validates the payment (same checks as Verify)
	// - Submits the transaction to the blockchain
	// - Waits for confirmation
	//
	// Returns a SettleResponse with the transaction hash or error.
	// This is an on-chain operation that consumes gas.
	Settle(ctx context.Context, request *types.SettleRequest) (*types.SettleResponse, error)

	// Supported returns the payment kinds supported by this facilitator.
	//
	// This includes all configured networks and their token deployments.
	Supported(ctx context.Context) (*types.SupportedPaymentKindsResponse, error)
}
