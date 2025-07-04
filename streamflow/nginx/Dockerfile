FROM nginx:alpine

# Copiar configuración
COPY nginx.conf /etc/nginx/nginx.conf

# Crear directorio para SSL
RUN mkdir -p /etc/nginx/ssl

# Crear certificados SSL autofirmados
RUN apk add --no-cache openssl && \
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/nginx/ssl/nginx.key \
    -out /etc/nginx/ssl/nginx.crt \
    -subj "/C=CL/ST=Antofagasta/L=Antofagasta/O=StreamFlow/OU=Development/CN=localhost"

# Crear directorio para logs
RUN mkdir -p /var/log/nginx

# Exponer puertos
EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
