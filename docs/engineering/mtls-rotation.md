# Sandbox mTLS Certificate Rotation

> Status: current · Last-reviewed: 2026-08-24 · Owner: EM
> Certificates are script-generated per environment and gitignored
> (`scripts/gen-sandbox-certs.sh`, ADR-0002).

## Topology

| Cert | CN | Validity | Source |
|---|---|---|---|
| CA | `intivai-sandbox-ca` | **10 years** (`-days 3650`, gen-sandbox-certs.sh:19) | self-signed RSA 2048, SHA-256 |
| Server (sidecar `sandbox-sidecar`) | SANs: `sandbox-sidecar`, `localhost`, `127.0.0.1` | **365 days** | EKU serverAuth |
| Client (app) | `intivai-app` | **365 days** | EKU clientAuth |

Trust chain: app verifies the sidecar's server cert against `INTIVAI_SANDBOX_CA_CERT`; sidecar verifies the app's client cert against the same CA. Files live in `backend/.sandbox-certs/` (mode 600 for keys), mounted read-only into both containers.

## Routine rotation (leaf certs)

Leaves expire every 365 days — calendar a renewal ~30 days before expiry:

```bash
cd /opt/intivai   # or repo root locally
rm backend/.sandbox-certs/server.pem backend/.sandbox-certs/server-key.pem \
   backend/.sandbox-certs/client.pem backend/.sandbox-certs/client-key.pem
bash scripts/gen-sandbox-certs.sh          # regenerates missing leaves against existing CA
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml up -d app sandbox-sidecar
curl -fsS https://<domain>/health           # verify mTLS handshake end-to-end
```

The script skips regeneration when all files exist, so only the deleted leaves are renewed; the CA is untouched. Distribute by redeploying the two containers (certs are bind-mounted from the host path, not baked into images). For multi-host deployments, copy `backend/.sandbox-certs/` to each host over a secured channel before re-up.

Check expiry proactively:

```bash
openssl x509 -enddate -noout -in backend/.sandbox-certs/server.pem
```

## CA expiry / planned CA rotation

CA lifetime is 10 years. Rotate by generating a NEW CA + leaves, deploying both containers together (they share one trust root — staggered rollout breaks the handshake), then retiring the old CA. Do this at year ~9, not after expiry.

## Compromise playbook (skeleton)

1. **Declare**: suspected key theft of any `.sandbox-certs/*.pem` on any host.
2. **Contain**: stop sandbox execution (`docker compose ... stop sandbox-sidecar app`) — sandbox features fail closed by design.
3. **Rotate everything** (compromise of a leaf implies CA distrust):
   ```bash
   rm -rf backend/.sandbox-certs && bash scripts/gen-sandbox-certs.sh
   ```
4. **Redeploy** app + sidecar on every affected host with the new material.
5. **Review access**: audit who/what read the host cert directory (SSH logs, container mounts); rotate the host credentials used.
6. **Postmortem**: record in `docs/FINDINGS.md`; if the compromise path was via the docker socket or socket-proxy, revisit ADR-0002 amendment controls.
