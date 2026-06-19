FROM golang:1.26.4-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION
ARG COMMIT=docker-compose

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN VERSION="${VERSION:-$(cat VERSION)}" \
    && CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -v -ldflags "-X cloudiac/common.VERSION=$VERSION -X cloudiac/common.BUILD=$COMMIT" -o /out/ct-runner ./cmds/runner

FROM cloudiac/base-ct-runner:v0.2.1

ARG WORKER_IMAGE=cloudiac/ct-worker:latest
EXPOSE 19030

COPY configs/config-runner.yml.sample config-runner.yml
COPY --from=builder /out/ct-runner ct-runner

# runner 容器化部署时不能设置 assets_path
# 同时如果指定了 WORKER_IMAGE，则做配置替换
RUN sed -i -e 's/ assets_path:.*/ assets_path: ""/' \
           -e 's#cloudiac/ct-worker:latest"#'$WORKER_IMAGE'"#g' config-runner.yml

CMD ./ct-runner
