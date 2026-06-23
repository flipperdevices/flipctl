FROM node:22-bookworm-slim AS web-build
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.22-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN go build -o /out/flipctld ./cmd/flipctld && go build -o /out/flipctl-tui ./cmd/flipctl-tui

FROM debian:bookworm-slim AS runtime-base
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=go-build /out/* /usr/local/bin/
COPY plugins plugins
COPY fakecmds fakecmds
COPY --from=web-build /src/web/dist web/dist
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=5 CMD curl -fsS http://localhost:8080/readyz >/dev/null || exit 1

FROM runtime-base AS runtime
USER nobody
CMD ["flipctld","-addr",":8080","-plugins","plugins","-web","web/dist"]

FROM runtime-base AS runtime-real-tools
USER root
RUN apt-get update \
    && apt-get install -y --no-install-recommends iputils-ping nmap \
    && rm -rf /var/lib/apt/lists/*
COPY plugins-real plugins-real
USER nobody
CMD ["flipctld","-addr",":18081","-plugins","plugins-real","-web","web/dist"]
