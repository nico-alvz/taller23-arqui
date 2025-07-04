FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o api-gateway .

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/api-gateway .

EXPOSE 8080

CMD ["./api-gateway"]
