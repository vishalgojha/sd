FROM node:22-alpine

WORKDIR /app

# Coolify's healthcheck probe looks for curl first, so make sure it is present.
RUN apk add --no-cache curl

COPY package.json ./
COPY server.mjs ./
COPY shared ./shared
COPY public ./public

ENV NODE_ENV=production
ENV PORT=3000

EXPOSE 3000

HEALTHCHECK --interval=15s --timeout=4s --start-period=3s --retries=3 \
  CMD curl -fsS http://127.0.0.1:${PORT}/healthz || exit 1

CMD ["node", "server.mjs"]