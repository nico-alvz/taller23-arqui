#!/bin/bash

echo "📋 Logs de StreamFlow Microservices"
echo ""

if [ $# -eq 0 ]; then
    echo "Ver logs de todos los servicios:"
    docker compose logs -f
else
    echo "Ver logs del servicio: $1"
    docker compose logs -f $1
fi
