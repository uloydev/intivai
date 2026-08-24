# ADR-0002: Sandbox code execution via gRPC sidecar

Candidate/recruiter code runs in throwaway Docker containers, not on the app host. A dedicated **sandbox sidecar** service owns the Docker socket and exposes a gRPC `Execute` RPC (mTLS, internal compose network, no published ports); the app holds no docker access.

ProcessRunner (bare `exec` with rlimits) was deleted: host execution is a root-equivalent attack surface on the app container, and the runtimes need per-language images anyway (go, python, node, ts — one image per language). The sidecar applies `--network=none`, memory/pids/cpu caps, read-only root with tmpfs workdir, and a per-run timeout; results are unary (no streaming) reusing the existing ExecutionRequest/ExecutionResult shapes. Certificates are script-generated per environment and gitignored.

**Consequences**: dev/CI must run the sidecar container for sandbox features; a Docker outage makes sandbox execution fail closed (no fallback path).

## Amendment 2026-08-24: socket containment (FINDINGS A12)

The original design left the sidecar holding the raw `/var/run/docker.sock`, which is root-equivalent on the host — a sidecar compromise meant host compromise, contradicting this ADR's own threat model.

Prod (`docker-compose.prod.yml`) now interposes a least-privilege **socket proxy** (`tecnativa/docker-socket-proxy`, digest-pinned): only the proxy mounts the socket (read-only), the sidecar reaches the daemon via `DOCKER_HOST=tcp://socket-proxy:2375` with `CONTAINERS=1 IMAGES=1 POST=1`. This works without Go changes because sandboxd shells out to the docker CLI (`internal/sandbox/sidecar/runner.go` `execDocker`), and the CLI honors `DOCKER_HOST` natively; no API client is embedded. The proxy publishes no ports; it is internal-network only.

Post-deploy verification duty: confirm `docker run` / `docker kill` (timeout path) pass through the proxy's POST filter; if an endpoint is blocked, widen with explicit proxy `ALLOW_*` vars — never by re-mounting the socket into the sidecar. The dev compose still mounts the raw socket (acceptable for local development).
