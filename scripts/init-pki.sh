#!/bin/bash

# Initialize PKI for ReignX
# This script creates the certificate authority hierarchy and initial certificates

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CERTS_DIR="$PROJECT_ROOT/certs"

echo "Initializing ReignX PKI..."

# Create certificates directory structure
mkdir -p "$CERTS_DIR"/{ca,server,agents}
mkdir -p "$CERTS_DIR/ca"/{root,intermediate}

echo "✓ Created certificate directories"

# Generate Root CA
echo "Generating Root CA..."
openssl genrsa -out "$CERTS_DIR/ca/root/ca-key.pem" 4096

openssl req -new -x509 -days 3650 -key "$CERTS_DIR/ca/root/ca-key.pem" \
    -out "$CERTS_DIR/ca/root/ca-cert.pem" \
    -subj "/C=US/ST=California/L=San Francisco/O=ReignX/CN=ReignX Root CA"

echo "✓ Generated Root CA"

# Generate Server Intermediate CA
echo "Generating Server Intermediate CA..."
openssl genrsa -out "$CERTS_DIR/ca/intermediate/server-ca-key.pem" 4096

openssl req -new -key "$CERTS_DIR/ca/intermediate/server-ca-key.pem" \
    -out "$CERTS_DIR/ca/intermediate/server-ca.csr" \
    -subj "/C=US/ST=California/L=San Francisco/O=ReignX/CN=ReignX Server Intermediate CA"

# Create extension file for intermediate CA
cat > "$CERTS_DIR/ca/intermediate/server-ca-ext.cnf" <<EOF
basicConstraints = CA:TRUE
keyUsage = keyCertSign, cRLSign
subjectKeyIdentifier = hash
EOF

openssl x509 -req -days 1825 \
    -in "$CERTS_DIR/ca/intermediate/server-ca.csr" \
    -CA "$CERTS_DIR/ca/root/ca-cert.pem" \
    -CAkey "$CERTS_DIR/ca/root/ca-key.pem" \
    -CAcreateserial \
    -out "$CERTS_DIR/ca/intermediate/server-ca-cert.pem" \
    -extfile "$CERTS_DIR/ca/intermediate/server-ca-ext.cnf"

echo "✓ Generated Server Intermediate CA"

# Generate Agent Intermediate CA
echo "Generating Agent Intermediate CA..."
openssl genrsa -out "$CERTS_DIR/ca/intermediate/agent-ca-key.pem" 4096

openssl req -new -key "$CERTS_DIR/ca/intermediate/agent-ca-key.pem" \
    -out "$CERTS_DIR/ca/intermediate/agent-ca.csr" \
    -subj "/C=US/ST=California/L=San Francisco/O=ReignX/CN=ReignX Agent Intermediate CA"

# Create extension file for intermediate CA
cat > "$CERTS_DIR/ca/intermediate/agent-ca-ext.cnf" <<EOF
basicConstraints = CA:TRUE
keyUsage = keyCertSign, cRLSign
subjectKeyIdentifier = hash
EOF

openssl x509 -req -days 1825 \
    -in "$CERTS_DIR/ca/intermediate/agent-ca.csr" \
    -CA "$CERTS_DIR/ca/root/ca-cert.pem" \
    -CAkey "$CERTS_DIR/ca/root/ca-key.pem" \
    -CAcreateserial \
    -out "$CERTS_DIR/ca/intermediate/agent-ca-cert.pem" \
    -extfile "$CERTS_DIR/ca/intermediate/agent-ca-ext.cnf"

echo "✓ Generated Agent Intermediate CA"

# Create CA bundle for verification
cat "$CERTS_DIR/ca/intermediate/server-ca-cert.pem" \
    "$CERTS_DIR/ca/root/ca-cert.pem" > "$CERTS_DIR/ca/server-ca-bundle.pem"

cat "$CERTS_DIR/ca/intermediate/agent-ca-cert.pem" \
    "$CERTS_DIR/ca/root/ca-cert.pem" > "$CERTS_DIR/ca/agent-ca-bundle.pem"

echo "✓ Created CA bundles"

# Generate initial server certificate (for API server)
echo "Generating API server certificate..."
openssl genrsa -out "$CERTS_DIR/server/apiserver-key.pem" 2048

openssl req -new -key "$CERTS_DIR/server/apiserver-key.pem" \
    -out "$CERTS_DIR/server/apiserver.csr" \
    -subj "/C=US/ST=California/L=San Francisco/O=ReignX/CN=reignx-apiserver"

# Create extension file for server certificate
cat > "$CERTS_DIR/server/apiserver-ext.cnf" <<EOF
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = reignx-apiserver
DNS.3 = reignx-apiserver.default.svc.cluster.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

openssl x509 -req -days 365 \
    -in "$CERTS_DIR/server/apiserver.csr" \
    -CA "$CERTS_DIR/ca/intermediate/server-ca-cert.pem" \
    -CAkey "$CERTS_DIR/ca/intermediate/server-ca-key.pem" \
    -CAcreateserial \
    -out "$CERTS_DIR/server/apiserver-cert.pem" \
    -extfile "$CERTS_DIR/server/apiserver-ext.cnf"

# Create server certificate bundle
cat "$CERTS_DIR/server/apiserver-cert.pem" \
    "$CERTS_DIR/ca/intermediate/server-ca-cert.pem" \
    "$CERTS_DIR/ca/root/ca-cert.pem" > "$CERTS_DIR/server/apiserver-bundle.pem"

echo "✓ Generated API server certificate"

# Set proper permissions
chmod 600 "$CERTS_DIR"/ca/root/*-key.pem
chmod 600 "$CERTS_DIR"/ca/intermediate/*-key.pem
chmod 600 "$CERTS_DIR"/server/*-key.pem
chmod 644 "$CERTS_DIR"/ca/root/*-cert.pem
chmod 644 "$CERTS_DIR"/ca/intermediate/*-cert.pem
chmod 644 "$CERTS_DIR"/server/*-cert.pem

echo "✓ Set certificate permissions"

# Clean up CSR files
rm -f "$CERTS_DIR"/ca/intermediate/*.csr
rm -f "$CERTS_DIR"/server/*.csr

echo "✓ Cleaned up temporary files"

echo ""
echo "PKI Initialization Complete!"
echo "=============================="
echo "Root CA:           $CERTS_DIR/ca/root/ca-cert.pem"
echo "Server CA:         $CERTS_DIR/ca/intermediate/server-ca-cert.pem"
echo "Agent CA:          $CERTS_DIR/ca/intermediate/agent-ca-cert.pem"
echo "API Server Cert:   $CERTS_DIR/server/apiserver-cert.pem"
echo ""
echo "CA Bundles for verification:"
echo "Server CA Bundle:  $CERTS_DIR/ca/server-ca-bundle.pem"
echo "Agent CA Bundle:   $CERTS_DIR/ca/agent-ca-bundle.pem"
echo ""
echo "IMPORTANT: Keep the Root CA private key secure!"
echo "  Location: $CERTS_DIR/ca/root/ca-key.pem"
