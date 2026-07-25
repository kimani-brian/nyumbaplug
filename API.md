# Kenyan House-Hunting Platform API Documentation

## Auth Endpoints

### `POST /api/v1/auth/register`
* **Role**: Public
* **Payload**:
```json
{
  "phone": "+254712345678",
  "email": "user@example.com",
  "password": "Password123!",
  "role": "landlord", // "landlord", "tenant", or "admin"
  "national_id_number": "12345678", // Required for landlord
  "is_caretaker": false,
  "authorized_by_landlord_id": null
}