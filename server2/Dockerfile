FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o server2 .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server2 .
EXPOSE 8082
CMD ["./server2"]
