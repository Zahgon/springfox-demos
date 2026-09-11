package boot

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// actuatorMediaTypes are the media types spring-boot-starter-actuator produces.
var actuatorMediaTypes = []string{
	web.MediaTypeJSON,
	"application/vnd.spring-boot.actuator.v2+json",
	"application/vnd.spring-boot.actuator.v3+json",
}

// actuatorContentType is the media type the actuator selects when a client
// sends no restrictive Accept header.
const actuatorContentType = "application/vnd.spring-boot.actuator.v3+json"

// ReactivePublisherModel is the model name Springfox derives from the reactive
// actuator handler's return type, which it leaves unresolved.
const ReactivePublisherModel = "Publisher«ResponseEntity«object»»"

// Link is org.springframework.boot.actuate.endpoint.web.Link.
type Link struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated"`
}

// RegisterActuator mounts the three endpoints spring-boot-starter-actuator
// exposes over the web by default: the links index, health and info.
//
// The servlet and reactive stacks differ in the tag names Springfox derives for
// them and in how the health path's trailing wildcard is spelled, so both
// variants are reproduced.
func RegisterActuator(app *web.App, baseURL string) {
	app.Models.Add(api.Model{
		Name: "Link",
		Properties: []api.Property{
			{Name: "href", Schema: api.Schema{Type: api.TypeString}},
			{Name: "templated", Schema: api.Schema{Type: api.TypeBoolean}},
		},
	})

	linksTag, handlerTag, healthWildcard := "web-mvc-links-handler", "operation-handler", "/actuator/health/**"
	if app.Stack == web.StackReactive {
		linksTag, handlerTag, healthWildcard = "web-flux-links-handler", "read-operation-handler", "/actuator/health/{*path}"
	}

	link := api.Schema{Ref: "Link"}
	inner := api.Schema{Type: api.TypeObject, AdditionalProperties: &link}
	linksSchema := api.Schema{Type: api.TypeObject, AdditionalProperties: &inner}

	app.Handle(web.Route{
		Method:   http.MethodGet,
		Pattern:  "/actuator",
		Produces: actuatorMediaTypes,
		Handler:  func(*web.Request) (web.ResponseEntity, error) { return actuatorLinks(baseURL, app.Stack), nil },
		Op: api.Operation{
			Name: "links", Tag: linksTag, Summary: "links",
			Produces:  actuatorMediaTypes,
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &linksSchema}},
		},
	})

	stringValue := api.Schema{Type: api.TypeString}
	bodySchema := api.Schema{Type: api.TypeObject, AdditionalProperties: &stringValue}
	// Springfox unwraps the servlet handler's Object return type but cannot
	// unwrap the reactive one, so it publishes the raw generic signature.
	replySchema := api.Schema{Type: api.TypeObject}
	if app.Stack == web.StackReactive {
		replySchema = api.Schema{Ref: ReactivePublisherModel}
	}
	handlerOp := func() api.Operation {
		s := replySchema
		return api.Operation{
			Name: "handle", Tag: handlerTag, Summary: "handle",
			Produces: actuatorMediaTypes,
			Params: []api.Parameter{{
				Name: "body", In: api.InBody, Description: "body", Required: false, Schema: bodySchema,
			}},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &s}},
		}
	}

	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/actuator/health", Produces: actuatorMediaTypes,
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(map[string]string{"status": "UP"}).WithContentType(actuatorContentType), nil
		},
		Op: handlerOp(),
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: healthWildcard, Produces: actuatorMediaTypes,
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(map[string]string{"status": "UP"}).WithContentType(actuatorContentType), nil
		},
		Op: handlerOp(),
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/actuator/info", Produces: actuatorMediaTypes,
		Handler: func(*web.Request) (web.ResponseEntity, error) {
			return web.OK(struct{}{}).WithContentType(actuatorContentType), nil
		},
		Op: handlerOp(),
	})
}

// actuatorLinks renders the /actuator index. The servlet and reactive stacks
// emit the same links in a different order, which the original's output shows.
func actuatorLinks(baseURL string, stack web.Stack) web.ResponseEntity {
	self := Link{Href: baseURL + "/actuator"}
	health := Link{Href: baseURL + "/actuator/health"}
	healthPath := Link{Href: baseURL + "/actuator/health/{*path}", Templated: true}
	info := Link{Href: baseURL + "/actuator/info"}

	links := orderedLinks{}
	if stack == web.StackReactive {
		links.add("self", self).add("health-path", healthPath).add("health", health).add("info", info)
	} else {
		links.add("self", self).add("health", health).add("health-path", healthPath).add("info", info)
	}
	return web.OK(map[string]orderedLinks{"_links": links}).WithContentType(actuatorContentType)
}

// orderedLinks serialises links in insertion order.
type orderedLinks struct {
	keys  []string
	links map[string]Link
}

func (o *orderedLinks) add(name string, l Link) *orderedLinks {
	if o.links == nil {
		o.links = map[string]Link{}
	}
	o.keys = append(o.keys, name)
	o.links[name] = l
	return o
}

// MarshalJSON writes the links in insertion order.
func (o orderedLinks) MarshalJSON() ([]byte, error) {
	keys, links := o.keys, o.links
	out := []byte("{")
	for i, k := range keys {
		if i > 0 {
			out = append(out, ',')
		}
		key, err := web.MarshalJSON(k)
		if err != nil {
			return nil, err
		}
		val, err := web.MarshalJSON(links[k])
		if err != nil {
			return nil, err
		}
		out = append(out, key...)
		out = append(out, ':')
		out = append(out, val...)
	}
	return append(out, '}'), nil
}
