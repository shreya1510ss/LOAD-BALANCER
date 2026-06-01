FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o server1 .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server1 .
EXPOSE 8081
CMD ["./server1"]
