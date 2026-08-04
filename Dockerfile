FROM --platform=$BUILDPLATFORM oven/bun:1 AS dashboard
WORKDIR /src
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard /src/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X github.com/rivly/rivly/internal/buildinfo.version=${VERSION}" \
    -o /out/rivly ./cmd/rivly

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 rivly \
    && mkdir -p /data \
    && chown rivly:rivly /data

COPY --from=docker/compose-bin:v5.1.2 /docker-compose /usr/local/bin/docker-compose
COPY --from=build /out/rivly /usr/local/bin/rivly

USER rivly
WORKDIR /data
VOLUME /data

ENV RIVLY_ADDR=:8080 \
    RIVLY_DATABASE=/data/rivly.db \
    RIVLY_DATA=/data/rivly

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
    CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null || exit 1

ENTRYPOINT ["rivly"]
