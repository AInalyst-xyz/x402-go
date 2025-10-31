#!/bin/bash

echo "================================"
echo "x402-go HTTP Logging Proxy"
echo "================================"
echo ""
echo "This will start a logging proxy that captures all HTTP traffic"
echo "to your facilitator running on port 4000"
echo ""
echo "Setup:"
echo "  1. Keep your facilitator running on port 4000"
echo "  2. Run this script in another terminal"
echo "  3. Access facilitator through http://localhost:8081 (instead of :4000)"
echo "  4. Watch logs in real-time"
echo ""
echo "Press Enter to start, or Ctrl+C to cancel..."
read

cd "$(dirname "$0")"
python3 scripts/proxy-logger.py
