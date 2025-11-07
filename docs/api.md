# API Reference

Base URL: `http://localhost:8080`

All responses are JSON. Errors follow:

```json
{
  "error": "Human readable label",
  "details": "Optional detail",
  "suggestion": "Optional recovery hint"
}
```

## GET /api/health

Returns service heartbeat information.

**Response 200**

```json
{
  "status": "ok",
  "timestamp": "2025-11-07T07:45:00Z"
}
```

## GET /api/pack-sizes

Fetches the currently configured pack sizes.

**Response 200**

```json
{
  "packSizes": [250, 500, 1000, 2000, 5000],
  "updatedAt": "2025-11-07T07:40:00Z"
}
```

**Errors**

- `500 Internal Server Error` – storage read failure (unexpected).

## PUT /api/pack-sizes

Updates active pack sizes. Accepts 1–10 positive integers; duplicates are removed automatically.

**Request Body**

```json
{
  "packSizes": [23, 31, 53]
}
```

**Response 200**

```json
{
  "packSizes": [23, 31, 53],
  "updatedAt": "2025-11-07T07:50:00Z",
  "message": "Pack sizes updated successfully"
}
```

**Validation Errors (400)**

- Missing or empty `packSizes` array.
- Non-positive integers or more than 10 distinct values.

**Other Errors**

- `500 Internal Server Error` – storage failure.

## POST /api/calculate

Runs the DP algorithm for a requested number of items.

**Request Body**

```json
{
  "items": 500000
}
```

**Response 200**

```json
{
  "items": 500000,
  "packs": {
    "53": 9429,
    "31": 7,
    "23": 2
  },
  "totalPacks": 9438,
  "totalItems": 500000,
  "remainder": 0,
  "calculationTimeMs": 151
}
```

**Validation Errors**

- `400 Bad Request` – `items` must be a positive integer (rejects zero/negative) or payload is malformed JSON.

**Domain Errors**

- `422 Unprocessable Entity` – impossible to fulfill exactly with current sizes (includes explanatory message and a `suggestion` describing how to resolve it).

**Rate Limit Errors**

- `429 Too Many Requests` – returned when the global rate limiter is exceeded; retry after a short pause.

**Server Errors**

- `500 Internal Server Error` – unexpected calculator/storage issues.

## Headers & Middleware

- `X-Request-ID` is read from inbound requests (if provided) and always echoed back.
- CORS: `Access-Control-Allow-Origin: *`, with OPTIONS preflight for `GET/POST/PUT`.
- All responses are `application/json`.
