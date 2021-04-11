FROM golang:latest AS build_base

# Set the Current Working Directory inside the container
WORKDIR /tmp/my-wol

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build -o ./out/my-wol .

# Start fresh from a smaller image
FROM alpine:latest
RUN apk add ca-certificates

COPY --from=build_base /tmp/my-wol/out/my-wol /app/my-wol

# This container exposes port 8080 to the outside world
EXPOSE 80

# Run the binary program produced by `go install`
CMD ["/app/my-wol"]