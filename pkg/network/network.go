package network

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/x402-rs/x402-go/pkg/types"
)

// ChainID represents an EVM chain ID
type ChainID uint64

const (
	ChainIDBaseSepolia   ChainID = 84532
	ChainIDBase          ChainID = 8453
	ChainIDAvalancheFuji ChainID = 43113
	ChainIDAvalanche     ChainID = 43114
	ChainIDPolygonAmoy   ChainID = 80002
	ChainIDPolygon       ChainID = 137
	ChainIDSei           ChainID = 1329
	ChainIDSeiTestnet    ChainID = 1328
	ChainIDXDC           ChainID = 50
)

// NetworkInfo contains metadata about a network
type NetworkInfo struct {
	Network types.Network
	ChainID ChainID
	Name    string
	IsEVM   bool
}

// USDCDeployment represents a USDC token deployment on a network
type USDCDeployment struct {
	Network      types.Network
	TokenAddress common.Address
	TokenSymbol  string
	Decimals     uint8
}

var (
	// NetworkInfoMap maps network names to their information
	NetworkInfoMap = map[types.Network]NetworkInfo{
		types.NetworkBaseSepolia: {
			Network: types.NetworkBaseSepolia,
			ChainID: ChainIDBaseSepolia,
			Name:    "Base Sepolia",
			IsEVM:   true,
		},
		types.NetworkBase: {
			Network: types.NetworkBase,
			ChainID: ChainIDBase,
			Name:    "Base",
			IsEVM:   true,
		},
		types.NetworkAvalancheFuji: {
			Network: types.NetworkAvalancheFuji,
			ChainID: ChainIDAvalancheFuji,
			Name:    "Avalanche Fuji",
			IsEVM:   true,
		},
		types.NetworkAvalanche: {
			Network: types.NetworkAvalanche,
			ChainID: ChainIDAvalanche,
			Name:    "Avalanche C-Chain",
			IsEVM:   true,
		},
		types.NetworkPolygonAmoy: {
			Network: types.NetworkPolygonAmoy,
			ChainID: ChainIDPolygonAmoy,
			Name:    "Polygon Amoy",
			IsEVM:   true,
		},
		types.NetworkPolygon: {
			Network: types.NetworkPolygon,
			ChainID: ChainIDPolygon,
			Name:    "Polygon",
			IsEVM:   true,
		},
		types.NetworkSei: {
			Network: types.NetworkSei,
			ChainID: ChainIDSei,
			Name:    "Sei",
			IsEVM:   true,
		},
		types.NetworkSeiTestnet: {
			Network: types.NetworkSeiTestnet,
			ChainID: ChainIDSeiTestnet,
			Name:    "Sei Testnet",
			IsEVM:   true,
		},
		types.NetworkXDC: {
			Network: types.NetworkXDC,
			ChainID: ChainIDXDC,
			Name:    "XDC",
			IsEVM:   true,
		},
		types.NetworkSolana: {
			Network: types.NetworkSolana,
			Name:    "Solana",
			IsEVM:   false,
		},
		types.NetworkSolanaDevnet: {
			Network: types.NetworkSolanaDevnet,
			Name:    "Solana Devnet",
			IsEVM:   false,
		},
	}

	// USDCDeployments maps networks to their USDC token deployments
	USDCDeployments = map[types.Network]USDCDeployment{
		types.NetworkBaseSepolia: {
			Network:      types.NetworkBaseSepolia,
			TokenAddress: common.HexToAddress("0x036CbD53842c5426634e7929541eC2318f3dCF7e"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkBase: {
			Network:      types.NetworkBase,
			TokenAddress: common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkAvalancheFuji: {
			Network:      types.NetworkAvalancheFuji,
			TokenAddress: common.HexToAddress("0x5425890298aed601595a70AB815c96711a31Bc65"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkAvalanche: {
			Network:      types.NetworkAvalanche,
			TokenAddress: common.HexToAddress("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkPolygonAmoy: {
			Network:      types.NetworkPolygonAmoy,
			TokenAddress: common.HexToAddress("0x41e94eb019c0762f9bfcf9fb1e58725bfb0e7582"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkPolygon: {
			Network:      types.NetworkPolygon,
			TokenAddress: common.HexToAddress("0x3c499c542cef5e3811e1192ce70d8cc03d5c3359"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
		types.NetworkXDC: {
			Network:      types.NetworkXDC,
			TokenAddress: common.HexToAddress("0xD4B5f10D61916Bd6E0860144a91Ac658dE8a1437"),
			TokenSymbol:  "USDC",
			Decimals:     6,
		},
	}

	// ValidatorAddress is the EIP-6492 validator contract address
	ValidatorAddress = common.HexToAddress("0xdAcD51A54883eb67D95FAEb2BBfdC4a9a6BD2a3B")
)

// GetNetworkInfo returns information about a network
func GetNetworkInfo(network types.Network) (NetworkInfo, error) {
	info, ok := NetworkInfoMap[network]
	if !ok {
		return NetworkInfo{}, fmt.Errorf("unknown network: %s", network)
	}
	return info, nil
}

// GetUSDCDeployment returns the USDC deployment for a network
func GetUSDCDeployment(network types.Network) (USDCDeployment, error) {
	deployment, ok := USDCDeployments[network]
	if !ok {
		return USDCDeployment{}, fmt.Errorf("no USDC deployment for network: %s", network)
	}
	return deployment, nil
}

// ParseAmount parses a decimal amount string to wei/smallest unit
func ParseAmount(amount string, decimals uint8) (*big.Int, error) {
	// This is a simplified version - in production use decimal parsing library
	value := new(big.Float)
	_, ok := value.SetString(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s", amount)
	}

	// Multiply by 10^decimals
	multiplier := new(big.Float).SetInt(new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(decimals)),
		nil,
	))
	value.Mul(value, multiplier)

	// Convert to integer
	result := new(big.Int)
	value.Int(result)
	return result, nil
}

// IsEVMNetwork checks if a network is EVM-compatible
func IsEVMNetwork(network types.Network) bool {
	info, err := GetNetworkInfo(network)
	if err != nil {
		return false
	}
	return info.IsEVM
}

// IsSolanaNetwork checks if a network is Solana-based
func IsSolanaNetwork(network types.Network) bool {
	return network == types.NetworkSolana || network == types.NetworkSolanaDevnet
}
