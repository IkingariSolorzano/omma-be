# Omma Backend Testing Guide

Esta guía te ayudará a probar la API del backend de Omma usando la colección de Postman incluida.

## Configuración Inicial

### 1. Importar la Colección
1. Abre Postman
2. Haz clic en "Import"
3. Selecciona el archivo `postman_collection.json`
4. La colección "Omma Backend API" aparecerá en tu workspace

### 2. Variables de Entorno
La colección incluye variables automáticas:
- `base_url`: http://localhost:8080/api/v1
- `auth_token`: Se establece automáticamente al hacer login

## Flujo de Pruebas Recomendado

### Paso 1: Autenticación
1. **Login Admin**: Usa las credenciales por defecto
   - Email: admin@omma.com
   - Password: admin123
   - El token se guarda automáticamente

### Paso 2: Configuración Inicial (Como Admin)
1. **Create Space**: Crea un consultorio
2. **Create Schedule**: Define horarios para el consultorio
3. **Create User**: Crea un usuario profesional
4. **Add Credits**: Asigna créditos al usuario

### Paso 3: Pruebas de Pagos (Como Admin)
1. **Register Payment**: Registra un pago por transferencia
   - user_id: 2, amount: 600.00, credits: 12
2. **Register Cash Payment**: Registra un pago en efectivo
   - user_id: 2, amount: 300.00, credits: 6
3. **Get All Payments**: Consulta todos los pagos
4. **Get User Payment History**: Historial de un usuario específico

### Paso 4: Pruebas de Dashboard (Como Admin)
1. **Get Dashboard Stats**: Estadísticas completas del sistema
2. **Get Recent Activity**: Actividad reciente

### Paso 5: Operaciones de Usuario
1. **Login Professional**: Inicia sesión como profesional
2. **Get Profile**: Verifica la información del usuario
3. **Get Credits**: Consulta créditos disponibles
4. **Get Spaces**: Ve los consultorios disponibles
5. **Create Reservation**: Hace una reserva
6. **Get My Reservations**: Consulta reservas

### Paso 6: Pruebas de Calendario
1. **Get Calendar - Week View**: Vista semanal de reservas
2. **Get Calendar - Month View**: Vista mensual
3. **Get Calendar - Custom Range**: Rango personalizado
4. **Get Calendar - Filtered by Spaces**: Filtrado por espacios
5. **Get Available Slots - Today**: Espacios disponibles hoy
6. **Get Available Slots - Specific Space**: Disponibilidad de un espacio

### Paso 7: Gestión de Reservas (Como Admin)
1. **Login Admin**: Vuelve a iniciar sesión como admin
2. **Get Pending Reservations**: Ve reservas pendientes de aprobación
3. **Approve Reservation**: Aprueba una reserva

## Nuevas Funcionalidades - Endpoints Agregados

### 📅 **Calendario**
- `GET /calendar` - Obtener reservas con filtros avanzados
- `GET /calendar/available` - Espacios disponibles por fecha

**Parámetros de filtrado:**
- `period`: week, month, day, custom
- `start_date`: Fecha inicio (YYYY-MM-DD)
- `end_date`: Fecha fin (solo para custom)
- `space_ids`: IDs de espacios separados por coma (1,2,3)

### 💰 **Pagos**
- `POST /admin/payments` - Registrar pago
- `GET /admin/payments` - Historial de pagos
- `GET /admin/payments?user_id=X` - Historial de usuario específico

### 📊 **Dashboard**
- `GET /admin/dashboard/stats` - Estadísticas completas
- `GET /admin/dashboard/activity` - Actividad reciente

## Datos de Ejemplo

### Usuario Profesional
```json
{
    "email": "doctor@example.com",
    "password": "password123",
    "name": "Dr. Juan Pérez",
    "phone": "+52 555 123 4567",
    "specialty": "Cardiología",
    "description": "Especialista en cardiología con 10 años de experiencia"
}
```

### Consultorio
```json
{
    "name": "Consultorio A",
    "description": "Consultorio amplio con vista al jardín",
    "capacity": 1,
    "cost_credits": 6
}
```

### Horario
```json
{
    "space_id": 1,
    "day_of_week": 1,
    "start_time": "09:00",
    "end_time": "18:00"
}
```

### Pago
```json
{
    "user_id": 2,
    "amount": 600.00,
    "credits": 12,
    "payment_method": "transfer",
    "reference": "TXN123456789",
    "notes": "Pago por transferencia bancaria"
}
```

## Estadísticas del Dashboard

El endpoint `/admin/dashboard/stats` retorna métricas completas:

### Usuarios
- `users_registered`: Total de usuarios registrados
- `users_with_credits`: Usuarios con créditos activos
- `users_with_expiring_credits`: Créditos que vencen en 7 días
- `users_with_expired_credits`: Usuarios con créditos vencidos
- `users_without_credits`: Usuarios sin créditos

### Espacios y Reservas
- `total_spaces`: Total de consultorios
- `total_hours_per_day/week`: Horas disponibles
- `spaces_reserved_today/this_week`: Espacios reservados
- `spaces_available_today/this_week`: Espacios disponibles

### Finanzas
- `credits_this_month/last_month`: Créditos vendidos
- `revenue_this_month/last_month`: Ingresos por ventas
- `cancellations_this_month/last_month`: Total cancelaciones
- `penalty_credits_this_month/last_month`: Créditos por penalización

## Solución de Problemas

### Error 401 Unauthorized
- Verifica que el token esté configurado correctamente
- Haz login nuevamente si el token expiró

### Error 403 Forbidden
- Asegúrate de estar usando una cuenta admin para endpoints administrativos

### Error 400 Bad Request
- Revisa que los datos JSON estén correctamente formateados
- Verifica que todos los campos requeridos estén presentes

### Error de Conexión
- Confirma que el servidor esté corriendo en el puerto 8080
- Verifica que la base de datos PostgreSQL esté disponible

### Problemas con Fechas
- Usa formato ISO 8601: `2024-01-15T10:00:00Z`
- Para filtros de calendario usa formato: `2024-01-15`

## Datos de Ejemplo

### Horarios (day_of_week)
- 0 = Domingo
- 1 = Lunes  
- 2 = Martes
- 3 = Miércoles
- 4 = Jueves
- 5 = Viernes
- 6 = Sábado

### Formato de Fechas
```json
{
    "start_time": "2024-01-15T10:00:00Z",
    "end_time": "2024-01-15T11:00:00Z"
}
```

### Créditos
- Siempre en múltiplos de 6
- 6 créditos = 1 hora de consultorio
- 2 créditos = penalización por cancelación tardía

## Casos de Prueba Importantes

### ✅ Casos Exitosos
1. **Reserva dentro de horario** → Se confirma automáticamente
2. **Cancelación > 24 horas** → Sin penalización
3. **Créditos suficientes** → Reserva exitosa

### ❌ Casos de Error
1. **Reserva fuera de horario** → Requiere aprobación
2. **Cancelación < 24 horas** → Penalización de 2 créditos
3. **Créditos insuficientes** → Error 400
4. **Horario ocupado** → Conflicto de reserva

## Endpoints Públicos

- `GET /api/v1/professionals` → No requiere autenticación
- Solo muestra profesionales con créditos activos

## Troubleshooting

### Error 401 Unauthorized
- Verifica que el token esté configurado
- Haz login nuevamente

### Error 403 Forbidden  
- Endpoint requiere rol admin
- Usa "Login Admin" antes de endpoints admin

### Error 400 Bad Request
- Revisa el formato JSON
- Verifica que los IDs existan
- Créditos deben ser múltiplos de 6
