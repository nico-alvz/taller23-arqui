# Guía de Despliegue - StreamFlow Microservicios
protoc --go_out=streamflow/services/users/pb --go-grpc_out=streamflow/servi
ces/users/pb -I ./protos ./users/videos.proto

protoc --go_out=streamflow/services/videos/pb --go-grpc_out=streamflow/services/videos/pb -I ./protos ./protos/videos.proto
## Requisitos del Sistema

### Hardware Mínimo
- **RAM**: 8GB disponible (recomendado 16GB)
- **CPU**: 4 cores (recomendado 8 cores)
- **Almacenamiento**: 20GB libres
- **Red**: Conexión a internet para descargar imágenes Docker

### Software Requerido
- **Docker**: versión 20.10 o superior
- **Docker Compose**: versión 2.0 o superior
- **Git**: para clonar el repositorio
- **Navegador web**: para acceder a la aplicación

### Puertos Utilizados
Asegúrese de que estos puertos estén disponibles:
- `80, 443`: Nginx (HTTP/HTTPS)
- `5432, 5433`: PostgreSQL (Auth y Playlists)
- `3306, 3307`: MySQL y MariaDB
- `27017`: MongoDB
- `5672, 15672`: RabbitMQ
- `8001`: Auth Service
- `8080-8082`: API Gateway (3 instancias)
- `50051-50057`: Microservicios gRPC

## Instalación y Configuración

### 1. Clonar el Repositorio

```bash
git clone <repository-url>
cd streamflow
```

### 2. Verificar Docker

```bash
# Verificar instalación de Docker
docker --version
docker-compose --version

# Verificar que Docker esté ejecutándose
docker ps
```

### 3. Configurar Variables de Entorno (Opcional)

Crear archivo `.env` si necesita personalizar configuraciones:

```bash
# .env
JWT_SECRET_KEY=streamflow_secret_key_2024
POSTGRES_PASSWORD=password
MYSQL_ROOT_PASSWORD=password
RABBITMQ_DEFAULT_USER=admin
RABBITMQ_DEFAULT_PASS=password
```

### 4. Iniciar el Sistema

#### Opción A: Script Automático (Recomendado)
```bash
./start.sh
```

#### Opción B: Manual
```bash
# Crear directorios necesarios
mkdir -p nginx/logs

# Construir e iniciar todos los servicios
docker-compose up --build -d

# Verificar estado de servicios
docker-compose ps
```

### 5. Verificar Despliegue

#### Health Checks
```bash
# Verificar servicios principales
curl -k https://localhost/health
curl http://localhost:8001/health
curl http://localhost:8080/health

# Verificar bases de datos
docker-compose exec postgres pg_isready
docker-compose exec mysql mysqladmin ping
docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"
```

#### Interfaz Web de RabbitMQ
- URL: http://localhost:15672
- Usuario: admin
- Contraseña: password

## Poblar con Datos de Prueba

### Ejecutar Seeder

```bash
# Instalar dependencias en el contenedor seeder
docker-compose run --rm seeder pip install requests

# Ejecutar seeder
docker-compose run --rm seeder python seeder.py
```

### Verificar Datos

```bash
# Login como administrador
curl -k -X POST https://localhost/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@streamflow.com","password":"admin123"}'

# Listar usuarios (con token del login anterior)
curl -k -X GET https://localhost/usuarios \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

## Credenciales por Defecto

### Usuario Administrador
- **Email**: admin@streamflow.com
- **Contraseña**: admin123

### Bases de Datos
- **PostgreSQL**: postgres/password
- **MySQL**: root/password
- **MariaDB**: root/password
- **MongoDB**: root/password
- **RabbitMQ**: admin/password

## Testing con Postman

### Importar Colecciones

1. Abrir Postman
2. Importar archivos desde `postman/`:
   - `environment.json` (Variables de entorno)
   - `complete_api.json` (API completa)
   - `flow1_cliente_basico.json` (Flujo 1)
   - `flow2_admin_facturas.json` (Flujo 2)

### Configurar Entorno

1. Seleccionar "StreamFlow Environment"
2. Verificar variables:
   - `base_url`: https://localhost
   - `admin_email`: admin@streamflow.com
   - `admin_password`: admin123

### Ejecutar Flujos de Prueba

#### Flujo 1: Cliente Básico
1. Obtener listado de videos
2. Registrar nuevo usuario cliente
3. Iniciar sesión con usuario creado
4. Obtener video por ID
5. Dar like al video

#### Flujo 2: Administrador
1. Login como administrador
2. Obtener todas las facturas
3. Marcar factura como pagada
4. Ver listado de acciones

## Monitoreo y Logs

### Ver Logs de Servicios

```bash
# Todos los servicios
docker-compose logs -f

