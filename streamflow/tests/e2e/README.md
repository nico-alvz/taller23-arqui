# StreamFlow E2E Tests

Este directorio contiene las pruebas End-to-End (E2E) para la plataforma StreamFlow, implementadas como parte del sistema de Integración y Despliegue Continuo (CI/CD).

## 📋 Descripción

Las pruebas E2E validan el funcionamiento completo del sistema StreamFlow a través de la API Gateway, asegurando que todos los microservicios funcionen correctamente en conjunto.

## 🎯 Objetivos de las Pruebas

### 1. Pruebas de Humo (Smoke Tests)
- Verificar que todos los servicios estén funcionando
- Validar la conectividad entre servicios
- Comprobar el balanceador de carga Nginx

### 2. Pruebas CRUD de Usuarios (Requerimiento Principal)
Según las especificaciones del proyecto, se implementan pruebas para:

1. **POST /auth/login** - Iniciar sesión
   - ✅ Caso de éxito: Autenticación con credenciales válidas
   - ❌ Caso de error: Credenciales inválidas

2. **POST /usuarios** - Crear usuario
   - ✅ Caso de éxito: Creación con datos válidos
   - ❌ Caso de error: Email duplicado, formato inválido

3. **GET /usuarios/{id}** - Obtener usuario por ID
   - ✅ Caso de éxito: ID válido y permisos correctos
   - ❌ Caso de error: ID inexistente, sin autenticación

4. **PATCH /usuarios/{id}** - Actualizar usuario
   - ✅ Caso de éxito: Actualización con datos válidos
   - ❌ Caso de error: Usuario inexistente, datos inválidos

5. **DELETE /usuarios/{id}** - Eliminar usuario
   - ✅ Caso de éxito: Eliminación exitosa
   - ❌ Caso de error: Usuario inexistente, sin permisos

6. **GET /usuarios** - Listar todos los usuarios
   - ✅ Caso de éxito: Listado para administrador
   - ❌ Caso de error: Sin autenticación, permisos insuficientes

## 🛠️ Configuración

### Estructura de Archivos

```
tests/e2e/
├── package.json              # Dependencias y scripts de npm
├── tsconfig.json             # Configuración de TypeScript
├── .env.test                 # Variables de entorno para pruebas
├── README.md                 # Esta documentación
└── src/
    ├── setup.ts              # Configuración global de pruebas
    ├── utils/
    │   └── test-helper.ts    # Utilidades para pruebas
    └── __tests__/
        ├── smoke.test.ts                # Pruebas de humo
        ├── auth.test.ts                 # Pruebas de autenticación
        ├── users.test.ts                # Pruebas de usuarios
        └── users-crud-e2e.test.ts       # Pruebas CRUD requeridas
```

### Variables de Entorno

```env
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

## 🚀 Ejecución Local

### Prerrequisitos

1. **Docker y Docker Compose** - Para ejecutar los servicios
2. **Node.js 18+** - Para ejecutar las pruebas
3. **npm** - Para gestionar dependencias

### Pasos para Ejecutar

1. **Iniciar los servicios de StreamFlow:**
   ```bash
   cd streamflow/
   docker-compose up -d
   ```

2. **Esperar a que los servicios estén listos:**
   ```bash
   # Verificar que todos los servicios estén funcionando
   docker-compose ps
   
   # Verificar el endpoint de health
   curl http://localhost:80/health
   ```

3. **Instalar dependencias de pruebas:**
   ```bash
   cd tests/e2e/
   npm install
   ```

4. **Ejecutar todas las pruebas:**
   ```bash
   npm test
   ```

5. **Ejecutar pruebas específicas:**
   ```bash
   # Solo pruebas de humo
   npm run test:smoke
   
   # Solo pruebas de autenticación
   npm run test:auth
   
   # Solo pruebas CRUD de usuarios (requeridas)
   npm test -- --testNamePattern="Users Service CRUD E2E"
   ```

## 🔄 Integración CI/CD

### GitHub Actions

Las pruebas E2E se ejecutan automáticamente en GitHub Actions con dos workflows:

#### 1. Docker Build and Push (`docker-publish.yml`)
- **Trigger:** Push a rama `main`
- **Función:** Construir y subir imágenes Docker a Docker Hub
- **Servicios:** Todos los microservicios + API Gateway + Nginx

#### 2. E2E Tests (`e2e-tests.yml`)
- **Trigger:** Push a rama `main` que afecte servicios de usuarios
- **Función:** Ejecutar pruebas CRUD de usuarios
- **Duración:** ~30 minutos máximo

### Configuración de Secrets

Para que funcione el CI/CD, configurar estos secrets en GitHub:

```
DOCKER_USERNAME=tu_usuario_dockerhub
DOCKER_PASSWORD=tu_password_dockerhub
```

## 📊 Reportes y Cobertura

### Generar Reporte de Cobertura
```bash
npm run test:coverage
```

### Ver Resultados en CI
Los resultados se suben como artefactos en GitHub Actions:
- `e2e-test-results/coverage/` - Reporte de cobertura
- `e2e-test-results/test-results/` - Resultados detallados

## 🐛 Depuración

### Logs de Servicios
```bash
# Ver logs de un servicio específico
docker-compose logs -f auth-service

