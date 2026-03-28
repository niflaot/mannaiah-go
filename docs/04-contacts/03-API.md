# Contacts — HTTP API

All contacts endpoints are mounted under `/contacts` and require a valid bearer token. Permissions
follow the `contact:*` scope family — see [03-auth/01-AUTH.md](../03-auth/01-AUTH.md) for the full
scope reference.

---

## Endpoints

### Create a Contact

```
POST /contacts
Permission: contact:manage
```

**Request body**

```json
{
  "documentType": "CC",
  "documentNumber": "1234567890",
  "firstName": "Ana",
  "lastName": "García",
  "email": "ana.garcia@example.com",
  "phone": "+57 310 000 0000",
  "address": "Calle 123 # 45-67",
  "addressExtra": "Apto 8",
  "cityCode": "BOG",
  "metadata": {
    "woo_customer_id": "42"
  }
}
```

Use `legalName` instead of `firstName`+`lastName` for businesses:

```json
{
  "documentType": "NIT",
  "documentNumber": "900123456-1",
  "legalName": "Empresa S.A.S.",
  "email": "facturacion@empresa.co"
}
```

**Response** — `201 Created` with the full `Contact` object.

---

### List Contacts

```
GET /contacts
Permission: contact:view
```

**Query parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| `page` | `int` | Page number (default `1`) |
| `limit` | `int` | Records per page (default `20`) |
| `orderBy` | `string` | Field to sort (e.g. `createdAt`) |
| `orderDir` | `string` | `asc` or `desc` |
| `email` | `string` | Email filter |
| `documentType` | `string` | Filter by document type |
| `documentNumber` | `string` | Exact document number |
| `metadataKey` | `string` | Filter contacts with this metadata key |
| `metadataValue` | `string` | Require this exact value for `metadataKey` |

**Response** — `200 OK`

```json
{
  "data": [ /* Contact[] */ ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 142,
    "totalPages": 8
  }
}
```

---

### Get a Contact

```
GET /contacts/:id
Permission: contact:view
```

**Response** — `200 OK` with the full `Contact` object, or `404 Not Found`.

---

### Update a Contact

```
PATCH /contacts/:id
Permission: contact:manage
```

All fields are optional. Only provided fields are updated.

```json
{
  "phone": "+57 320 111 2222",
  "metadata": {
    "crm_segment": "vip"
  }
}
```

**Response** — `200 OK` with the updated `Contact` object.

---

### Delete a Contact

```
DELETE /contacts/:id
Permission: contact:manage
```

**Response** — `200 OK` (empty body). Returns `404 Not Found` if the contact does not exist.

---

## Contact Object Schema

```json
{
  "_id": "uuid",
  "documentType": "CC",
  "documentNumber": "1234567890",
  "legalName": "",
  "firstName": "Ana",
  "lastName": "García",
  "email": "ana.garcia@example.com",
  "phone": "+57 310 000 0000",
  "address": "Calle 123 # 45-67",
  "addressExtra": "Apto 8",
  "cityCode": "BOG",
  "metadata": {
    "woo_customer_id": "42"
  },
  "createdAt": "2026-01-15T10:00:00Z",
  "updatedAt": "2026-03-01T08:30:00Z"
}
```

---

## Error Reference

| HTTP Status | Condition |
|-------------|-----------|
| `400` | Validation failure (missing email, invalid name convention) |
| `401` | Missing or invalid bearer token |
| `403` | Token lacks required scope |
| `404` | Contact not found by ID |
| `409` | Duplicate email or document number |
| `500` | Internal server error |
