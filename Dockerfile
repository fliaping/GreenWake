# 构建前端
FROM node:18-alpine AS web_builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm install
COPY web/ .
RUN npm run build

# 测试阶段
FROM golang:1.21-alpine AS tester
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go test -v

# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wol cmd/server/main.go

# 最终镜像
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /app
COPY --from=builder /app/wol ./wol
COPY --from=web_builder /web/dist ./web/dist
COPY config.yml ./

EXPOSE 8055
CMD ["/app/wol"]