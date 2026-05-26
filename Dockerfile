# syntax=docker/dockerfile:1.7
# Build from monorepo root:
#   docker build -f services/operations/dms/Dockerfile .
FROM golang:1.25-alpine AS build

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY shared/platform-go /src/shared/platform-go

WORKDIR /src/services/operations/dms
COPY services/operations/dms/go.mod services/operations/dms/go.sum ./
RUN go mod download

COPY services/operations/dms/ .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /dms .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata wget
WORKDIR /app
COPY --from=build /dms /app/dms
COPY services/operations/dms/index.html /app/index.html
COPY services/operations/dms/assets /app/assets

ENV PORT=4010 \
    GIN_MODE=release \
    LOG_FORMAT=json \
    AUTH_MODE=jwt \
    AUTO_MIGRATE=false \
    SEED_ON_EMPTY=false

EXPOSE 4010
HEALTHCHECK --interval=15s --timeout=5s --start-period=25s --retries=5 \
  CMD wget -q -O /dev/null http://127.0.0.1:4010/ready || exit 1
USER nobody
ENTRYPOINT ["/app/dms"]
