# API Contracts

## POST /api/v1/orders
Request
```json
{
  "orderId": "o-1",
  "pickup": "point-a",
  "dropoff": "point-b",
  "priority": "STANDARD"
}
```

Response
```json
{
  "status": "CREATED"
}
```

## POST /api/v1/assignments
Request
```json
{
  "orderId": "o-1",
  "candidateIds": ["c-1", "c-2"]
}
```

Response
```json
{
  "assignmentId": "a-1",
  "courierId": "c-2",
  "etaMinutes": 28
}
```
