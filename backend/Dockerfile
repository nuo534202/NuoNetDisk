FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=v0.1.0
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=${VERSION}" -o /server ./cmd/server/

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /server /server
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/server"]
