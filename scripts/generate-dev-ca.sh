#!/bin/bash
# Generate development CA for FreeKiosk Hub
# Usage: ./scripts/generate-dev-ca.sh [output_dir]

OUTPUT_DIR="${1:-certs}"
mkdir -p "$OUTPUT_DIR"

CA_CERT="$OUTPUT_DIR/ca.crt"
CA_KEY="$OUTPUT_DIR/ca.key"

echo "🔐 Generating development CA for FreeKiosk Hub..."

# Generate CA private key
openssl genrsa -out "$CA_KEY" 2048 2>/dev/null

# Generate CA certificate (self-signed)
openssl req -new -x509 -key "$CA_KEY" -out "$CA_CERT" -days 3650 \
    -subj "/O=FreeKiosk Enterprise/CN=FreeKiosk Enterprise CA" 2>/dev/null

echo "✅ Development CA generated:"
echo "   Certificate: $CA_CERT"
echo "   Private Key: $CA_KEY"

# Display certificate info
echo ""
echo "📜 Certificate details:"
openssl x509 -in "$CA_CERT" -noout -subject -issuer -dates 2>/dev/null
