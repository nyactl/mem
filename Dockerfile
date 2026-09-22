# syntax=docker/dockerfile:1

# Build stages run on the build host's architecture and cross-compile, so a
# multi-arch image needs no emulation.
FROM --platform=$BUILDPLATFORM node:26-alpine3.24@sha256:dbaa92e5758cbbcf85d65d5403fdb530fe3442cbe8c6dbfb7ef23365450d5070 AS ui
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine3.24@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
ARG TARGETOS TARGETARCH VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=ui /src/internal/web/dist internal/web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/mem ./cmd/mem

FROM alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6
# git: mem serve commits every write to the notes directory.
RUN apk add --no-cache git \
 && addgroup -g 1000 mem \
 && adduser -D -u 1000 -G mem mem \
 && install -d -o mem -g mem /data
COPY --from=build /out/mem /usr/local/bin/mem
USER mem
ENV MEM_NOTES_DIR=/data/notes \
    MEM_ATTACHMENTS_DIR=/data/attachments \
    MEM_LISTEN_ADDR=:4747
EXPOSE 4747
HEALTHCHECK --interval=1m --timeout=3s --start-period=10s --retries=3 \
  CMD wget -q --spider http://127.0.0.1:4747/ || exit 1
ENTRYPOINT ["mem"]
CMD ["serve"]
