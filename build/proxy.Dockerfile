# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache module downloads separately from source code.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree.
COPY . .

# Build a statically linked binary with debug info stripped.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/proxy \
    ./cmd/proxy

# ── Final stage ───────────────────────────────────────────────────────────────
# Use distroless/static for a minimal, non-root image with no shell.
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/proxy /proxy

EXPOSE 8080

ENTRYPOINT ["/proxy"]
