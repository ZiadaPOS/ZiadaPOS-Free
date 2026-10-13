FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /ziada-api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Africa/Dar_es_Salaam

WORKDIR /app
COPY --from=builder /ziada-api /app/ziada-api

EXPOSE 8080
ENTRYPOINT ["/app/ziada-api"]
