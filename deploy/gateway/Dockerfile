FROM node:24-alpine AS web-builder

WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ ./
RUN npm run build

FROM nginx:1.28-alpine
COPY deploy/gateway/default.conf /etc/nginx/conf.d/default.conf
COPY --from=web-builder /src/dist /usr/share/nginx/html
