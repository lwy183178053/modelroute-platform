ARG NODE_BUILDER_IMAGE=node:24-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd
ARG NGINX_RUNTIME_IMAGE=nginx:1.28-alpine@sha256:a8b39bd9cf0f83869a2162827a0caf6137ddf759d50a171451b335cecc87d236
FROM ${NODE_BUILDER_IMAGE} AS web-builder

WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

FROM ${NGINX_RUNTIME_IMAGE}
COPY deploy/gateway/default.conf /etc/nginx/conf.d/default.conf
COPY --from=web-builder /src/dist /usr/share/nginx/html
