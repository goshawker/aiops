FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/service ./cmd/${SERVICE_NAME}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/service /usr/local/bin/service
COPY configs/ /etc/aiops/

ENV SERVICE_NAME=gateway
CMD ["sh", "-c", "service"]
