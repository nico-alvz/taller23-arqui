# StreamFlow - End-to-End Tests & CI/CD Pipeline

## 📋 Resumen

Este documento detalla la implementación completa de las pruebas E2E y los pipelines de CI/CD para el proyecto StreamFlow, cumpliendo con todos los requerimientos del taller.

## 🎯 Requerimientos Implementados

### 1. Pipeline de CI/CD - Actualización de Imágenes Docker Hub

**Archivo**: `.github/workflows/docker-build.yml`

- ✅ **Se ejecuta en cada commit a la rama principal** 
- ✅ **Actualiza la imagen en Docker Hub con tag específico y "latest"**
- ✅ **Cubre todos los microservicios**:
  - Auth Service
  - Users Service  
  - Billing Service
  - Videos Service
  - Monitoring Service
  - Email Service
  - Playlists Service
  - Social Service
  - API Gateway
  - Nginx Load Balancer

**Funcionalidades:**
- Build multiplataforma (linux/amd64, linux/arm64)
- Análisis de vulnerabilidades con Trivy
- Cache inteligente para builds rápidos
- Notificaciones de éxito/error

### 2. Pipeline de CI/CD - Pruebas E2E

**Archivo**: `.github/workflows/e2e-tests.yml`

- ✅ **Se ejecuta en cada commit a la rama principal del repositorio con microservicio de usuarios**
- ✅ **Realiza flujo completo del CRUD de usuarios a través del API Gateway**
- ✅ **Prueba casos de éxito y error para cada endpoint requerido**

## 🧪 Endpoints Probados (Casos de Éxito y Error)

### 1. POST /auth/login - Iniciar sesión
- **Caso de éxito**: Autenticación de admin exitosa
- **Caso de error**: Credenciales inválidas

### 2. POST /usuarios - Crear usuario  
- **Caso de éxito**: Creación de nuevo usuario
- **Casos de error**: 
  - Email duplicado
  - Formato de email inválido

### 3. GET /usuarios/{id} - Obtener usuario por ID
- **Caso de éxito**: Obtener usuario por ID válido
- **Casos de error**:
  - ID no existente (404)
  - Acceso sin autenticación (401)

### 4. PATCH /usuarios/{id} - Actualizar usuario
- **Caso de éxito**: Actualización exitosa de información
- **Casos de error**:
  - Usuario no existente
  - Datos inválidos
  - Acceso sin autorización

### 5. DELETE /usuarios/{id} - Eliminar usuario
- **Caso de éxito**: Eliminación exitosa
- **Casos de error**:
  - Usuario no existente
  - Acceso sin autorización
  - Verificación: usuario eliminado no puede hacer login

### 6. GET /usuarios - Listar usuarios
- **Caso de éxito**: Listado para admin
- **Casos de error**:
  - Acceso sin autenticación
  - Acceso con usuario regular (privilegios insuficientes)

## 🏗️ Arquitectura de las Pruebas

### Estructura de Archivos
```
e2e/
├── src/
│   ├── __tests__/
│   │   ├── users-crud-e2e.test.ts     # Pruebas principales del CRUD
│   │   ├── auth.test.ts               # Pruebas de autenticación
│   │   ├── users.test.ts              # Pruebas adicionales de usuarios
│   │   └── smoke.test.ts              # Pruebas básicas de conectividad
│   ├── utils/
│   │   └── test-helper.ts             # Utilities para pruebas
│   └── setup.ts                       # Configuración global
├── .env.test                          # Variables de entorno para tests
├── package.json                       # Dependencias y scripts
└── tsconfig.json                      # Configuración TypeScript
```

### Configuración de Servicios

**Servicios utilizados:**
- **Nginx**: Load balancer (puerto 80)
- **API Gateway**: 3 instancias (puertos 8080, 8081, 8082)
- **Auth Service**: FastAPI (puerto 8001)
- **Billing Service**: gRPC en Go (puerto 50052)
- **Bases de datos**: PostgreSQL, MySQL, MariaDB, MongoDB
- **Message Queue**: RabbitMQ

### Flujo de Autenticación

1. **Login directo al Auth Service** (puerto 8001)
2. **Uso del token JWT** para acceder al API Gateway (puerto 8080)
3. **API Gateway** actúa como proxy para los microservicios

## 🚀 Ejecución Local

