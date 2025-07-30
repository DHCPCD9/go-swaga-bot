FROM golang:1.24.2-alpine as builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bot

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bot /app/bot

WORKDIR /app
ENTRYPOINT ["/app/bot"]