# Servicio específico
docker-compose logs -f auth-service
docker-compose logs -f nginx
docker-compose logs -f api-gateway-1
```

### Logs de Nginx

```bash
# Ver logs de acceso con cuerpos de petición
docker-compose exec nginx tail -f /var/log/nginx/access.log

# Ver logs de error
docker-compose exec nginx tail -f /var/log/nginx/error.log
```

### Monitoreo de RabbitMQ

1. Acceder a http://localhost:15672
2. Ver colas: `user_creation_queue`, `invoice_update_queue`, `password_update_queue`
3. Monitorear mensajes y consumidores

## Endpoints Principales

### Acceso Público
- `GET /videos` - Listar videos
- `POST /usuarios` - Registrar usuario cliente
- `POST /auth/login` - Iniciar sesión
- `GET /comedia` - Endpoint cómico de Nginx

### Autenticados (Token JWT requerido)
- `GET /usuarios/{id}` - Obtener usuario
- `POST /videos` - Subir video (Admin)
- `POST /facturas` - Crear factura (Admin)
- `POST /interacciones/{id}/likes` - Dar like
- `GET /monitoreo/acciones` - Ver acciones (Admin)

## Solución de Problemas

### Problemas Comunes

#### 1. Servicios no inician
```bash
# Reiniciar sistema completo
docker-compose down
docker-compose up --build -d

# Verificar logs de error
docker-compose logs auth-service
```

#### 2. Error de conexión a base de datos
```bash
# Verificar estado de contenedores BD
docker-compose ps postgres mysql mariadb mongodb

# Reiniciar BD específica
docker-compose restart postgres
```

#### 3. Error de certificados SSL
```bash
# Regenerar certificados
docker-compose exec nginx openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/nginx.key \
  -out /etc/nginx/ssl/nginx.crt \
  -subj "/C=CL/ST=Antofagasta/L=Antofagasta/O=StreamFlow/OU=Development/CN=localhost"

docker-compose restart nginx
```

#### 4. RabbitMQ no conecta
```bash
# Reiniciar RabbitMQ
docker-compose restart rabbitmq

# Verificar configuración
docker-compose exec rabbitmq rabbitmq-diagnostics status
```

#### 5. API Gateway no responde
```bash
# Verificar las 3 instancias
docker-compose ps api-gateway-1 api-gateway-2 api-gateway-3

# Reiniciar instancias
docker-compose restart api-gateway-1 api-gateway-2 api-gateway-3
```

### Comandos de Depuración

```bash
# Estado completo del sistema
./logs.sh

# Reinicio completo
./stop.sh && ./start.sh

# Limpieza completa (CUIDADO: elimina datos)
./clean.sh
```

### Verificación de Red

```bash
# Verificar conectividad entre servicios
docker-compose exec api-gateway-1 ping auth-service
docker-compose exec api-gateway-1 ping users-service

# Verificar puertos internos
docker-compose exec api-gateway-1 nc -zv auth-service 8001
docker-compose exec api-gateway-1 nc -zv users-service 50051
```

## Escalabilidad y Producción

### Consideraciones para Producción

1. **Secretos**: Usar gestores de secretos en lugar de variables en texto plano
2. **Certificados**: Usar certificados válidos de CA en lugar de autofirmados
3. **Bases de Datos**: Configurar replicación y backup automático
4. **Monitoreo**: Implementar Prometheus/Grafana para métricas avanzadas
5. **Logs**: Centralizar logs con ELK Stack o similar

### Escalado Horizontal

```bash
# Escalar API Gateway a 5 instancias
docker-compose up --scale api-gateway-1=5 -d

# Escalar microservicios específicos
docker-compose up --scale videos-service=3 -d
```

## Contacto y Soporte

Para problemas técnicos o dudas sobre la implementación:
- Revisar logs detallados: `./logs.sh [service-name]`
- Verificar documentación del código en cada servicio
- Consultar colecciones Postman para ejemplos de uso

---

**Nota**: Esta guía cubre el despliegue completo del sistema StreamFlow con arquitectura de microservicios. Asegúrese de tener suficientes recursos de sistema antes de proceder con el despliegue.
