package facilitator

import (
	"context"
	"fmt"

	"github.com/x402-rs/x402-go/pkg/chain/evm"
	"github.com/x402-rs/x402-go/pkg/network"
	"github.com/x402-rs/x402-go/pkg/types"
)

// LocalFacilitator is a concrete implementation of the Facilitator interface.
//
// It manages providers for multiple blockchain networks and routes
// verification/settlement requests to the appropriate chain handler.
type LocalFacilitator struct {
	evmProviders map[types.Network]*evm.Provider
	// solanaProviders map[types.Network]*solana.Provider
}

// NewLocalFacilitator creates a new LocalFacilitator instance.
func NewLocalFacilitator() *LocalFacilitator {
	return &LocalFacilitator{
		evmProviders: make(map[types.Network]*evm.Provider),
		// solanaProviders: make(map[types.Network]*solana.Provider),
	}
}

// AddEVMProvider registers an EVM provider for a network.
func (f *LocalFacilitator) AddEVMProvider(network types.Network, provider *evm.Provider) {
	f.evmProviders[network] = provider
}

// // AddSolanaProvider registers a Solana provider for a network.
// func (f *LocalFacilitator) AddSolanaProvider(network types.Network, provider *solana.Provider) {
// 	// f.solanaProviders[network] = provider
// }

// Verify implements Facilitator.Verify
func (f *LocalFacilitator) Verify(ctx context.Context, request *types.VerifyRequest) (*types.VerifyResponse, error) {
	// Basic validation
	if err := f.validateRequest(&request.PaymentPayload, &request.PaymentRequirements); err != nil {
		return nil, err
	}

	network := request.PaymentPayload.Network

	// Route to appropriate chain handler
	if network.IsEVM() {
		provider, ok := f.evmProviders[network]
		if !ok {
			err := types.NewUnsupportedNetworkError(nil)
			response := types.NewInvalidResponse(err.Message, nil)
			return &response, nil
		}
		return provider.Verify(ctx, request)
	}

	// if network.IsSolana() {
	// 	provider, ok := f.solanaProviders[network]
	// 	if !ok {
	// 		err := types.NewUnsupportedNetworkError(nil)
	// 		response := types.NewInvalidResponse(err.Message, nil)
	// 		return &response, nil
	// 	}
	// 	return provider.Verify(ctx, request)
	// }

	err := types.NewUnsupportedNetworkError(nil)
	response := types.NewInvalidResponse(err.Message, nil)
	return &response, nil
}

// Settle implements Facilitator.Settle
func (f *LocalFacilitator) Settle(ctx context.Context, request *types.SettleRequest) (*types.SettleResponse, error) {
	// Basic validation
	if err := f.validateRequest(&request.PaymentPayload, &request.PaymentRequirements); err != nil {
		return nil, err
	}

	network := request.PaymentPayload.Network

	// Route to appropriate chain handler
	if network.IsEVM() {
		provider, ok := f.evmProviders[network]
		if !ok {
			return &types.SettleResponse{
				Success: false,
				Error:   "network not supported",
			}, nil
		}
		return provider.Settle(ctx, request)
	}

	// if network.IsSolana() {
	// 	provider, ok := f.solanaProviders[network]
	// 	if !ok {
	// 		return &types.SettleResponse{
	// 			Success: false,
	// 			Error:   "network not supported",
	// 		}, nil
	// 	}
	// 	return provider.Settle(ctx, request)
	// }

	return &types.SettleResponse{
		Success: false,
		Error:   "network not supported",
	}, nil
}

// Supported implements Facilitator.Supported
func (f *LocalFacilitator) Supported(ctx context.Context) (*types.SupportedPaymentKindsResponse, error) {
	kinds := []types.SupportedPaymentKind{}

	// Add EVM networks with USDC
	for net := range f.evmProviders {
		deployment, err := network.GetUSDCDeployment(net)
		if err != nil {
			continue // Skip if no USDC deployment
		}

		kinds = append(kinds, types.SupportedPaymentKind{
			Version:     types.X402VersionV1,
			Scheme:      types.SchemeExact,
			Network:     net,
			Token:       types.NewEvmAddress(deployment.TokenAddress),
			TokenSymbol: deployment.TokenSymbol,
		})
	}

	// // Add Solana networks
	// for net := range f.solanaProviders {
	// 	// TODO: Add Solana USDC mint addresses
	// 	kinds = append(kinds, types.SupportedPaymentKind{
	// 		Version:     types.X402VersionV1,
	// 		Scheme:      types.SchemeExact,
	// 		Network:     net,
	// 		TokenSymbol: "USDC",
	// 	})
	// }

	return &types.SupportedPaymentKindsResponse{
		Kinds: kinds,
	}, nil
}

// validateRequest performs basic validation on the request
func (f *LocalFacilitator) validateRequest(payload *types.PaymentPayload, requirements *types.PaymentRequirements) error {
	// Check scheme match
	if payload.Scheme != requirements.Scheme {
		return types.NewSchemeMismatchError(requirements.Scheme, payload.Scheme, nil)
	}

	// Check network match
	if payload.Network != requirements.Network {
		return types.NewNetworkMismatchError(requirements.Network, payload.Network, nil)
	}

	// Check version
	if payload.X402Version != 1 {
		return fmt.Errorf("unsupported version: %d", payload.X402Version)
	}

	return nil
}
