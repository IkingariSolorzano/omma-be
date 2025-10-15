# Corrección de Fugas de Conexiones a Base de Datos

## Problema Detectado en Producción

Se detectaron **35+ conexiones PostgreSQL en estado "idle in transaction"**, lo que indica que las transacciones se iniciaban pero nunca se completaban (ni commit ni rollback).

```bash
postgres 2032337  0.0  1.6 226224 16464 ?  Ss   Oct13   0:00 postgres: 16/main: omma_admin omma_db 127.0.0.1(50258) idle in transaction
postgres 2033008  0.0  1.6 226172 16408 ?  Ss   Oct13   0:00 postgres: 16/main: omma_admin omma_db 127.0.0.1(33932) idle in transaction
# ... 33 más conexiones en el mismo estado
```

## Problemas Encontrados y Corregidos

### 1. ❌ Transacción sin verificar Commit en `AdminCancelReservation`
**Archivo:** `services/reservation.go`

**Antes:**
```go
tx := config.DB.Begin()
// ... operaciones ...
tx.Commit()  // ❌ NO verifica el error
return nil
```

**Después:**
```go
tx := config.DB.Begin()
// ... operaciones ...
if err := tx.Commit().Error; err != nil {
    return err
}
return nil
```

### 2. ❌ Transacción sin verificar Commit en `RegisterPayment`
**Archivo:** `services/payment.go`

**Antes:**
```go
tx := config.DB.Begin()
// ... operaciones ...
tx.Commit()  // ❌ NO verifica el error
return &payment, nil
```

**Después:**
```go
tx := config.DB.Begin()
// ... operaciones ...
if err := tx.Commit().Error; err != nil {
    return nil, err
}
return &payment, nil
```

### 3. ❌ Falta de Configuración del Pool de Conexiones
**Archivo:** `config/database.go`

**Agregado:**
```go
// Configurar el pool de conexiones
sqlDB, err := database.DB()
if err != nil {
    log.Fatal("Error al obtener la instancia de SQL DB:", err)
}

// Configuración del pool de conexiones
sqlDB.SetMaxOpenConns(25)                  // Máximo de conexiones abiertas
sqlDB.SetMaxIdleConns(5)                   // Máximo de conexiones inactivas
sqlDB.SetConnMaxLifetime(5 * time.Minute)  // Tiempo de vida máximo de una conexión
sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Tiempo máximo de inactividad
```

## Impacto de las Correcciones

### Antes:
- ❌ Conexiones ilimitadas sin control
- ❌ Transacciones abiertas indefinidamente si Commit() fallaba
- ❌ Conexiones nunca se cerraban automáticamente
- ❌ Riesgo de agotar el pool de PostgreSQL (default: 100 conexiones)

### Después:
- ✅ Máximo 25 conexiones abiertas simultáneas
- ✅ Conexiones inactivas se cierran después de 10 minutos
- ✅ Conexiones se reciclan cada 5 minutos
- ✅ Errores de Commit se manejan correctamente
- ✅ Transacciones se completan o se revierten apropiadamente

## Pasos para Desplegar en Producción

### 1. Verificar Conexiones Actuales
```bash
# Ver conexiones actuales
ps aux | grep postgres | grep omma_db

# Contar conexiones "idle in transaction"
ps aux | grep "idle in transaction" | wc -l
```

### 2. Detener el Servicio
```bash
sudo systemctl stop omma-be
# o si es un proceso manual:
sudo pkill omma-be
```

### 3. Limpiar Conexiones Huérfanas (Opcional)
```sql
-- Conectarse a PostgreSQL
sudo -u postgres psql omma_db

-- Ver conexiones activas
SELECT pid, usename, application_name, client_addr, state, state_change 
FROM pg_stat_activity 
WHERE datname = 'omma_db';

-- Terminar conexiones "idle in transaction" (CUIDADO: solo si el backend está detenido)
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE datname = 'omma_db' 
  AND state = 'idle in transaction'
  AND state_change < NOW() - INTERVAL '5 minutes';
```

### 4. Compilar y Desplegar Nueva Versión
```bash
# En tu máquina local o servidor de build
cd omma-be
go build -o omma-be

# Subir al servidor
scp omma-be root@servidor:/var/www/html/omma-be/

# En el servidor
sudo systemctl start omma-be
# o si es manual:
cd /var/www/html/omma-be
./omma-be &
```

### 5. Monitorear Conexiones
```bash
# Monitorear conexiones cada 5 segundos
watch -n 5 "ps aux | grep postgres | grep omma_db | wc -l"

# Ver estado de las conexiones
watch -n 5 "ps aux | grep postgres | grep omma_db | grep -c 'idle in transaction'"
```

## Monitoreo Continuo

### Query para PostgreSQL
```sql
-- Conexiones por estado
SELECT state, count(*) 
FROM pg_stat_activity 
WHERE datname = 'omma_db' 
GROUP BY state;

-- Conexiones "idle in transaction" por más de 1 minuto
SELECT pid, usename, state, state_change, 
       NOW() - state_change as duration
FROM pg_stat_activity 
WHERE datname = 'omma_db' 
  AND state = 'idle in transaction'
  AND state_change < NOW() - INTERVAL '1 minute';
```

### Script de Monitoreo (Bash)
```bash
#!/bin/bash
# monitor_connections.sh

while true; do
    echo "=== $(date) ==="
    echo "Total conexiones omma_db:"
    ps aux | grep "postgres.*omma_db" | grep -v grep | wc -l
    
    echo "Conexiones 'idle in transaction':"
    ps aux | grep "postgres.*omma_db.*idle in transaction" | wc -l
    
    echo ""
    sleep 60
done
```

## Configuración Recomendada de PostgreSQL

En `/etc/postgresql/16/main/postgresql.conf`:

```conf
# Conexiones máximas (ajustar según recursos del servidor)
max_connections = 100

# Timeout para conexiones inactivas
idle_in_transaction_session_timeout = 600000  # 10 minutos en ms

# Timeout para statements
statement_timeout = 300000  # 5 minutos en ms

# Log de conexiones lentas
log_min_duration_statement = 1000  # Log queries > 1 segundo
```

Después de modificar, reiniciar PostgreSQL:
```bash
sudo systemctl restart postgresql
```

## Verificación Post-Despliegue

### Checklist:
- [ ] El backend inicia correctamente
- [ ] No hay errores en los logs
- [ ] Las conexiones se mantienen bajo 25
- [ ] No hay conexiones "idle in transaction" por más de 10 minutos
- [ ] Las operaciones de reserva funcionan correctamente
- [ ] Los pagos se procesan sin errores

### Logs a Revisar:
```bash
# Ver logs del backend
journalctl -u omma-be -f

# O si es manual:
tail -f /var/log/omma-be.log

# Buscar errores de conexión
grep -i "connection" /var/log/omma-be.log
grep -i "transaction" /var/log/omma-be.log
```

## Notas Adicionales

- **MaxOpenConns=25**: Suficiente para un servidor con tráfico moderado. Ajustar si es necesario.
- **MaxIdleConns=5**: Mantiene algunas conexiones listas para uso inmediato.
- **ConnMaxLifetime=5m**: Previene conexiones obsoletas o con problemas.
- **ConnMaxIdleTime=10m**: Libera conexiones que no se usan.

Si el tráfico aumenta significativamente, considera aumentar `MaxOpenConns` a 50 o más, pero siempre manteniéndolo por debajo de `max_connections` de PostgreSQL.
