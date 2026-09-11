# Estágio 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Copia os arquivos de configuração de dependências
COPY package.json package-lock.json ./

# Instala as dependências (ambiente limpo ignorando node_modules local graças ao .dockerignore)
RUN npm ci

# Copia o restante do código fonte
COPY . .

# Argumentos recebidos do docker-compose no momento do build.
# O Vite precisa dessas variáveis disponíveis durante o npm run build
# para injetá-las no bundle final.
ARG VITE_EVENT_API_URL
ARG VITE_ALARM_API_URL

ENV VITE_EVENT_API_URL=$VITE_EVENT_API_URL
ENV VITE_ALARM_API_URL=$VITE_ALARM_API_URL

# Gera os arquivos estáticos na pasta /app/dist
RUN npm run build

# Estágio 2: Nginx
FROM nginx:alpine

# Remove os arquivos padrão do Nginx
RUN rm -rf /usr/share/nginx/html/*

# Copia os arquivos estáticos gerados no estágio anterior
COPY --from=builder /app/dist /usr/share/nginx/html

# Expõe a porta padrão do Nginx
EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]