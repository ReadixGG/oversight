# OverSight Game Server

## Requirements
- Go 1.23+ (for local development)
- Docker (for deployment)

## Local Development
```bash
cd server
go mod tidy
go run . -addr :8080 -tick 20
```

## Docker Deployment
```bash
cd server
docker compose up -d --build
```

## Health Check
```
GET http://localhost:8080/health
```

## WebSocket Endpoint
```
ws://localhost:8080/ws
```
