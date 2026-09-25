FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS builder

WORKDIR /go/src/app

# Install dependencies first for better caching
COPY go.mod go.sum /go/src/app/
RUN go get -v ./...

# Do a completely static build
COPY . .
RUN CGO_ENABLED=0 go install -ldflags '-s -w -extldflags "-static"' -tags timetzdata ./cmd
RUN ls -l /go/bin

FROM scratch AS runner

LABEL org.opencontainers.image.source="https://github.com/mac-lucky/plausible-exporter"
LABEL org.opencontainers.image.description="Prometheus exporter for Plausible Analytics stats"

# Copy the built binary
COPY --from=builder /go/bin/cmd /plausible-exporter
# Copy the CA root certificats from the latest alpine image
COPY --from=alpine:latest@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "/plausible-exporter" ]
