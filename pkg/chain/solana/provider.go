package solana

// import (
// 	"context"
// 	"encoding/base64"
// 	"fmt"

// 	"github.com/gagliardetto/solana-go"
// 	"github.com/gagliardetto/solana-go/rpc"
// 	x402types "github.com/x402-rs/x402-go/pkg/types"
// )

// // Provider handles Solana-based payment verification and settlement
// type Provider struct {
// 	client  *rpc.Client
// 	signer  solana.PrivateKey
// 	network x402types.Network
// }

// // NewProvider creates a new Solana provider
// func NewProvider(rpcURL string, network x402types.Network, privateKeyBase58 string) (*Provider, error) {
// 	client := rpc.New(rpcURL)

// 	// Parse private key
// 	privateKey, err := solana.PrivateKeyFromBase58(privateKeyBase58)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid private key: %w", err)
// 	}

// 	return &Provider{
// 		client:  client,
// 		signer:  privateKey,
// 		network: network,
// 	}, nil
// }

// // Verify validates a Solana payment without submitting a transaction
// func (p *Provider) Verify(ctx context.Context, request *x402types.VerifyRequest) (*x402types.VerifyResponse, error) {
// 	payload := request.PaymentPayload.Payload.Solana
// 	if payload == nil {
// 		return nil, x402types.NewDecodingError("missing Solana payload")
// 	}

// 	// Decode transaction
// 	txBytes, err := base64.StdEncoding.DecodeString(payload.Transaction)
// 	if err != nil {
// 		return nil, x402types.NewDecodingError(fmt.Sprintf("invalid transaction base64: %v", err))
// 	}

// 	// Parse transaction
// 	tx, err := solana.TransactionFromBytes(txBytes)
// 	if err != nil {
// 		return nil, x402types.NewDecodingError(fmt.Sprintf("failed to parse transaction: %v", err))
// 	}

// 	// Validate transaction structure
// 	// TODO: Implement detailed instruction parsing and validation
// 	// - Check compute budget instructions
// 	// - Check CreateATA instruction (if needed)
// 	// - Check transfer instruction amount and recipient

// 	// For now, return a basic validation
// 	if len(tx.Message.Instructions) == 0 {
// 		return &x402types.VerifyResponse{
// 			Valid:  false,
// 			Reason: "transaction has no instructions",
// 		}, nil
// 	}

// 	// Get the first account as payer (simplified)
// 	if len(tx.Message.AccountKeys) == 0 {
// 		return &x402types.VerifyResponse{
// 			Valid:  false,
// 			Reason: "transaction has no account keys",
// 		}, nil
// 	}

// 	payer := x402types.NewSolanaAddress(tx.Message.AccountKeys[0].String())

// 	// Simulate the transaction
// 	simResult, err := p.client.SimulateTransaction(ctx, tx)
// 	if err != nil {
// 		return &x402types.VerifyResponse{
// 			Valid:  false,
// 			Reason: fmt.Sprintf("simulation failed: %v", err),
// 			Payer:  &payer,
// 		}, nil
// 	}

// 	if simResult.Value.Err != nil {
// 		return &x402types.VerifyResponse{
// 			Valid:  false,
// 			Reason: fmt.Sprintf("simulation error: %v", simResult.Value.Err),
// 			Payer:  &payer,
// 		}, nil
// 	}

// 	// All checks passed
// 	return &x402types.VerifyResponse{
// 		Valid: true,
// 		Payer: &payer,
// 	}, nil
// }

// // Settle executes a Solana payment on-chain
// func (p *Provider) Settle(ctx context.Context, request *x402types.SettleRequest) (*x402types.SettleResponse, error) {
// 	// First verify
// 	verifyReq := &x402types.VerifyRequest{
// 		PaymentPayload:      request.PaymentPayload,
// 		PaymentRequirements: request.PaymentRequirements,
// 	}
// 	verifyResp, err := p.Verify(ctx, verifyReq)
// 	if err != nil {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   fmt.Sprintf("verification failed: %v", err),
// 		}, nil
// 	}
// 	if !verifyResp.Valid {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   verifyResp.Reason,
// 		}, nil
// 	}

// 	// Decode transaction
// 	payload := request.PaymentPayload.Payload.Solana
// 	txBytes, err := base64.StdEncoding.DecodeString(payload.Transaction)
// 	if err != nil {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   fmt.Sprintf("invalid transaction: %v", err),
// 		}, nil
// 	}

// 	// Parse transaction
// 	tx, err := solana.TransactionFromBytes(txBytes)
// 	if err != nil {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   fmt.Sprintf("failed to parse transaction: %v", err),
// 		}, nil
// 	}

// 	// Send transaction
// 	sig, err := p.client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
// 		SkipPreflight: false,
// 	})
// 	if err != nil {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   fmt.Sprintf("failed to send transaction: %v", err),
// 		}, nil
// 	}

// 	// Wait for confirmation (simplified - should poll with timeout)
// 	_, err = p.client.GetSignatureStatuses(ctx, true, sig)
// 	if err != nil {
// 		return &x402types.SettleResponse{
// 			Success: false,
// 			Error:   fmt.Sprintf("failed to confirm transaction: %v", err),
// 		}, nil
// 	}

// 	return &x402types.SettleResponse{
// 		Success: true,
// 		TransactionHash: &x402types.TransactionHash{
// 			Type: "solana",
// 			Hash: sig.String(),
// 		},
// 	}, nil
// }
