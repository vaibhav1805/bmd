FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -o bmd ./cmd/bmd

FROM alpine:latest

RUN apk add --no-cache bash curl python3 py3-pip

COPY --from=builder /app/bmd /usr/local/bin/bmd

# Install PageIndex for semantic search
RUN pip install --no-cache-dir --break-system-packages pageindex

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD bmd serve --mcp || exit 1

ENTRYPOINT ["bmd"]
CMD ["serve", "--mcp"]
