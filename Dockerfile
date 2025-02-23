# Build frontend
FROM oven/bun:1 as frontend-builder
ARG VITE_API_URL=/api/v1
ENV VITE_API_URL=$VITE_API_URL
WORKDIR /app
COPY web/ ./
RUN bun install && bun run build

# Build backend with goreleaser
FROM goreleaser/goreleaser:latest as backend-builder
WORKDIR /app
COPY . .
COPY --from=frontend-builder /app/dist ./web/dist
RUN mkdir -p cmd/server/dist && \
    cp -r web/dist/* cmd/server/dist/ && \
    goreleaser build --single-target --snapshot --clean --skip before

# Final stage
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=backend-builder /app/dist/*/youtube-video-summary ./app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]