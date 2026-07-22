FROM golang:1.24-alpine AS builder

ARG APP_VERSION=dev
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/modelroute-api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/modelroute-worker ./cmd/worker

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S modelroute \
    && adduser -S -G modelroute modelroute
USER modelroute
COPY --from=builder /out/modelroute-api /usr/local/bin/modelroute-api
COPY --from=builder /out/modelroute-worker /usr/local/bin/modelroute-worker
ENTRYPOINT ["/usr/local/bin/modelroute-api"]
