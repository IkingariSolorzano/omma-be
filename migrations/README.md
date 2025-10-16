# Database Migrations

Este directorio contiene las migraciones de base de datos usando [Goose](https://github.com/pressly/goose).

## Estructura

Las migraciones están escritas en Go para aprovechar el sistema de tipos y las bibliotecas existentes (como bcrypt para hashear contraseñas).

## Migraciones Actuales

### 00001_create_default_admin.go
Crea el usuario administrador por defecto. Las credenciales se toman de las variables de entorno:
- `ADMIN_EMAIL` (default: admin@omma.com)
- `ADMIN_PASSWORD` (default: admin123)

La contraseña se hashea con bcrypt usando cost 14, igual que en el resto de la aplicación.

**Nota**: Esta migración NO inserta créditos directamente en la tabla `users` porque los créditos se manejan en una tabla separada (`credits`) con una relación uno-a-muchos.

### 00002_create_default_business_hours.go
Configura los horarios de negocio por defecto:
- **Lunes a Viernes**: 10:00 - 20:00
- **Sábado**: 09:00 - 18:00
- **Domingo**: Cerrado

## Instalación de Goose

Para instalar Goose como herramienta CLI (opcional):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

## Uso

### Ejecutar migraciones automáticamente
Las migraciones se ejecutan automáticamente al iniciar la aplicación en `main.go`.

### Ejecutar migraciones manualmente con Goose CLI

```bash
# Aplicar todas las migraciones pendientes
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=omma sslmode=disable" up

# Revertir la última migración
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=omma sslmode=disable" down

# Ver el estado de las migraciones
goose -dir migrations postgres "host=localhost port=5432 user=postgres password=postgres dbname=omma sslmode=disable" status
```

## Crear una nueva migración

Para crear una nueva migración en Go:

```bash
goose -dir migrations create nombre_de_migracion go
```

Luego edita el archivo generado siguiendo el patrón de las migraciones existentes.

## Notas Importantes

- **NO** elimines el archivo `generate_hash.go` si existe, es un helper para generar hashes de contraseñas.
- Las migraciones se ejecutan en orden numérico.
- Cada migración debe tener funciones `Up` y `Down` para poder revertirlas.
- Las migraciones se ejecutan dentro de transacciones, por lo que si una falla, se revierte automáticamente.
