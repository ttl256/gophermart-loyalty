FROM golang:1.25-alpine3.22 as builder
LABEL stage=builder
RUN apk update && apk add --no-cache make
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build

FROM alpine:3.22 as server
WORKDIR /root
COPY --from=builder /app/bin/gophermart .
ENTRYPOINT ["./gophermart"]
