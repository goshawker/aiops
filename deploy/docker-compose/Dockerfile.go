FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .
ARG SERVICE_NAME
RUN CGO_ENABLED=1 GOOS=linux go build -o /app/bin/service ./cmd/${SERVICE_NAME}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/service /usr/local/bin/service
COPY configs/ /app/configs/

ENV SERVICE_NAME=gateway
WORKDIR /app
CMD ["sh", "-c", "service"]
