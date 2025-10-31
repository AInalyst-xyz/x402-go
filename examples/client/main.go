package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/x402-rs/x402-go/middleware/client"
)

func main() {
	// Get private key from environment
	privateKey := os.Getenv("EVM_PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("EVM_PRIVATE_KEY environment variable not set")
	}

	// Create paying client
	payingClient, err := client.NewPayingClient(privateKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Try to access free endpoint
	fmt.Println("Accessing free endpoint...")
	resp, err := payingClient.Get("http://localhost:3000/free")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response (status %d): %s\n\n", resp.StatusCode, string(body))

	// Try to access premium endpoint (will automatically pay)
	fmt.Println("Accessing premium endpoint...")
	resp, err = payingClient.Get("http://localhost:3000/premium")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode == 200 {
		fmt.Printf("✓ Payment successful!\n")
		fmt.Printf("Response: %s\n", string(body))
	} else {
		fmt.Printf("✗ Payment failed (status %d): %s\n", resp.StatusCode, string(body))
	}
}
