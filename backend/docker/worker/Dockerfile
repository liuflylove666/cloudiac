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
    && CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -v -ldflags "-X cloudiac/common.VERSION=$VERSION -X cloudiac/common.BUILD=$COMMIT" -o /out/iac-tool ./cmds/tool

FROM cloudiac/base-ct-worker:v0.2.1

COPY assets/terraform.py /cloudiac/assets/terraform.py
COPY assets/terraformrc-* /cloudiac/assets/
COPY assets/cloudiac-playbook /usr/local/bin/cloudiac-playbook
COPY --from=builder /out/iac-tool /usr/yunji/cloudiac/iac-tool
