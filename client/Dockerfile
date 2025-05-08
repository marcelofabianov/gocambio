FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./client/. ./

RUN go build -o /app/client_app_built .

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/client_app_built /usr/local/bin/client_app
RUN chmod +x /usr/local/bin/client_app

CMD ["/usr/local/bin/client_app"]