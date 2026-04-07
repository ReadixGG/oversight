FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o oversight-server .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/oversight-server .

EXPOSE 8080

ENTRYPOINT ["./oversight-server"]
CMD ["-addr", ":8080", "-tick", "20"]
