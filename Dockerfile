FROM golang:1.11.5-alpine3.9 as builder


ARG APP_NAME

RUN apk add --virtual .build-deps \
    alpine-sdk \
    cmake \
    libssh2 libssh2-dev\
    git \
    xz \
    curl

WORKDIR /src

ENV GO111MODULE=on

ADD . .

RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o ${APP_NAME}_build main.go

FROM alpine:3.9

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /root

ARG APP_NAME

COPY --from=builder /src/${APP_NAME}_build .

CMD ["${APP_NAME}_build"]
