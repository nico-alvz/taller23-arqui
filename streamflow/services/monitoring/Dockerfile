FROM golang:1.23-bookworm AS builder

WORKDIR /app

# Install protoc compiler and dependencies
RUN apt-get update && apt-get install -y protobuf-compiler git build-essential

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .


RUN protoc -I ./protos \
    --go_out=./pb           --go_opt=paths=source_relative \
    --go-grpc_out=./pb      --go-grpc_opt=paths=source_relative \
   ./protos/monitoring.proto

RUN go build -o monitoring-service .
EXPOSE 50054

CMD ["./monitoring-service"]

