FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build
FROM node:22-alpine
RUN addgroup -g 1001 -S nodeapp && adduser -u 1001 -S nodeapp -G nodeapp
WORKDIR /app
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
COPY --from=build --chown=nodeapp:nodeapp /app/.next/standalone ./
COPY --from=build --chown=nodeapp:nodeapp /app/.next/static ./.next/static
USER nodeapp
EXPOSE 3000
CMD ["node","server.js"]
