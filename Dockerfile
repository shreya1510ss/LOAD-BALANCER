FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o loadbalancer .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/loadbalancer .
EXPOSE 8080
CMD ["./loadbalancer"]
