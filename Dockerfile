ARG GO_IMAGE=golang
ARG GO_IMAGE_VERSION=1.20-alpine3.17
ARG ALPINE_IMAGE_TAG=alpine:3.17

FROM ${GO_IMAGE}:${GO_IMAGE_VERSION} AS build_step
LABEL maintainer=alkorgun@gmail.com

# ARG GOPROXY
ARG ALPINE_REPO=https://mirror.yandex.ru/mirrors/alpine/v3.17/main/
# ARG GIN_MODE=release
# ARG GOGC=30

WORKDIR /opt/app

RUN apk update -X ${ALPINE_REPO} && \
    apk add --no-cache -X ${ALPINE_REPO} git gcc pkgconf musl-dev

COPY . .

# ENV GOPROXY=${GOPROXY}

RUN go mod download

# ENV CGO_ENABLED=0

# RUN go build -o service ./cmd/main.go

RUN go build -o service -tags=alpine-linux,musl ./cmd/main.go

FROM ${ALPINE_IMAGE_TAG}

WORKDIR /opt/app

COPY --from=build_step /opt/app/service .

USER daemon

# ENV GIN_MODE=${GIN_MODE}
# ENV GOGC=${GOGC}

# EXPOSE 8000

CMD ["./service"]
