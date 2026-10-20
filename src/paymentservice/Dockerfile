FROM node:22-bookworm AS build

RUN apt-get update \
    && apt-get install --no-install-recommends -y python3 make g++ \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
ENV NODE_ENV=production
COPY package*.json ./
RUN npm ci --omit=dev && npm cache clean --force
COPY . .

FROM gcr.io/distroless/nodejs22-debian12:nonroot
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/index.js /app/server.js /app/charge.js /app/logger.js ./
COPY --from=build /app/proto ./proto
EXPOSE 50051
ENTRYPOINT ["/nodejs/bin/node"]
CMD ["index.js"]