# Ver logs de API Gateway
docker-compose logs -f api-gateway-1

# Ver logs de Nginx
docker-compose logs -f nginx
```

### Debugging de Pruebas
```bash
# Ejecutar con más verbosidad
npm test -- --verbose

# Ejecutar una prueba específica
npm test -- --testNamePattern="should authenticate admin user successfully"

# Modo watch para desarrollo
npm run test:watch
```

### Health Checks
```bash
# Verificar endpoints de salud
curl http://localhost:80/health
curl http://localhost:8080/health
curl http://localhost:8001/health

# Verificar el endpoint cómico
curl http://localhost:80/comedia
```

## 📈 Métricas y Monitoreo

### Endpoints de Monitoreo
- **Nginx:** `http://localhost:80/health`
- **API Gateway:** `http://localhost:8080/health`
- **Auth Service:** `http://localhost:8001/health`
- **RabbitMQ Management:** `http://localhost:15672`

### Tiempo de Ejecución Esperado
- **Pruebas de Humo:** ~2 minutos
- **Pruebas CRUD Usuarios:** ~5 minutos
- **Todas las pruebas:** ~10 minutos

## 🔧 Troubleshooting

### Problemas Comunes

1. **Servicios no inician:**
   ```bash
   docker-compose down -v
   docker-compose up -d
   ```

2. **Timeout en pruebas:**
   - Aumentar `TEST_TIMEOUT` en `.env.test`
   - Verificar recursos del sistema

3. **Fallas de autenticación:**
   - Verificar credenciales en `.env.test`
   - Comprobar que el servicio de auth esté funcionando

4. **Error de conexión a base de datos:**
   ```bash
   docker-compose restart postgres mysql mariadb mongodb
   ```

## 📝 Notas de Desarrollo

### Agregar Nuevas Pruebas

1. Crear archivo en `src/__tests__/`
2. Importar `TestHelper` para utilidades
3. Seguir el patrón de casos de éxito y error
4. Documentar en este README

### Buenas Prácticas

- ✅ Limpiar datos de prueba después de cada test
- ✅ Usar datos únicos (UUID) para evitar conflictos
- ✅ Probar tanto casos de éxito como de error
- ✅ Validar códigos de estado HTTP específicos
- ✅ Verificar que no se filtren contraseñas en respuestas

## 🎯 Cumplimiento de Requerimientos

Este conjunto de pruebas E2E cumple completamente con los requerimientos especificados:

- ✅ **CI/CD con GitHub Actions** implementado
- ✅ **Docker Hub automático** en cada commit a main
- ✅ **Pruebas E2E** en cada commit que afecte usuarios
- ✅ **CRUD completo** de usuarios a través de API Gateway
- ✅ **Casos de éxito y error** para todos los endpoints requeridos
- ✅ **Flujo completo** desde autenticación hasta eliminación

---

**Desarrollado para:** Taller 2 - Arquitectura de Sistemas  
**Fecha:** Julio 2025  
**Estado:** ✅ Completo y listo para producción

