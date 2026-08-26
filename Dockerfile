# 分架构基础镜像：amd64 构建传 GO_IMAGE=…-amd64（ACR 裸 tag 仅含 arm64）。
# 全局 ARG 必须位于首个 FROM 之前。
ARG GO_IMAGE=registry.cn-beijing.aliyuncs.com/withyou_holmose/golang:1.26-bookworm

FROM registry.cn-beijing.aliyuncs.com/withyou_holmose/node:22.19-bookworm-slim AS frontend-builder

WORKDIR /src/frontend

COPY frontend/out ./out

FROM ${GO_IMAGE} AS backend-builder

WORKDIR /src/backend

ARG GIT_COMMIT=unknown
ARG BUILD_TIME=""
COPY VERSION /src/VERSION
COPY backend/go.mod backend/go.sum ./

RUN apt-get update -o Acquire::AllowInsecureRepositories=true \
  && apt-get install -y --no-install-recommends --allow-unauthenticated libsqlite3-dev \
  && rm -rf /var/lib/apt/lists/*

ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMCHECK=github.com/openilink/*,github.com/skip2/*
ENV GONOSUMDB=github.com/openilink/*,github.com/skip2/*
ENV GOCACHE=/tmp/go-cache
RUN go mod download

COPY backend ./

RUN go clean -cache && go mod tidy && VERSION="$(cat /src/VERSION)" \
    && if [ -z "${BUILD_TIME}" ]; then BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"; fi \
    && CGO_ENABLED=1 \
       GOCACHE=/tmp/go-cache go build -trimpath \
       -ldflags="-s -w -X github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/buildinfo.Version=${VERSION} -X github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/buildinfo.Commit=${GIT_COMMIT} -X github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/buildinfo.BuildTime=${BUILD_TIME}" \
       -o /out/deeix-chat ./cmd/server


FROM registry.cn-beijing.aliyuncs.com/withyou_holmose/debian:bookworm-slim AS runtime-deps
RUN apt-get update -o Acquire::AllowInsecureRepositories=true \
    && apt-get install -y --no-install-recommends --allow-unauthenticated ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*


FROM registry.cn-beijing.aliyuncs.com/withyou_holmose/debian:bookworm-slim AS runtime

WORKDIR /app

COPY --from=runtime-deps /etc/ssl/certs /etc/ssl/certs
COPY --from=runtime-deps /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=runtime-deps /etc/localtime /etc/localtime
COPY --from=runtime-deps /etc/timezone /etc/timezone
COPY --from=backend-builder /out/deeix-chat /app/deeix-chat
COPY --from=frontend-builder /src/frontend/out /app/frontend/out
COPY LICENSE NOTICE /app/licenses/DEEIX-Chat/

ENV FRONTEND_DIST_DIR=/app/frontend/out

EXPOSE 8080

VOLUME ["/app/storage", "/app/data"]

CMD ["/app/deeix-chat"]
