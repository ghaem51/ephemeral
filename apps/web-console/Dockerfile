FROM node:24-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .

ARG VITE_API_BASE_URL=
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM node:24-alpine

WORKDIR /app
COPY --from=build /app/dist ./dist
COPY server.mjs ./server.mjs

ENV PORT=3000 \
    API_PROXY_URL=http://control-plane:8080

USER node
EXPOSE 3000
CMD ["node", "server.mjs"]
