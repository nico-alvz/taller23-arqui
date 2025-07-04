# StreamFlow - Plataforma de Streaming con Microservicios

Este proyecto implementa una arquitectura de microservicios para la plataforma de streaming StreamFlow.

## Arquitectura

### Microservicios

1. **Autenticación** (Puerto 8001, HTTP)
   - Base de datos: PostgreSQL
   - Responsabilidades: JWT, blacklist, login/logout

2. **Usuarios** (Puerto 50051, gRPC)
   - Base de datos: MySQL
   - Responsabilidades: CRUD usuarios, roles

3. **Facturación** (Puerto 50052, gRPC)
   - Base de datos: MariaDB
   - Responsabilidades: Gestión facturas, pagos

4. **Videos** (Puerto 50053, gRPC)
   - Base de datos: MongoDB
   - Responsabilidades: Gestión contenido audiovisual

5. **Monitoreo** (Puerto 50054, gRPC)
   - Base de datos: MongoDB
   - Responsabilidades: Logs de acciones y errores

6. **Listas de Reproducción** (Puerto 50055, gRPC)
   - Base de datos: PostgreSQL
   - Responsabilidades: Playlists de usuarios

7. **Interacciones Sociales** (Puerto 50056, gRPC)
   - Base de datos: MongoDB
   - Responsabilidades: Likes y comentarios

8. **Envío de Correos** (Puerto 50057, gRPC)
   - Sin base de datos
   - Responsabilidades: Notificaciones por email

9. **API Gateway** (Puertos 8080-8082, HTTP)
   - Sin base de datos
   - Responsabilidades: Punto de entrada único

### Comunicación

- **Externa → API Gateway**: HTTP/HTTPS
- **API Gateway → Autenticación**: HTTP
- **API Gateway → Otros servicios**: gRPC
- **Entre microservicios**: RabbitMQ

### Balanceador de Carga

- **Nginx**: Puertos 80 (HTTP) y 443 (HTTPS)
- Balancea entre 3 instancias del API Gateway
- Configurado con SSL/TLS

## Despliegue

### Prerequisitos

- Docker y Docker Compose
- Al menos 8GB de RAM disponible

### Iniciar el Sistema

```bash
# Clonar el repositorio
git clone <repository-url>
cd streamflow

# Iniciar todos los servicios
docker-compose up -d

# Ver logs
docker-compose logs -f

# Verificar estado de servicios
docker-compose ps
```

### Configuración de Base de Datos

Las bases de datos se inicializan automáticamente al iniciar los contenedores.

### Seeder

Para poblar las bases de datos con datos de prueba:

```bash
# Ejecutar seeder
docker-compose exec api-gateway-1 /app/scripts/seed.sh
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

## Endpoints API

### Autenticación
- `POST /auth/login` - Iniciar sesión
- `PATCH /auth/usuarios/{id}` - Cambiar contraseña
- `POST /auth/logout` - Cerrar sesión

### Usuarios
- `POST /usuarios` - Crear usuario
- `GET /usuarios/{id}` - Obtener usuario
- `PATCH /usuarios/{id}` - Actualizar usuario
- `DELETE /usuarios/{id}` - Eliminar usuario
- `GET /usuarios` - Listar usuarios

### Facturación
- `POST /facturas` - Crear factura
- `GET /facturas/{id}` - Obtener factura
- `PATCH /facturas/{id}` - Actualizar factura
- `DELETE /facturas/{id}` - Eliminar factura
- `GET /facturas` - Listar facturas

### Videos
- `POST /videos` - Subir video
- `GET /videos/{id}` - Obtener video
- `PATCH /videos/{id}` - Actualizar video
- `DELETE /videos/{id}` - Eliminar video
- `GET /videos` - Listar videos

### Monitoreo
- `GET /monitoreo/acciones` - Listar acciones
- `GET /monitoreo/errores` - Listar errores

### Listas de Reproducción
- `POST /listas-reproduccion` - Crear lista
- `POST /listas-reproduccion/{id}/videos` - Añadir video
- `GET /listas-reproduccion` - Ver listas
- `GET /listas-reproduccion/{id}/videos` - Ver videos de lista
- `DELETE /listas-reproduccion/{id}/videos` - Eliminar video de lista
- `DELETE /listas-reproduccion/{id}` - Eliminar lista

### Interacciones Sociales
- `POST /interacciones/{id}/likes` - Dar like
- `POST /interacciones/{id}/comentarios` - Comentar
- `GET /interacciones/{id}` - Ver interacciones

## Monitoreo

### RabbitMQ Management
- **URL**: http://localhost:15672
- **Usuario**: admin
- **Contraseña**: password

### Logs
```bash
# Ver logs de un servicio específico
docker-compose logs -f [service-name]

# Ver logs de Nginx
docker-compose logs -f nginx
```

## Desarrollo

### Estructura del Código

```
streamflow/
├── services/           # Microservicios
│   ├── auth/          # Servicio de autenticación
│   ├── users/         # Servicio de usuarios  
│   ├── billing/       # Servicio de facturación
│   ├── videos/        # Servicio de videos
│   ├── monitoring/    # Servicio de monitoreo
│   ├── playlists/     # Servicio de listas
│   ├── social/        # Servicio social
│   └── email/         # Servicio de email
├── api-gateway/       # API Gateway
├── nginx/             # Configuración Nginx
├── protos/            # Archivos Protocol Buffers
├── scripts/           # Scripts de utilidad
├── postman/           # Colecciones Postman
└── docs/              # Documentación
```

### Testing

#### Colecciones Postman

Se incluyen colecciones Postman para probar los flujos principales:

1. **Flujo Cliente**: Registro, login, ver videos, dar likes
2. **Flujo Administrador**: Gestión facturas, usuarios, contenido
3. **Flujo Listas**: Crear playlists, gestionar videos
4. **Flujo Completo**: Casos de uso end-to-end

## Seguridad

### Autenticación JWT
- Tokens con expiración de 24 horas
- Blacklist para logout seguro
- Validación en API Gateway

### HTTPS/SSL
- Certificados autofirmados incluidos
- Redirección automática HTTP → HTTPS
- Headers de seguridad configurados

### Validaciones
- Autorización basada en roles
- Validación de entrada en todos los endpoints
- Soft delete para datos sensibles

## Troubleshooting

### Problemas Comunes

1. **Servicios no inician**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

2. **Error de conexión de base de datos**
   ```bash
   # Verificar estado de contenedores
   docker-compose ps
   
   # Reiniciar base de datos específica
   docker-compose restart [postgres|mysql|mariadb|mongodb]
   ```

3. **RabbitMQ no conecta**
   ```bash
   docker-compose restart rabbitmq
   ```

4. **Logs de depuración**
   ```bash
   # Ver todos los logs
   docker-compose logs

   # Logs de un servicio específico
   docker-compose logs [service-name]
   ```

## Contribución

1. Fork el repositorio
2. Crear branch para feature (`git checkout -b feature/nueva-funcionalidad`)
3. Commit cambios (`git commit -am 'Agregar nueva funcionalidad'`)
4. Push al branch (`git push origin feature/nueva-funcionalidad`)
5. Crear Pull Request

## Licencia

Este proyecto es para fines educativos del curso de Arquitectura de Sistemas.
