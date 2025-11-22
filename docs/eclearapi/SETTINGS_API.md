# Settings API Endpoints

## Overview
The eClear API now includes endpoints for managing system settings including Parameters, Holidays, and Session Time.

## Endpoints

### Parameters

#### Get Parameter
Retrieve the current system parameters.

**URL:** `GET /parameter`

**Response:**
```json
{
  "status": "success",
  "message": "Parameter retrieved",
  "data": {
    "parameter": {
      "nid": 1732271234567,
      "description": "PME System Parameters",
      "flat_fee": 0.5,
      "lending_fee": 15.0,
      "borrowing_fee": 18.0,
      "max_quantity": 1000000.0,
      "borrow_max_open_day": 90,
      "denomination_limit": 100,
      "update": "2025-01-15 10:30:00"
    }
  }
}
```

#### Update Parameter
Update system parameters.

**URL:** `POST /parameter/update`

**Request Body:**
```json
{
  "description": "Updated PME System Parameters",
  "flat_fee": 0.5,
  "lending_fee": 15.0,
  "borrowing_fee": 18.0,
  "max_quantity": 1000000.0,
  "borrow_max_open_day": 90,
  "denomination_limit": 100
}
```

**Validation Rules:**
- All fee values must be non-negative
- `max_quantity` must be positive
- `borrow_max_open_day` must be positive
- `denomination_limit` must be positive

**Response:**
```json
{
  "status": "success",
  "message": "Parameter updated successfully",
  "data": {
    "parameter": { ... }
  }
}
```

### Holidays

#### Get Holidays
Retrieve all registered holidays.

**URL:** `GET /holiday/list`

**Response:**
```json
{
  "status": "success",
  "message": "Holidays retrieved",
  "data": {
    "count": 2,
    "holidays": [
      {
        "nid": 1732271234567,
        "tahun": 2025,
        "date": "2025-01-01",
        "description": "New Year's Day"
      },
      {
        "nid": 1732271234568,
        "tahun": 2025,
        "date": "2025-12-25",
        "description": "Christmas Day"
      }
    ]
  }
}
```

#### Add Holiday
Add a new holiday to the system.

**URL:** `POST /holiday/add`

**Request Body:**
```json
{
  "date": "2025-01-01",
  "description": "New Year's Day"
}
```

**Date Format:** `YYYY-MM-DD`

**Response:**
```json
{
  "status": "success",
  "message": "Holiday added successfully",
  "data": {
    "holiday": {
      "nid": 1732271234567,
      "tahun": 2025,
      "date": "2025-01-01",
      "description": "New Year's Day"
    }
  }
}
```

### Session Time

#### Get Session Time
Retrieve current trading session times.

**URL:** `GET /sessiontime`

**Response:**
```json
{
  "status": "success",
  "message": "Session time retrieved",
  "data": {
    "sessiontime": {
      "nid": 1732271234567,
      "description": "Regular Trading Hours",
      "session1_start": "09:00:00",
      "session1_end": "12:00:00",
      "session2_start": "13:30:00",
      "session2_end": "16:00:00",
      "update": "2025-01-15 10:30:00"
    }
  }
}
```

#### Update Session Time
Update trading session times.

**URL:** `POST /sessiontime/update`

**Request Body:**
```json
{
  "description": "Regular Trading Hours",
  "session1_start": "09:00:00",
  "session1_end": "12:00:00",
  "session2_start": "13:30:00",
  "session2_end": "16:00:00"
}
```

**Time Format:** `HH:MM:SS` or `HH:MM`

**Response:**
```json
{
  "status": "success",
  "message": "Session time updated successfully",
  "data": {
    "sessiontime": { ... }
  }
}
```

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "status": "error",
  "message": "Error description",
  "error": "Detailed error message"
}
```

**Common HTTP Status Codes:**
- `200` - Success
- `400` - Bad Request (validation error)
- `500` - Internal Server Error

## Dashboard Integration

All these settings can be managed through the web dashboard at:
- `http://localhost:8081/`

The dashboard provides user-friendly forms for:
- **Parameters Tab** - Edit all system parameters in a grid layout
- **Holidays Tab** - Add new holidays and view existing ones in a table
- **Session Time Tab** - Set trading session start/end times

## Usage Examples

### Using cURL

**Get Parameters:**
```bash
curl http://localhost:8081/parameter
```

**Update Parameters:**
```bash
curl -X POST http://localhost:8081/parameter/update \
  -H "Content-Type: application/json" \
  -d '{
    "description": "PME System Parameters",
    "flat_fee": 0.5,
    "lending_fee": 15.0,
    "borrowing_fee": 18.0,
    "max_quantity": 1000000.0,
    "borrow_max_open_day": 90,
    "denomination_limit": 100
  }'
```

**Add Holiday:**
```bash
curl -X POST http://localhost:8081/holiday/add \
  -H "Content-Type: application/json" \
  -d '{
    "date": "2025-01-01",
    "description": "New Year'\''s Day"
  }'
```

**Update Session Time:**
```bash
curl -X POST http://localhost:8081/sessiontime/update \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Regular Trading Hours",
    "session1_start": "09:00:00",
    "session1_end": "12:00:00",
    "session2_start": "13:30:00",
    "session2_end": "16:00:00"
  }'
```

## Event Sourcing

All settings changes are committed to Kafka and synchronized across all services:
- Parameter updates trigger `SyncParameter` events
- Holiday additions trigger `SyncHoliday` events
- Session time updates trigger `SyncSessionTime` events

This ensures all services (OMS, APME API, DB Exporter) stay synchronized with the latest settings.
