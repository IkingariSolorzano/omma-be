# Omma Backend - Sistema de Renta de Consultorios

Backend API para la aplicación Omma, un sistema de renta de consultorios por horas con manejo de créditos y reservaciones.

## Características

- **Gestión de Usuarios**: Administradores y profesionales con autenticación JWT
- **Sistema de Créditos**: Compra y manejo de créditos con expiración de 30 días
- **Reservaciones**: Sistema de reservas con validación de horarios y aprobaciones
- **Penalizaciones**: Sistema de penalizaciones por cancelaciones tardías (< 24 horas)
- **Directorio de Profesionales**: Listado público de profesionales con créditos activos

## Tecnologías

- **Go 1.23.2**
- **Gin Framework** - API REST
- **GORM** - ORM para PostgreSQL
- **Goose** - Migraciones de base de datos
- **JWT** - Autenticación
- **PostgreSQL** - Base de datos
- **Docker** - Containerización

## Instalación y Configuración

### Prerrequisitos

- Go 1.23.2 o superior
- PostgreSQL 15
- Docker (opcional)

### Configuración Local

1. Clona el repositorio:
```bash
git clone <repository-url>
cd omma-be
```

2. Copia el archivo de configuración:
```bash
cp .env.example .env
```

3. Configura las variables de entorno en `.env`:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=omma_user
DB_PASSWORD=your_password
DB_NAME=omma_db
DB_SSLMODE=disable
JWT_SECRET=your_jwt_secret_key_here
PORT=8080
ADMIN_EMAIL=admin@omma.com
ADMIN_PASSWORD=admin123
```

4. Instala las dependencias y configura las migraciones:

**En Windows (PowerShell):**
```bash
.\install_migrations.ps1
```

**En Linux/Mac:**
```bash
chmod +x install_migrations.sh
./install_migrations.sh
```

**O manualmente:**
```bash
go get github.com/pressly/goose/v3
go mod tidy
```

5. Ejecuta la aplicación:
```bash
go run main.go
```

Las migraciones se ejecutarán automáticamente al iniciar la aplicación, creando:
- Usuario administrador por defecto
- Horarios de negocio (Lun-Vie: 10:00-20:00, Sáb: 09:00-18:00)

### Usando Docker

1. Ejecuta con Docker Compose:
```bash
docker-compose up -d
```

Esto iniciará PostgreSQL y la aplicación automáticamente.

## API Endpoints

### Autenticación
- `POST /api/v1/auth/login` - Iniciar sesión
- `POST /api/v1/auth/register` - Registrar nuevo profesional

### Usuario (Requiere autenticación)
- `GET /api/v1/profile` - Obtener perfil del usuario
- `GET /api/v1/credits` - Obtener créditos del usuario
- `GET /api/v1/spaces` - Listar espacios disponibles
- `GET /api/v1/reservations` - Obtener reservaciones del usuario
- `POST /api/v1/reservations` - Crear nueva reservación
- `DELETE /api/v1/reservations/:id` - Cancelar reservación

### Administración (Solo administradores)
- `POST /api/v1/admin/users` - Crear usuario
- `GET /api/v1/admin/users` - Listar usuarios
- `POST /api/v1/admin/credits` - Asignar créditos
- `POST /api/v1/admin/spaces` - Crear espacio
- `GET /api/v1/admin/spaces` - Listar espacios
- `POST /api/v1/admin/schedules` - Crear horario
- `GET /api/v1/admin/reservations/pending` - Ver reservaciones pendientes
- `PUT /api/v1/admin/reservations/:id/approve` - Aprobar reservación

### Público
- `GET /api/v1/professionals` - Directorio de profesionales

## Modelo de Datos

### Usuarios
- Roles: `admin`, `professional`
- Campos: email, nombre, teléfono, especialidad, descripción

### Créditos
- Sistema de múltiplos de 6 créditos
- Expiración: 30 días desde la compra
- Deducción FIFO (primero en expirar, primero en usar)

### Espacios
- Costo estándar: 6 créditos (60-100 pesos)
- Horarios configurables por día de la semana

### Reservaciones
- Estados: `pending`, `confirmed`, `cancelled`, `completed`
- Validación de conflictos de horario
- Aprobación requerida para horarios fuera de lo establecido

### Penalizaciones
- Cancelación < 24 horas: 2 créditos de penalización
- Cancelación > 24 horas: sin penalización, reembolso completo

## Despliegue en VPS Ubuntu

1. Instala Docker y Docker Compose en tu VPS
2. Clona el repositorio
3. Configura las variables de entorno de producción
4. Ejecuta: `docker-compose up -d`
5. Configura un proxy reverso (Nginx) si es necesario

## Migraciones de Base de Datos

El proyecto utiliza **Goose** para gestionar las migraciones de base de datos. Las migraciones se ejecutan automáticamente al iniciar la aplicación.

### Migraciones Incluidas

1. **Creación de usuario administrador**: Se crea automáticamente usando las credenciales de las variables de entorno `ADMIN_EMAIL` y `ADMIN_PASSWORD`
2. **Horarios de negocio por defecto**:
   - Lunes a Viernes: 10:00 - 20:00
   - Sábado: 09:00 - 18:00
   - Domingo: Cerrado

Para más información sobre las migraciones, consulta [MIGRATION_SETUP.md](./MIGRATION_SETUP.md) y [migrations/README.md](./migrations/README.md).

## Usuario Administrador por Defecto

Al iniciar la aplicación por primera vez, la migración crea automáticamente un usuario administrador:
- **Email**: admin@omma.com (configurable con `ADMIN_EMAIL`)
- **Password**: admin123 (configurable con `ADMIN_PASSWORD`)

**¡Importante!** Cambia estas credenciales en producción configurando las variables de entorno antes del primer inicio.

## Estructura del Proyecto

```
omma-be/
├── config/          # Configuración de base de datos y migraciones
├── controllers/     # Controladores HTTP
├── middleware/      # Middleware de autenticación
├── migrations/      # Migraciones de base de datos (Goose)
├── models/          # Modelos de datos
├── routes/          # Definición de rutas
├── services/        # Lógica de negocio
├── main.go          # Punto de entrada
├── Dockerfile       # Configuración Docker
├── docker-compose.yml
├── MIGRATION_SETUP.md    # Guía de configuración de migraciones
└── install_migrations.*  # Scripts de instalación
```
