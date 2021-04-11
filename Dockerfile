FROM golang:1.16.3-alpine3.13 AS build_base
WORKDIR /tmp/wol
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -v
RUN go build -o ./out/wol .
FROM alpine:latest
RUN apk add ca-certificates
COPY --from=build_base /tmp/wol/out/wol /app/wol
EXPOSE 80
CMD ["/app/wol"]