package config

import (
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/x402-rs/x402-go/pkg/chain/evm"
	"github.com/x402-rs/x402-go/pkg/facilitator"
	"github.com/x402-rs/x402-go/pkg/network"
	"github.com/x402-rs/x402-go/pkg/types"
)

// Config holds the application configuration
type Config struct {
	Host             string
	Port             string
	EVMPrivateKeys   []string
	SolanaPrivateKey string
	RPCURLs          map[types.Network]string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		Host:    getEnvOrDefault("HOST", "0.0.0.0"),
		Port:    getEnvOrDefault("PORT", "8080"),
		RPCURLs: make(map[types.Network]string),
	}

	// Load private keys
	evmKey := os.Getenv("EVM_PRIVATE_KEY")
	if evmKey != "" {
		cfg.EVMPrivateKeys = []string{evmKey}
	}

	// Support multiple EVM private keys
	evmKeys := os.Getenv("EVM_PRIVATE_KEYS")
	if evmKeys != "" {
		cfg.EVMPrivateKeys = strings.Split(evmKeys, ",")
	}

	cfg.SolanaPrivateKey = os.Getenv("SOLANA_PRIVATE_KEY")

	// Load RPC URLs
	rpcMapping := map[types.Network]string{
		types.NetworkBaseSepolia:   "RPC_URL_BASE_SEPOLIA",
		types.NetworkBase:          "RPC_URL_BASE",
		types.NetworkAvalancheFuji: "RPC_URL_AVALANCHE_FUJI",
		types.NetworkAvalanche:     "RPC_URL_AVALANCHE",
		types.NetworkPolygonAmoy:   "RPC_URL_POLYGON_AMOY",
		types.NetworkPolygon:       "RPC_URL_POLYGON",
		types.NetworkSei:           "RPC_URL_SEI",
		types.NetworkSeiTestnet:    "RPC_URL_SEI_TESTNET",
		types.NetworkXDC:           "RPC_URL_XDC",
		types.NetworkSolana:        "RPC_URL_SOLANA",
		types.NetworkSolanaDevnet:  "RPC_URL_SOLANA_DEVNET",
	}

	for network, envKey := range rpcMapping {
		if url := os.Getenv(envKey); url != "" {
			cfg.RPCURLs[network] = url
		}
	}

	return cfg, nil
}

// InitializeFacilitator creates a facilitator from the configuration
func (c *Config) InitializeFacilitator() (*facilitator.LocalFacilitator, error) {
	fac := facilitator.NewLocalFacilitator()

	if len(c.EVMPrivateKeys) == 0 {
		return nil, fmt.Errorf("no EVM private keys configured")
	}

	// Initialize EVM providers
	for net, rpcURL := range c.RPCURLs {
		if !net.IsEVM() {
			continue
		}

		netInfo, err := network.GetNetworkInfo(net)
		if err != nil {
			return nil, fmt.Errorf("failed to get network info for %s: %w", net, err)
		}

		chainID := big.NewInt(int64(netInfo.ChainID))
		provider, err := evm.NewProvider(rpcURL, chainID, net, c.EVMPrivateKeys)
		if err != nil {
			return nil, fmt.Errorf("failed to create EVM provider for %s: %w", net, err)
		}

		fac.AddEVMProvider(net, provider)
		fmt.Printf("Initialized EVM provider for %s (chain ID: %d) at %s\n", netInfo.Name, chainID, rpcURL)
	}

	// // Initialize Solana providers
	// if c.SolanaPrivateKey != "" {
	// 	for net, rpcURL := range c.RPCURLs {
	// 		if !net.IsSolana() {
	// 			continue
	// 		}

	// 		provider, err := solana.NewProvider(rpcURL, net, c.SolanaPrivateKey)
	// 		if err != nil {
	// 			return nil, fmt.Errorf("failed to create Solana provider for %s: %w", net, err)
	// 		}

	// 		fac.AddSolanaProvider(net, provider)
	// 		fmt.Printf("Initialized Solana provider for %s at %s\n", net, rpcURL)
	// 	}
	// }

	return fac, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
