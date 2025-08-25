# Distroless image for GoReleaser
# GoReleaser will provide the pre-built binary
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the pre-built binary
# GoReleaser will inject the binary here
COPY ottl /ottl

# Use nonroot user
USER nonroot:nonroot

# Set the entrypoint
ENTRYPOINT ["/ottl"]

# Default command
CMD ["--help"]