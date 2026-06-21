FROM golang:1.22-alpine3.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o secrethub ./cmd/secrethub

FROM alpine:3.21
COPY --from=builder /app/secrethub /usr/local/bin/
EXPOSE 4949
CMD ["secrethub", "serve", "--host", "0.0.0.0"]
