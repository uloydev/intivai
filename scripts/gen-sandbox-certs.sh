#!/usr/bin/env bash
# gen-sandbox-certs.sh — local CA + mTLS certs for the sandbox sidecar
# (ADR-0002). Output: backend/.sandbox-certs/{ca,server,server-key,client,client-key}.pem
# Gitignored; regenerate per environment. Prod should swap in real secrets.
set -euo pipefail

cd "$(dirname "$0")/../backend"
OUT=.sandbox-certs
mkdir -p "$OUT"
cd "$OUT"

if [[ -f ca.pem && -f server.pem && -f client.pem ]]; then
  echo "certs already exist in backend/$OUT — delete them to regenerate"
  exit 0
fi

# CA
openssl genrsa -out ca-key.pem 2048 2>/dev/null
openssl req -x509 -new -key ca-key.pem -sha256 -days 3650 \
  -subj "/CN=intivai-sandbox-ca" -out ca.pem

# Server cert (sidecar): reachable as sandbox-sidecar (compose DNS) and localhost
openssl genrsa -out server-key.pem 2048 2>/dev/null
openssl req -new -key server-key.pem -subj "/CN=sandbox-sidecar" -out server.csr
cat > server-ext.cnf <<'EOF'
subjectAltName = DNS:sandbox-sidecar, DNS:localhost, IP:127.0.0.1
extendedKeyUsage = serverAuth
EOF
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 365 -sha256 -extfile server-ext.cnf -out server.pem 2>/dev/null

# Client cert (app)
openssl genrsa -out client-key.pem 2048 2>/dev/null
openssl req -new -key client-key.pem -subj "/CN=intivai-app" -out client.csr
cat > client-ext.cnf <<'EOF'
extendedKeyUsage = clientAuth
EOF
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -days 365 -sha256 -extfile client-ext.cnf -out client.pem 2>/dev/null

rm -f server.csr client.csr server-ext.cnf client-ext.cnf
chmod 600 ca-key.pem server-key.pem client-key.pem
echo "sandbox mTLS certs written to backend/$OUT (ca.pem, server.pem, server-key.pem, client.pem, client-key.pem)"
