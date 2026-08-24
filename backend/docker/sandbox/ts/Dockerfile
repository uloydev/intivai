# Sandbox execution image: TypeScript (ADR-0002) — node base + tsx runner
# preinstalled (no network needed at run time).
FROM node:22.23.2-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32
RUN adduser -D -u 10001 sandbox && npm install -g tsx typescript
USER sandbox
WORKDIR /tmp
