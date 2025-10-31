package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/x402-rs/x402-go/middleware/server"
	"github.com/x402-rs/x402-go/pkg/types"
)

func main() {
	// Create x402 middleware pointing to facilitator
	x402 := server.NewX402Middleware("http://localhost:8080")

	// Create price tag for protected content
	// 0.025 USDC on Base Sepolia
	priceTag := server.NewPriceTagBuilder().
		Network(types.NetworkBaseSepolia).
		Amount("25000"). // 0.025 USDC in smallest units (6 decimals)
		TokenSymbol("USDC").
		PayTo(types.NewEvmAddress(common.HexToAddress("0xYourAddress"))).                              // Replace with your address
		Token(types.NewEvmAddress(common.HexToAddress("0x036CbD53842c5426634e7929541eC2318f3dCF7e"))). // USDC on Base Sepolia
		Build()

	// Create HTTP handlers
	mux := http.NewServeMux()

	// Free endpoint
	mux.HandleFunc("/free", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "This content is free!",
		})
	})

	// Protected endpoint (requires payment)
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "This is premium content! You paid for this.",
			"secret":  "The answer is 42",
		})
	})
	mux.Handle("/premium", x402.Protect(protectedHandler, priceTag))

	// Start server
	addr := ":3000"
	fmt.Printf("Server listening on %s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET /free     - Free content")
	fmt.Println("  GET /premium  - Paid content (0.025 USDC)")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
