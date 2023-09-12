
##
## Build
##
FROM golang:1.18.10-buster AS build

ENV GOPROXY https://goproxy.cn,direct

WORKDIR /app

COPY . ./

RUN go build -ldflags -s -v -o unipay_svr cmd/main.go
RUN go build -ldflags -s -v -o refund_svr cmd/refund/main.go

##
## Deploy
##
FROM ubuntu

ARG TZ=Asia/Shanghai

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y ca-certificates tzdata \
    && ln -fs /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo ${TZ} > /etc/timezone \
    && dpkg-reconfigure tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/unipay_svr /app/unipay_svr
COPY --from=build /app/refund_svr /app/refund_svr
COPY --from=build /app/config/config.example.yaml /app/config/config.yaml

ENTRYPOINT ["/app/unipay_svr", "--config", "/app/config/config.yaml"]