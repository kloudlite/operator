# context-dir: <project-root>
FROM golang:alpine3.19 AS base
# RUN apk add make gcc libc-dev
USER root
ENV GOPATH=/tmp/go
ENV GOCACHE=/tmp/go-cache
WORKDIR /tmp/app
# COPY --chown=1001 ./go.mod ./go.sum ./tools.go pkg common grpc-interfaces ./
# COPY --chown=1001 --from=project-root ./go.mod ./go.sum ./
# RUN go mod download
COPY --chown=1001 ./ ./
ARG APP_DIR
WORKDIR $APP_DIR
ENV CGO_ENABLED=0
RUN go build -tags musl -o /tmp/bin/wg-restart ./main.go
RUN chmod +x /tmp/bin/wg-restart

FROM ghcr.io/linuxserver/wireguard:latest
WORKDIR /tmp/bin
COPY --from=base /tmp/bin/wg-restart ./
ENTRYPOINT ["./wg-restart"]
