FROM golang:1.26-alpine AS build

WORKDIR /src

COPY nitro-enclave/go.mod nitro-enclave/go.sum ./
RUN go mod download

COPY nitro-enclave/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/attestation ./cmd/attestation

FROM scratch

COPY --from=build /out/attestation /attestation

ENV VSOCK_PORT=5000

ENTRYPOINT ["/attestation"]
