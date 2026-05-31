FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/server ./cmd/comment_post/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/server .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./server"]