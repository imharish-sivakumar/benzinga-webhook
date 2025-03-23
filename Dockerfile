FROM golang:1.22 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o webhook-receiver ./cmd/main

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/webhook-receiver .

EXPOSE 8080
ENTRYPOINT ["./webhook-receiver"]
