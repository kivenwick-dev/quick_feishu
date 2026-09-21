# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS frontend-builder
WORKDIR /src

COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN --mount=type=cache,target=/root/.npm \
    cd frontend && npm ci

COPY frontend ./frontend
RUN mkdir -p internal/server/web && cd frontend && npm run build

FROM golang:1.26.4-alpine AS backend-builder
WORKDIR /src

ARG ALPINE_MIRROR=https://mirrors.aliyun.com/alpine
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY} \
    GOSUMDB=${GOSUMDB}

RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}#g" /etc/apk/repositories
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
COPY --from=frontend-builder /src/internal/server/web ./internal/server/web
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/quick-feishu .

FROM alpine:3.22 AS runtime

ARG ALPINE_MIRROR=https://mirrors.aliyun.com/alpine
RUN sed -i "s#https://dl-cdn.alpinelinux.org/alpine#${ALPINE_MIRROR}#g" /etc/apk/repositories
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 quickfeishu \
    && adduser -S -D -H -u 10001 -G quickfeishu quickfeishu \
    && mkdir -p /app/config /app/data \
    && chown -R quickfeishu:quickfeishu /app

WORKDIR /app
COPY --from=backend-builder --chown=quickfeishu:quickfeishu /out/quick-feishu /app/quick-feishu

ENV TZ=Asia/Shanghai \
    QF_CONFIG_FILE=/app/config/config.yaml \
    QF_DATA_DIR=/app/data

VOLUME ["/app/config", "/app/data"]
EXPOSE 8080

USER quickfeishu

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/api/dashboard || exit 1

ENTRYPOINT ["/app/quick-feishu"]
CMD ["serve"]
