package springfox

import (
	"embed"
	"mime"
	"path"
	"strings"

	"github.com/springfox/springfox-demos/internal/web"
)

//go:embed swaggerui
var swaggerUIFS embed.FS

// SwaggerUIResources serves the swagger-ui distribution the original mounts
// from the springfox-swagger-ui webjar at
// classpath:/META-INF/resources/webjars/springfox-swagger-ui/.
//
// Only index.html is carried over: it is the page the resource handler and the
// "/swagger-ui/" view controller actually resolve, and it is the only part of
// the webjar this repository's behaviour depends on. The minified JavaScript
// and CSS bundle it links to is a third-party binary asset that is not part of
// this repository's source and is out of scope for the migration.
type SwaggerUIResources struct{}

// Open implements web.ResourceProvider.
func (SwaggerUIResources) Open(name string) (data []byte, mediaType string, ok bool) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || name == "." {
		return nil, "", false
	}
	b, err := swaggerUIFS.ReadFile("swaggerui/" + name)
	if err != nil {
		return nil, "", false
	}
	ct := mime.TypeByExtension(path.Ext(name))
	switch path.Ext(name) {
	case ".html":
		// The dispatcher appends the container's default charset.
		ct = web.MediaTypeTextHTML + ";charset="
	case "":
		ct = web.MediaTypeTextPlain
	}
	return b, ct, true
}
