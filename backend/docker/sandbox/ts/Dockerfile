# Sandbox execution image: TypeScript (ADR-0002) — node base + tsx runner
# preinstalled (no network needed at run time).
FROM node:22-alpine
RUN adduser -D -u 10001 sandbox && npm install -g tsx typescript
USER sandbox
WORKDIR /tmp
