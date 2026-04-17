#!/bin/bash

# Script to generate development certificates for BM Distributed Solution
# This creates a complete PKI hierarchy for development/testing

set -e

CERTS_DIR="certs"
CA_DIR="${CERTS_DIR}/ca"
SERVER_DIR="${CERTS_DIR}/server"
AGENT_DIR="${CERTS_DIR}/agent"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Generating certificates for BM Distributed Solution${NC}"
echo ""

# Create directory structure
mkdir -p "${CA_DIR}" "${SERVER_DIR}" "${AGENT_DIR}"

# Generate Root CA
echo -e "${YELLOW}Generating Root CA...${NC}"
if [ ! -f "${CA_DIR}/ca.key" ]; then
    openssl genrsa -out "${CA_DIR}/ca.key" 4096
    openssl req -new -x509 -days 3650 -key "${CA_DIR}/ca.key" -out "${CA_DIR}/ca.crt" \
        -subj "/C=US/ST=State/L=City/O=BM Solution/OU=Development/CN=BM Root CA"
    echo -e "${GREEN}Root CA generated${NC}"
else
    echo "Root CA already exists, skipping..."
fi

# Generate Intermediate CA for Servers
echo -e "${YELLOW}Generating Intermediate CA for Servers...${NC}"
if [ ! -f "${CA_DIR}/intermediate-server.key" ]; then
    openssl genrsa -out "${CA_DIR}/intermediate-server.key" 4096
    openssl req -new -key "${CA_DIR}/intermediate-server.key" -out "${CA_DIR}/intermediate-server.csr" \
        -subj "/C=US/ST=State/L=City/O=BM Solution/OU=Development/CN=BM Server Intermediate CA"
    openssl x509 -req -days 1825 -in "${CA_DIR}/intermediate-server.csr" \
        -CA "${CA_DIR}/ca.crt" -CAkey "${CA_DIR}/ca.key" -CAcreateserial \
        -out "${CA_DIR}/intermediate-server.crt"
    echo -e "${GREEN}Server Intermediate CA generated${NC}"
else
    echo "Server Intermediate CA already exists, skipping..."
fi

# Generate Intermediate CA for Agents
echo -e "${YELLOW}Generating Intermediate CA for Agents...${NC}"
if [ ! -f "${CA_DIR}/intermediate-agent.key" ]; then
    openssl genrsa -out "${CA_DIR}/intermediate-agent.key" 4096
    openssl req -new -key "${CA_DIR}/intermediate-agent.key" -out "${CA_DIR}/intermediate-agent.csr" \
        -subj "/C=US/ST=State/L=City/O=BM Solution/OU=Development/CN=BM Agent Intermediate CA"
    openssl x509 -req -days 1825 -in "${CA_DIR}/intermediate-agent.csr" \
        -CA "${CA_DIR}/ca.crt" -CAkey "${CA_DIR}/ca.key" -CAcreateserial \
        -out "${CA_DIR}/intermediate-agent.crt"
    echo -e "${GREEN}Agent Intermediate CA generated${NC}"
else
    echo "Agent Intermediate CA already exists, skipping..."
fi

# Generate Server Certificate (API Server, Scheduler)
echo -e "${YELLOW}Generating Server Certificate...${NC}"
if [ ! -f "${SERVER_DIR}/server.key" ]; then
    openssl genrsa -out "${SERVER_DIR}/server.key" 2048
    openssl req -new -key "${SERVER_DIR}/server.key" -out "${SERVER_DIR}/server.csr" \
        -subj "/C=US/ST=State/L=City/O=BM Solution/OU=Development/CN=localhost"

    # Create SAN config
    cat > "${SERVER_DIR}/san.cnf" <<EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = apiserver
DNS.3 = scheduler
DNS.4 = *.local
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

    openssl x509 -req -days 365 -in "${SERVER_DIR}/server.csr" \
        -CA "${CA_DIR}/intermediate-server.crt" -CAkey "${CA_DIR}/intermediate-server.key" \
        -CAcreateserial -out "${SERVER_DIR}/server.crt" \
        -extensions v3_req -extfile "${SERVER_DIR}/san.cnf"

    # Create full chain
    cat "${SERVER_DIR}/server.crt" "${CA_DIR}/intermediate-server.crt" "${CA_DIR}/ca.crt" > "${SERVER_DIR}/server-chain.crt"

    echo -e "${GREEN}Server certificate generated${NC}"
else
    echo "Server certificate already exists, skipping..."
fi

# Generate Agent Certificate
echo -e "${YELLOW}Generating Agent Certificate...${NC}"
if [ ! -f "${AGENT_DIR}/agent.key" ]; then
    openssl genrsa -out "${AGENT_DIR}/agent.key" 2048
    openssl req -new -key "${AGENT_DIR}/agent.key" -out "${AGENT_DIR}/agent.csr" \
        -subj "/C=US/ST=State/L=City/O=BM Solution/OU=Development/CN=bm-agent-dev"
    openssl x509 -req -days 365 -in "${AGENT_DIR}/agent.csr" \
        -CA "${CA_DIR}/intermediate-agent.crt" -CAkey "${CA_DIR}/intermediate-agent.key" \
        -CAcreateserial -out "${AGENT_DIR}/agent.crt"

    # Create full chain
    cat "${AGENT_DIR}/agent.crt" "${CA_DIR}/intermediate-agent.crt" "${CA_DIR}/ca.crt" > "${AGENT_DIR}/agent-chain.crt"

    echo -e "${GREEN}Agent certificate generated${NC}"
else
    echo "Agent certificate already exists, skipping..."
fi

# Copy CA cert to top-level certs directory for easy access
cp "${CA_DIR}/ca.crt" "${CERTS_DIR}/ca.crt"
cp "${SERVER_DIR}/server.crt" "${CERTS_DIR}/server.crt"
cp "${SERVER_DIR}/server.key" "${CERTS_DIR}/server.key"
cp "${AGENT_DIR}/agent.crt" "${CERTS_DIR}/agent.crt"
cp "${AGENT_DIR}/agent.key" "${CERTS_DIR}/agent.key"

echo ""
echo -e "${GREEN}Certificate generation completed successfully!${NC}"
echo ""
echo "Certificate locations:"
echo "  Root CA: ${CA_DIR}/ca.crt"
echo "  Server Cert: ${SERVER_DIR}/server.crt"
echo "  Server Key: ${SERVER_DIR}/server.key"
echo "  Agent Cert: ${AGENT_DIR}/agent.crt"
echo "  Agent Key: ${AGENT_DIR}/agent.key"
echo ""
echo "Certificates are valid for:"
echo "  Root CA: 10 years"
echo "  Intermediate CAs: 5 years"
echo "  Server/Agent Certs: 1 year"
echo ""
echo -e "${YELLOW}WARNING: These certificates are for DEVELOPMENT ONLY!${NC}"
echo "Do NOT use in production environments."
