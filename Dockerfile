FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /lan-a2a ./cmd/lan-a2a && \
    CGO_ENABLED=0 go build -o /lan-relay ./cmd/lan-relay

FROM alpine:3.20 AS agent
COPY --from=builder /lan-a2a /usr/local/bin/lan-a2a
EXPOSE 19100
ENTRYPOINT ["lan-a2a"]

FROM alpine:3.20 AS relay
RUN apk add --no-cache ca-certificates
COPY --from=builder /lan-relay /usr/local/bin/lan-relay
EXPOSE 19200 19201
ENTRYPOINT ["lan-relay"]
