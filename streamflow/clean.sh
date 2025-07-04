#!/bin/bash

echo "🧹 Limpiando StreamFlow Microservices..."

echo "Deteniendo contenedores..."
docker-compose down

echo "Eliminando volúmenes..."
docker-compose down -v

echo "Eliminando imágenes..."
docker-compose down --rmi all

echo "Limpiando sistema Docker..."
docker system prune -f

echo "✅ Lim099