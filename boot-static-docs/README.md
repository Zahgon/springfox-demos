## boot-static-docs

__Experimental__
- An application with a default swagger2 configuration
- A test which uses `internal/staticdocs` to generate Asciidoc source from the
  application's JSON API.

To generate the asciidoc documentation run:

```bash
go test ./boot-static-docs -run TestStaticDocs
```

Rendering the generated Asciidoc to PDF and HTML was done by the
asciidoctor-gradle-plugin and has no Go counterpart; it is out of scope.

### Running the app
```bash
go run ./boot-static-docs
```

http://localhost:8080/v2/api-docs
http://localhost:8080/swagger-ui/index.html
