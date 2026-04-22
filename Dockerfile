# Minimal Dockerfile for OpenPass
# Uses scratch base for smallest possible image
# Build context: repository root with goreleaser-built binary

FROM scratch

# Copy CA certificates for HTTPS operations (git push/pull)
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary built by GoReleaser
COPY openpass /usr/bin/openpass

# OpenPass stores vault data in ~/.openpass by default
# In container context, users should mount a volume:
#   docker run -v ~/.openpass:/root/.openpass ghcr.io/danieljustus/openpass:latest
VOLUME ["/root/.openpass"]

ENTRYPOINT ["/usr/bin/openpass"]
CMD ["--help"]
