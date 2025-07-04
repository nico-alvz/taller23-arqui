FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache protobuf git build-base

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
COPY . .
    
COPY go.mod go.sum ./
RUN go mod download


#Generar stubs Go en la misma ruta del proto
RUN protoc -I ./protos \
    --go_out=./pb           --go_opt=paths=source_relative \
    --go-grpc_out=./pb      --go-grpc_opt=paths=source_relative \
   ./protos/users.proto


RUN go build -o users-service .
EXPOSE 50051

CMD ["./users-service"]

