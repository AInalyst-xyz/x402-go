# x402-go

A Go-based implementation of the x402 protocol.

## Requirements

- Go 1.23+

## Install


## Configure

Create a minimal `.env` (example uses Base Sepolia and a dummy key):

```bash
cat > .env << 'EOF'
HOST=0.0.0.0
PORT=8080
RPC_URL_BASE_SEPOLIA=https://sepolia.base.org
EVM_PRIVATE_KEY=0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
EOF
```

## Run

```bash
make run-facilitator
```

## Verify

```bash
curl -s http://localhost:8080/supported | jq .
```
