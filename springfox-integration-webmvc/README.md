## springfox-integration-webmvc
Demonstrates documenting message-gateway endpoints on a servlet project.

### Running
```
go run ./springfox-integration-webmvc
```

### Endpoints
- `POST /conversions/upper` (`text/plain`) — upper-cases the request body
- `POST /conversions/lower` (`application/json`) — lower-cases `bar` of a `Foo`
- `GET /conversions/pathvariable/{upperLower}?toConvert=…`
- `POST /conversion/controller` (`application/json`) — echoes a `Baz`

### Swagger 2 Documentation
http://localhost:8080/v2/api-docs
