FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache curl && \
    curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o teine-andres .

FROM alpine:3.19

WORKDIR /app
RUN apk add --no-cache ca-certificates openssh-client
COPY --from=builder /app/teine-andres .
COPY --from=builder /usr/local/bin/dbmate /usr/local/bin/dbmate
COPY --from=builder /app/db/migrations ./db/migrations
COPY --from=builder /app/prompts ./prompts

COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
