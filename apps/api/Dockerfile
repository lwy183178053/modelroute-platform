ARG GO_BUILDER_IMAGE=golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191
ARG ALPINE_RUNTIME_IMAGE=alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce
FROM ${GO_BUILDER_IMAGE} AS builder

ARG APP_VERSION=dev
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/modelroute-api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/modelroute-worker ./cmd/worker \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${APP_VERSION}" -o /out/modelroute-seed-models ./cmd/seed-models

FROM ${ALPINE_RUNTIME_IMAGE}
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S modelroute \
    && adduser -S -G modelroute modelroute
USER modelroute
COPY --from=builder /out/modelroute-api /usr/local/bin/modelroute-api
COPY --from=builder /out/modelroute-worker /usr/local/bin/modelroute-worker
COPY --from=builder /out/modelroute-seed-models /usr/local/bin/modelroute-seed-models
ENTRYPOINT ["/usr/local/bin/modelroute-api"]
