package web

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

const categoryTag = "category-controller"

// CategoryController is springfoxdemo.boot.swagger.web.CategoryController.
type CategoryController struct{}

// NewCategoryController creates the controller.
func NewCategoryController() *CategoryController { return &CategoryController{} }

// Register mounts the controller's request mappings, in declaration order.
func (c *CategoryController) Register(app *web.App) {
	stringSchema := api.Schema{Type: api.TypeString}
	enumSchema := api.Schema{Type: api.TypeString, Enum: CategoryNames}

	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/category/Resource",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.search,
		Op: api.Operation{
			Name: "search", Tag: categoryTag, Summary: "search",
			Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "someEnum", In: api.InQuery, Description: "someEnum", Required: true,
				Schema: enumSchema,
			}},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &stringSchema}},
		},
	})

	petMap := api.Schema{Type: api.TypeObject, AdditionalProperties: &api.Schema{
		Type: api.TypeObject, AdditionalProperties: &api.Schema{Ref: "Pet"},
	}}
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/category/map",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.mapOfPets,
		Op: api.Operation{
			Name: "map", Tag: categoryTag, Summary: "map",
			Produces:  []string{web.MediaTypeAll},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK", Schema: &petMap}},
		},
	})

	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/category/{id}",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.someOperation,
		Op: api.Operation{
			Name: "someOperation", Tag: categoryTag, Summary: "someOperation",
			Consumes: []string{web.MediaTypeJSON}, Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{
				{
					Name: "id", In: api.InPath, Description: "id", Required: true,
					Schema: api.Schema{Type: api.TypeInteger, Format: "int64"},
				},
				{
					Name: "userId", In: api.InBody, Description: "userId", Required: true,
					Schema: api.Schema{Type: api.TypeInteger, Format: "int32"},
				},
			},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})

	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/category/{id}/{userId}",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.ignoredParam,
		Op: api.Operation{
			Name: "ignoredParam", Tag: categoryTag, Summary: "ignoredParam",
			Consumes: []string{web.MediaTypeJSON}, Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{
				{
					Name: "id", In: api.InPath, Description: "id", Required: true,
					Schema: api.Schema{Type: api.TypeInteger, Format: "int64"},
				},
				{
					// @ApiIgnore on the parameter: a Docket configured with
					// ignoredParameterTypes(ApiIgnore.class) drops it.
					Name: "userId", In: api.InPath, Description: "userId", Required: true,
					Schema: api.Schema{Type: api.TypeInteger, Format: "int32"}, Ignored: true,
				},
			},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})

	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/category/{id}/map",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.mapWithParams,
		Op: api.Operation{
			Name: "map", Tag: categoryTag, Summary: "map",
			Consumes: []string{web.MediaTypeJSON}, Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{
				{
					Name: "id", In: api.InPath, Description: "id", Required: true,
					Schema: stringSchema,
				},
				{
					// Springfox describes a Map<String,String> request
					// parameter with an `items` schema and no `type`.
					Name: "test", In: api.InQuery, Description: "test", Required: true,
					Schema: api.Schema{Items: &api.Schema{
						Type: api.TypeObject, AdditionalProperties: &stringSchema,
					}},
				},
			},
			Responses: []api.Response{{Code: http.StatusOK, Description: "OK"}},
		},
	})

	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/categories",
		Produces: []string{web.MediaTypeAll},
		Handler:  c.categories,
		Op: api.Operation{
			Name: "map", Tag: categoryTag, Summary: "map",
			Consumes: []string{web.MediaTypeJSON}, Produces: []string{web.MediaTypeAll},
			Params: []api.Parameter{{
				Name: "categories", In: api.InQuery, Description: "categories", Required: false,
				Schema:           api.Schema{Type: api.TypeArray, Items: &stringSchema},
				CollectionFormat: "multi",
			}},
			Responses: []api.Response{{
				Code: http.StatusOK, Description: "OK",
				Schema: &api.Schema{Type: api.TypeArray, Items: &enumSchema},
			}},
		},
	})
}

// search answers GET /category/Resource, returning the enum constant's name.
func (c *CategoryController) search(r *web.Request) (web.ResponseEntity, error) {
	raw, err := r.RequireQuery("someEnum")
	if err != nil {
		return web.ResponseEntity{}, err
	}
	someEnum, ok := ParseCategory(raw)
	if !ok {
		// Spring's enum converter failure is a 400.
		return web.ResponseEntity{}, web.ErrBadRequest
	}
	return web.OKText(someEnum.Name()), nil
}

// mapOfPets answers GET /category/map with a freshly created, always empty map.
func (c *CategoryController) mapOfPets(*web.Request) (web.ResponseEntity, error) {
	return web.OK(map[string]map[string]any{}), nil
}

// someOperation answers POST /category/{id} with an empty 200.
func (c *CategoryController) someOperation(*web.Request) (web.ResponseEntity, error) {
	return web.OKEmpty(), nil
}

// ignoredParam answers POST /category/{id}/{userId} with an empty 200.
func (c *CategoryController) ignoredParam(*web.Request) (web.ResponseEntity, error) {
	return web.OKEmpty(), nil
}

// mapWithParams answers POST /category/{id}/map with an empty 200.
func (c *CategoryController) mapWithParams(*web.Request) (web.ResponseEntity, error) {
	return web.OKEmpty(), nil
}

// categories answers POST /categories with an empty 200: the Java handler
// returns ResponseEntity.ok(null).
func (c *CategoryController) categories(*web.Request) (web.ResponseEntity, error) {
	return web.OKEmpty(), nil
}
