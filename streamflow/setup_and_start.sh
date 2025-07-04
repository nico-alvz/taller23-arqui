#!/bin/bash
set -e

echo "🔍 Verificando permisos para Docker..."
if ! docker ps >/dev/null 2>&1; then
    echo "❌ No tienes permisos para usar Docker como usuario normal."
    echo "➡️  Ejecuta: sudo usermod -aG docker $USER"
    echo "🧠 Luego cierra sesión y vuelve a entrar o corre: newgrp docker"
    exit 1
fi

echo "🔧 Verificando Go..."
if ! command -v go >/dev/null || [[ "$(go version | cut -d' ' -f3)" < "go1.21" ]]; then
    echo "⬇️  Instalando Go 1.21.9..."
    cd /tmp
    wget -q https://go.dev/dl/go1.21.9.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.21.9.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    source ~/.bashrc
    echo "✅ Go instalado."
else
    echo "✅ Go ya está instalado."
fi

echo "📦 Ejecutando go mod tidy en users-service..."
cd services/users
go mod tidy

echo "🚀 Iniciando servicios con Docker Compose..."
cd ../../
./start.sh
echo "✅ StreamFlow Microservices iniciados."