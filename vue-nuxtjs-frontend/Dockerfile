# Stage 1: Install dependencies & build Nuxt 3
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
RUN npm run build

# Stage 2: Production runner (Nitro Server)
FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NITRO_PORT=3001
ENV NITRO_HOST=0.0.0.0
ENV NUXT_PUBLIC_API_BASE_URL=http://localhost:8080

COPY --from=builder /app/.output ./.output

EXPOSE 3001
CMD ["node", ".output/server/index.mjs"]
