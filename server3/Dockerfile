FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o server3 .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server3 .
EXPOSE 8083
CMD ["./server3"]
