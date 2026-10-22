FROM golang:1.27-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS builder

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
COPY --from=alpine:latest@sha256:71e9b7bae160ba167fa17414401b94cbcdaf3c32feeab9298bd794a2a0614c35 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "/plausible-exporter" ]
