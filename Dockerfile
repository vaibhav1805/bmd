# BMD Multi-Stage Dockerfile
# Stage 1: Build the bmd binary
# Stage 2: Create knowledge tar from docs/ (optional)
# Stage 3: Final minimal image with binary + knowledge

# ─── Stage 1: Build ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -trimpath -o /bmd ./cmd/bmd

# ─── Stage 2: Package knowledge (optional) ───────────────────────────────────
# This stage pre-builds the knowledge index from docs/ if present.
# Skip this stage if you mount docs at runtime instead.
FROM builder AS knowledge-builder

# Copy docs if they exist (build arg to override path).
ARG DOCS_PATH=./docs
COPY ${DOCS_PATH} /docs

# Build knowledge tar from docs directory.
# If docs/ is empty or missing, this produces a minimal archive.
RUN mkdir -p /output && \
    if [ -d /docs ] && [ "$(find /docs -name '*.md' | head -1)" ]; then \
      /bmd export --from /docs --output /output/knowledge.tar.gz; \
    else \
      echo '{"version":"1.0.0","file_count":0}' > /tmp/knowledge.json && \
      tar czf /output/knowledge.tar.gz -C /tmp knowledge.json; \
    fi

# ─── Stage 3: Final image ────────────────────────────────────────────────────
FROM alpine:3.20

# Minimal runtime dependencies.
RUN apk add --no-cache ca-certificates

# Copy binary.
COPY --from=builder /bmd /usr/local/bin/bmd

# Copy pre-built knowledge tar (from Stage 2).
COPY --from=knowledge-builder /output/knowledge.tar.gz /knowledge.tar.gz

# Create non-root user for security.
RUN addgroup -S bmd && adduser -S bmd -G bmd
USER bmd

# Working directory for runtime data.
WORKDIR /data

# Health check: verify binary is functional.
HEALTHCHECK --interval=30s --timeout=5s --start-period=3s --retries=3 \
  CMD ["/usr/local/bin/bmd", "--help"]

# Default: headless MCP server with pre-built knowledge.
ENTRYPOINT ["/usr/local/bin/bmd"]
CMD ["serve", "--mcp", "--headless", "--knowledge-tar", "/knowledge.tar.gz"]

# Labels for container registries.
LABEL org.opencontainers.image.title="BMD - Beautiful Markdowns" \
      org.opencontainers.image.description="Terminal documentation platform with knowledge graphs and agent intelligence" \
      org.opencontainers.image.source="https://github.com/vaibhav1805/bmd" \
      org.opencontainers.image.licenses="MIT"
