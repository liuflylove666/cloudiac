# syntax=docker/dockerfile:1.7

FROM golang:1.26.4-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION
ARG COMMIT=docker-compose

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./

COPY . .
RUN --mount=type=cache,id=cloudiac-portal-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=cloudiac-portal-gobuild,target=/root/.cache/go-build \
    go install github.com/swaggo/swag/cmd/swag@v1.8.8 \
    && swag init -g portal/web/api/v1/route.go \
    && VERSION="${VERSION:-$(cat VERSION)}" \
    && CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -v -ldflags "-X cloudiac/common.VERSION=$VERSION -X cloudiac/common.BUILD=$COMMIT" -o /out/iac-portal ./cmds/portal \
    && CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" go build -v -ldflags "-X cloudiac/common.VERSION=$VERSION -X cloudiac/common.BUILD=$COMMIT" -o /out/iac-tool ./cmds/tool

FROM cloudiac/base-iac-portal:v0.2.1

EXPOSE 9030

COPY configs/config-portal.yml.sample config-portal.yml
COPY configs/dotenv.sample .
COPY --from=builder /out/iac-portal .
COPY --from=builder /out/iac-tool .

CMD  ./iac-portal
