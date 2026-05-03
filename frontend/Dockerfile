# syntax=docker/dockerfile:1.7

# ----- Build stage --------------------------------------------------------
FROM node:20-alpine AS builder
WORKDIR /src
# package-lock.json должен лежать рядом с package.json — иначе `npm ci` упадёт.
# Сгенерировать локально один раз: `cd frontend && npm install` (создаст lock-файл).
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .

# VITE_*-переменные приходят как build args из docker-compose.
# Vite использует только значения, начинающиеся на VITE_, и вшивает их в собранный JS.
ARG VITE_BACKEND_URL=
ARG VITE_YANDEX_CAPTCHA_KEY=
ENV VITE_BACKEND_URL=$VITE_BACKEND_URL \
    VITE_YANDEX_CAPTCHA_KEY=$VITE_YANDEX_CAPTCHA_KEY

RUN npm run build

# ----- Runtime stage (nginx) ----------------------------------------------
FROM nginx:1.27-alpine
COPY --from=builder /src/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
