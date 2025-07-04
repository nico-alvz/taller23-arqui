FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache protobuf-dev git build-base

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

COPY . .
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p pb && \
    protoc -I ./protos \
    -I /usr/include \
    --go_out=pb --go_opt=paths=source_relative \
    --go-grpc_out=pb --go-grpc_opt=paths=source_relative \
    ./protos/email.proto

RUN go build -o  email-service .
EXPOSE 50057 50058


CMD ["./email-service"]
