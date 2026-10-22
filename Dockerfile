FROM golang:1.27-alpine@sha256:d530cf60723abf1ffae0cd16bb546b379ec0550d2f3815e5b16ed69dbc00fba4 AS builder

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
COPY --from=alpine:latest@sha256:c6b073f8e9e58d881f6ffba6ebf3b9ebd60b9e7d6e8337018c0e68bb5c356873 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "/plausible-exporter" ]
