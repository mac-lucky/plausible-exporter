FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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
COPY --from=alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "/plausible-exporter" ]
