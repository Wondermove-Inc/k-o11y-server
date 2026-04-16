FROM golang:1.24.11 AS builder

WORKDIR /app

COPY . .

RUN go mod tidy -e
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/ko11y-core ./cmd/main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/ko11y-core /app/ko11y-core

EXPOSE 3001

CMD ["/app/ko11y-core"]
