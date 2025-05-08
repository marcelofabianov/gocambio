FROM golang:1.24-alpine AS builder

RUN apk add --no-cache build-base sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY ./server/. ./
ENV CGO_ENABLED=1
RUN go build -o /app/server_app_built .

FROM alpine:latest

RUN apk add --no-cache sqlite-libs
WORKDIR /app
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder /app/server_app_built /usr/local/bin/server_app
RUN chmod +x /usr/local/bin/server_app
RUN chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080

CMD ["/usr/local/bin/server_app"]