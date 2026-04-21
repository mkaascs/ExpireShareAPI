FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk --no-cache add git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /expire-share ./cmd/expire-share/main.go

FROM alpine:3.21

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /expire-share .
COPY config/ ./config/
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["/app/expire-share"]