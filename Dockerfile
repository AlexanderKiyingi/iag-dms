# syntax=docker/dockerfile:1.7
#
# Targets:
#   standalone (default) — iag-dms repo root on Railway
#   monorepo             — IAG_multi_backend root context (deploy/docker-compose)
#
# Monorepo:   docker build -f services/operations/dms/Dockerfile --target monorepo .
# Standalone: docker build --target standalone .

FROM golang:1.25-alpine AS base
RUN apk add --no-cache git ca-certificates
ENV PLATFORM_GO_DEP=/deps/platform-go

FROM base AS platform-go-copy
COPY shared/platform-go ${PLATFORM_GO_DEP}

FROM base AS build-standalone
# Standalone (iag-dms repo root): the meta-repo is private, so Railway can't
# clone it at build time (Metal builder has no credentials and no BuildKit secret
# mounts). Instead the standalone repo carries a committed snapshot at
# third_party/platform-go (refreshed via scripts/sync-platform-go.sh). Copy that
# into /deps/platform-go and point the replace directive at it.
WORKDIR /src
COPY third_party/platform-go ${PLATFORM_GO_DEP}
COPY go.mod go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && go mod download
COPY . .
ARG VERSION=dev
# `COPY . .` restored go.mod from the build context, which still carries the
# meta-repo-only `replace => ../../../shared/platform-go`. That path does not
# exist inside the build container, so re-apply the vendored replace before build.
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /dms .

FROM base AS build-monorepo
COPY --from=platform-go-copy ${PLATFORM_GO_DEP} ${PLATFORM_GO_DEP}
WORKDIR /src/services/operations/dms
COPY services/operations/dms/go.mod services/operations/dms/go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && go mod download
COPY services/operations/dms/ .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /dms .

FROM alpine:3.21 AS monorepo
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build-monorepo /dms /app/dms
COPY --from=build-monorepo /src/services/operations/dms/index.html /app/index.html
COPY --from=build-monorepo /src/services/operations/dms/assets /app/assets
ENV PORT=4010 \
    GIN_MODE=release \
    LOG_FORMAT=json \
    AUTO_MIGRATE=false \
    SEED_ON_EMPTY=false
EXPOSE 4010
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4010/ready || exit 1
USER nobody
ENTRYPOINT ["/app/dms"]

FROM alpine:3.21 AS standalone
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build-standalone /dms /app/dms
COPY --from=build-standalone /src/index.html /app/index.html
COPY --from=build-standalone /src/assets /app/assets
ENV PORT=4010 \
    GIN_MODE=release \
    LOG_FORMAT=json \
    AUTO_MIGRATE=false \
    SEED_ON_EMPTY=false
EXPOSE 4010
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4010/ready || exit 1
USER nobody
ENTRYPOINT ["/app/dms"]
