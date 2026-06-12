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

FROM base AS platform-go-clone
ARG IAG_META_REF=main
ARG IAG_META_REPO=https://github.com/AlexanderKiyingi/IAG_multi_backend.git
# The meta-repo is private, so an anonymous clone fails in CI with
# "could not read Username for 'https://github.com'". Provide a GitHub token as
# a BuildKit secret (id=gh_token) and it is injected into the clone URL without
# ever landing in an image layer. When no secret is mounted (e.g. a public/
# monorepo build) the clone falls back to the plain anonymous URL.
RUN --mount=type=secret,id=gh_token \
    set -e; \
    CLONE_URL="${IAG_META_REPO}"; \
    if [ -s /run/secrets/gh_token ]; then \
      CLONE_URL=$(printf '%s' "${IAG_META_REPO}" | sed "s#https://#https://x-access-token:$(cat /run/secrets/gh_token)@#"); \
    fi; \
    git clone --depth 1 --branch "${IAG_META_REF}" "${CLONE_URL}" /tmp/iag \
    && mv /tmp/iag/shared/platform-go "${PLATFORM_GO_DEP}" \
    && rm -rf /tmp/iag

FROM base AS platform-go-copy
COPY shared/platform-go ${PLATFORM_GO_DEP}

FROM base AS build-standalone
COPY --from=platform-go-clone ${PLATFORM_GO_DEP} ${PLATFORM_GO_DEP}
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod edit -replace=github.com/alvor-technologies/iag-platform-go=${PLATFORM_GO_DEP} \
    && go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /dms .

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
