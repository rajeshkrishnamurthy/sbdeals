FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sbd ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sbd-migrate ./cmd/migrate

FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app
RUN adduser -D -u 10001 sbd

COPY --from=builder /out/sbd /usr/local/bin/sbd
COPY --from=builder /out/sbd-migrate /usr/local/bin/sbd-migrate

USER sbd
EXPOSE 8080

ENTRYPOINT ["sbd"]
