FROM golang:1.23-alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o videos-service .

EXPOSE 50053 51053
CMD ["./videos-service"]