FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/mini-store-api ./cmd/api

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /out/mini-store-api /app/mini-store-api
COPY --from=builder /app/configs /app/configs

EXPOSE 8080

ENTRYPOINT ["/app/mini-store-api"]
