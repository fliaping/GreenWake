# 构建前端
FROM node:18-alpine AS web_builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm install
COPY web/ .
RUN npm run build

# 构建后端
FROM golang:alpine AS build_base
WORKDIR /tmp/wol
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -v
RUN go build -o ./out/wol ./cmd/server

# 最终镜像
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

WORKDIR /app
COPY --from=build_base /tmp/wol/out/wol ./wol
COPY --from=web_builder /web/dist ./web/dist
COPY config.yml ./

EXPOSE 8055
CMD ["/app/wol"]