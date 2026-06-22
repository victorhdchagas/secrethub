FROM golang:1.22-alpine3.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o secrethub ./cmd/secrethub

FROM alpine:3.21
RUN addgroup -S -g 1000 secrethub \
 && adduser -S -G secrethub -u 1000 -h /home/secrethub -s /sbin/nologin secrethub \
 && mkdir -p /home/secrethub/.secrethub \
 && chown -R secrethub:secrethub /home/secrethub
COPY --from=builder /app/secrethub /usr/local/bin/
USER secrethub
EXPOSE 4949
CMD ["secrethub", "serve", "--host", "0.0.0.0"]