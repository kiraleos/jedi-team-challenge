FROM golang:1.24.3-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags="-s -w" -o /jedi-team-challenge ./cmd/server/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /jedi-team-challenge .

EXPOSE 8080

ENTRYPOINT ["./jedi-team-challenge"]