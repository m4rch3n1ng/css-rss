FROM golang:alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY main.go .

RUN go build .

FROM alpine

WORKDIR /app

COPY --from=build /app/css-rss /css-rss

ENTRYPOINT ["/css-rss"]

