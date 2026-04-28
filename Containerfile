FROM docker.io/golang:1.26-alpine AS builder

RUN apk add --no-cache binutils git make

WORKDIR /app
COPY . .

RUN make build

FROM docker.io/alpine:latest

WORKDIR /app

COPY --from=builder /app/hottake .
