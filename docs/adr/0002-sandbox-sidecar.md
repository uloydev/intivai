# ADR-0002: Sandbox code execution via gRPC sidecar

Candidate/recruiter code runs in throwaway Docker containers, not on the app host. A dedicated **sandbox sidecar** service owns the Docker socket and exposes a gRPC `Execute` RPC (mTLS, internal compose network, no published ports); the app holds no docker access.

ProcessRunner (bare `exec` with rlimits) was deleted: host execution is a root-equivalent attack surface on the app container, and the runtimes need per-language images anyway (go, python, node, ts — one image per language). The sidecar applies `--network=none`, memory/pids/cpu caps, read-only root with tmpfs workdir, and a per-run timeout; results are unary (no streaming) reusing the existing ExecutionRequest/ExecutionResult shapes. Certificates are script-generated per environment and gitignored.

**Consequences**: dev/CI must run the sidecar container for sandbox features; a Docker outage makes sandbox execution fail closed (no fallback path).
