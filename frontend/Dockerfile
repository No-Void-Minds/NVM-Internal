# syntax=docker/dockerfile:1

FROM node:24-alpine AS build
WORKDIR /app

COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY . .
# Vite menanam nilai ini saat build; mengubahnya butuh build ulang. Kosong berarti tombol Dashboard tidak tampil
# dan ?kembali= hanya menerima path internal.
ARG VITE_DASHBOARD_URL=
ARG VITE_ALLOWED_RETURN_HOSTS=
ENV VITE_DASHBOARD_URL=$VITE_DASHBOARD_URL
ENV VITE_ALLOWED_RETURN_HOSTS=$VITE_ALLOWED_RETURN_HOSTS
RUN npm run build

FROM nginx:1.29-alpine
# Dirender ke /etc/nginx/conf.d/default.conf saat start, dengan NVM_GATEWAY_IP diisi (pola compro).
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY nginx-security-headers.conf /etc/nginx/security-headers.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
