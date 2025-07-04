#!/bin/bash

echo "🚀 Iniciando StreamFlow Microservices..."

# Verificar Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker no está instalado"
    exit 1
fi

# Docker Compose is now integrated with Docker
if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose no está disponible"
    exit 1
fi

# Crear directorios necesarios
mkdir -p nginx/logs

# Iniciar servicios
echo "📦 Construyendo e iniciando contenedores..."
docker compose up --build -d

echo "⏳ Esperando que los servicios estén listos..."
sleep 30

# Verificar estado
echo "🔍 Verificando estado de servicios..."
docker compose ps

echo "✅ StreamFlow iniciado exitosamente!"
echo ""
echo "📋 Servicios disponibles:"
echo "  - Nginx Load Balancer: http://localhost (HTTPS: https://localhost)"
echo "  - API Gateway 1: http://localhost:8080"
echo "  - API Gateway 2: http://localhost:8081" 
echo "  - API Gateway 3: http://localhost:8082"
echo "  - Auth Service: http://localhost:8001"
echo "  - RabbitMQ Management: http://localhost:15672 (admin/password)"
echo ""
echo "🎭 Endpoint cómico: http://localhost/comedia"
echo ""
echo "👤 Usuario administrador por defecto:"
echo "  Email: admin@streamflow.com"
echo "  Contraseña: admin123"
