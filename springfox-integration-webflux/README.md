## springfox-integration-webflux
Demonstrates documenting message-gateway endpoints on a reactive project.

### Running
```
go run ./springfox-integration-webflux
```

### Endpoints
- `POST /conversions/upper` (`text/plain`) — upper-cases the request body
- `GET /conversions/pathvariable/{upperLower}?toConvert=…`
- `POST /conversions/lower` (`application/json`) — fails with 500; see
  `toLowerFlow` in application.go

### Swagger 2 Documentation
http://localhost:8080/v2/api-docs
