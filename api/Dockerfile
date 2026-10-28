# Build context is the repository root (so content/ is available):
#   docker build -f api/Dockerfile .
# Cross-compiles on the build host for each target platform, so multi-arch
# images build fast without emulation.
FROM --platform=$BUILDPLATFORM golang:1.27 AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN for cmd in api migrate content; do \
      CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/$cmd ./cmd/$cmd || exit 1; \
    done

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/migrate /out/content /usr/local/bin/
COPY api/db/migrations /migrations
COPY content /content
ENV MIGRATIONS_DIR=/migrations
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
