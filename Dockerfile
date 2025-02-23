# Build frontend
FROM oven/bun:1 as frontend-builder
WORKDIR /app
COPY web/ ./
RUN bun install && bun run build

# Build backend with goreleaser
FROM goreleaser/goreleaser:latest as backend-builder
WORKDIR /app
COPY . .
COPY --from=frontend-builder /app/dist ./web/dist
# Create dist directory first
RUN mkdir -p cmd/server/dist && \
    cp -r web/dist/* cmd/server/dist/ && \
    goreleaser build --single-target --snapshot --clean

# Final stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
# Copy the binary using the name from goreleaser.yaml
COPY --from=backend-builder /app/dist/*/youtube-video-summary ./app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]