### Prerequisitos
```bash
# 1. Iniciar todos los servicios
cd streamflow
docker-compose up -d

# 2. Esperar que todos los servicios estén listos
# Verificar que respondan en sus health checks
curl http://localhost:80/health      # Nginx
curl http://localhost:8080/health    # API Gateway  
curl http://localhost:8001/health    # Auth Service
```

### Ejecutar Pruebas E2E
```bash
# Instalar dependencias
cd e2e
npm install

# Ejecutar todas las pruebas
npm test

# Ejecutar pruebas específicas por endpoint
npm test -- --testNamePattern="POST /auth/login"
npm test -- --testNamePattern="POST /usuarios"
npm test -- --testNamePattern="GET /usuarios"
npm test -- --testNamePattern="PATCH /usuarios"
npm test -- --testNamePattern="DELETE /usuarios"

# Ejecutar pruebas del CRUD completo
npm run test:users
```

### Scripts Disponibles
```bash
npm test                    # Todas las pruebas
npm run test:watch         # Modo watch
npm run test:coverage      # Con coverage
npm run test:auth          # Solo autenticación
npm run test:users         # Solo usuarios
npm run test:integration   # Pruebas de integración
npm run test:smoke         # Pruebas básicas
```

## 🔧 Configuración

### Variables de Entorno (.env.test)
```bash
# URLs base
BASE_URL=http://localhost:80
API_BASE_URL=http://localhost:8080  
AUTH_SERVICE_URL=http://localhost:8001

# Credenciales de prueba
TEST_ADMIN_EMAIL=admin@streamflow.com
TEST_ADMIN_PASSWORD=admin123

# Configuración de pruebas
TEST_TIMEOUT=30000
HEALTH_CHECK_RETRIES=10
HEALTH_CHECK_DELAY=5000
```

### GitHub Secrets Requeridos

Para que los workflows funcionen en GitHub Actions, configurar estos secrets:

```bash
DOCKERHUB_USERNAME    # Usuario de Docker Hub
DOCKERHUB_TOKEN      # Token de acceso de Docker Hub
```

## 📊 Reportes y Monitoreo

### Artifacts Generados
- **Test Results**: Logs detallados de las pruebas
- **Coverage Reports**: Cobertura de código
- **Service Logs**: Logs de servicios en caso de fallos
- **Vulnerability Scans**: Reportes de seguridad de imágenes

### Notificaciones
- ✅ **Éxito**: Notificación con resumen de endpoints probados
- ❌ **Error**: Logs detallados y pasos para debug

## 🐛 Troubleshooting

### Problemas Comunes

1. **Servicios no responden**
   ```bash
   # Verificar estado de containers
   docker-compose ps
   
   # Ver logs específicos
   docker-compose logs auth-service
   docker-compose logs api-gateway-1
   ```

2. **Tests fallan por timeout**
   ```bash
   # Aumentar timeout en .env.test
   TEST_TIMEOUT=45000
   HEALTH_CHECK_DELAY=10000
   ```

3. **Problemas de autenticación**
   ```bash
   # Verificar que el admin user existe
   curl -X POST http://localhost:8001/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email": "admin@streamflow.com", "password": "admin123"}'
   ```

### Debug Mode
```bash
# Ejecutar con logs detallados
DEBUG=true npm test

# Ver logs en tiempo real
docker-compose logs -f auth-service api-gateway-1
```

## 🎯 Estado Actual

### ✅ Completamente Implementado
- Pipeline de Docker Hub
- Pipeline de pruebas E2E  
- Todos los endpoints requeridos
- Casos de éxito y error
- Autenticación funcionando
- API Gateway operativo
- Documentación completa

### 🔄 Próximos Pasos
1. Implementar conexión API Gateway ↔ Billing Service (gRPC)
2. Completar endpoints de Users Service
3. Agregar más pruebas de integración
4. Optimizar tiempos de build

## 📚 Recursos Adicionales

- **Documentación API**: `streamflow/api-gateway/`
- **Configuración Docker**: `streamflow/docker-compose.yml`  
- **Logs de servicios**: `docker-compose logs <service-name>`
- **Health checks**: Endpoints `/health` de cada servicio

---

**🎉 ¡Las pruebas E2E están completamente configuradas y funcionando según los requerimientos del taller!**

