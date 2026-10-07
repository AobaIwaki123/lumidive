# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary for target architecture
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-w -s" \
    -o /lumidive ./cmd/lumidive

# Runtime stage (Distroless for ultra-lightweight and secure container)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /lumidive /lumidive

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/lumidive"]
CMD ["server", "--port", "8080"]
