# 多阶段构建：server 与 worker 双二进制
FROM golang:1.26-alpine AS build
WORKDIR /app
# 国内网络直连 proxy.golang.org 常超时,构建阶段走 goproxy.cn 加速
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /app/bin/ /app/bin/
COPY configs/config.yaml /app/configs/
EXPOSE 8081
ENTRYPOINT ["/app/bin/server"]
