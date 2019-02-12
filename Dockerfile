FROM golang:1.11.5-alpine3.9 as builder

ARG APP_NAME

WORKDIR /${GOPATH}/src/${APP_NAME}

ADD . .

RUN apk add git curl

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep

RUN dep ensure -vendor-only

RUN go build -o /src/${APP_NAME}_build main.go

FROM alpine:3.9

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /root

ARG APP_NAME

COPY --from=builder /src/${APP_NAME}_build .

CMD ["./${APP_NAME}_build"]
