package petstore

import (
	"net/http"
	"strconv"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// petProduces is the class-level @RequestMapping(produces = …) of PetController.
var petProduces = []string{web.MediaTypeJSON, web.MediaTypeXML}

// petTagDescription is the @Api(description = …) of PetController.
const petTagDescription = "Operations about pets"

// petSecurity is the @Authorization set every pet operation declares.
var petSecurity = []api.SecurityRequirement{
	{Name: "petstore_auth", Scopes: []string{"write:pets", "read:pets"}},
}

// PetController is springfox.petstore.controller.PetController.
type PetController struct {
	pets *repository[int64, Pet]
}

// NewPetController creates the controller with an empty repository, matching
// the `new PetRepository()` field initialiser.
func NewPetController() *PetController {
	return &PetController{pets: newRepository(Pet.Identifier)}
}

// notFound is springfox.petstore.controller.NotFoundException. Nothing maps it
// to a status, so Spring reports it as a 500 — which is what the original does
// for a missing pet, order or user.
func notFound() error { return errNotFound{} }

// notFoundExceptionName is the Java type the handlers raise; nothing maps it,
// so it becomes a 500.
const notFoundExceptionName = "NotFoundException"

type errNotFound struct{}

func (errNotFound) Error() string { return notFoundExceptionName }

// Register mounts the controller's request mappings.
func (c *PetController) Register(app *web.App) {
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/pet/{petId}", Produces: petProduces,
		Handler: c.getPetByID,
		Op: api.Operation{
			Name: "getPetById", Tag: "pet-controller", ListedTagDescription: petTagDescription,
			Summary: "Find pet by ID",
			Notes: "Returns a pet when ID < 10. ID > 10 or non-integers will simulate API " +
				"error conditions",
			Produces: petProduces,
			Params: []api.Parameter{{
				Name: "petId", In: api.InPath, Description: "ID of pet that needs to be fetched",
				Required: true, Schema: stringSchema(), Range: &api.Range{Min: 1, Max: 5},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "PetRes"}},
				{Code: http.StatusBadRequest, Description: "Invalid ID supplied", Headers: petIDHeaders(true)},
				{Code: http.StatusNotFound, Description: "Pet not found", Headers: petIDHeaders(false)},
			},
			Security: append([]api.SecurityRequirement{}, append(petSecurity, api.SecurityRequirement{Name: "api_key", Scopes: []string{}})...),
		},
	})
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/api/pet",
		Produces: petProduces,
		Handler:  c.addPet,
		Op: api.Operation{
			Name: "addPet", Tag: "pet-controller", ListedTagDescription: petTagDescription,
			Summary:  "Add a new pet to the store",
			Consumes: []string{web.MediaTypeJSON}, Produces: petProduces,
			Params: []api.Parameter{{
				Name: "pet", In: api.InBody, Description: "Pet object that needs to be added to the store",
				Required: true, Schema: api.Schema{Ref: "PetReq"},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusMethodNotAllowed, Description: "Invalid input"},
			},
			Security: petSecurity,
		},
	})
	app.Handle(web.Route{
		Method: http.MethodPut, Pattern: "/api/pet",
		Produces: petProduces,
		Handler:  c.updatePet,
		Op: api.Operation{
			Name: "updatePet", Tag: "pet-controller", ListedTagDescription: petTagDescription,
			Summary:  "Update an existing pet",
			Consumes: []string{web.MediaTypeJSON}, Produces: petProduces,
			Params: []api.Parameter{{
				Name: "pet", In: api.InBody, Description: "Pet object that needs to be added to the store",
				Required: true, Schema: api.Schema{Ref: "PetReq"},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid ID supplied"},
				{Code: http.StatusNotFound, Description: "Pet not found"},
				{Code: http.StatusMethodNotAllowed, Description: "Validation exception"},
			},
			Security: petSecurity,
		},
	})
	petArray := arrayOf(api.Schema{Ref: "PetRes"})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/pet/findByStatus", Produces: petProduces,
		Handler: c.findPetsByStatus,
		Op: api.Operation{
			Name: "findPetsByStatus", Tag: "pet-controller", ListedTagDescription: petTagDescription,
			Summary:  "Finds Pets by status",
			Notes:    "Multiple status values can be provided with comma-separated strings",
			Produces: petProduces,
			Params: []api.Parameter{{
				Name: "status", In: api.InQuery,
				Description: "Status values that need to be considered for filter", Required: true,
				Schema: api.Schema{
					Type:    api.TypeString,
					Default: "available",
					Enum:    []string{"available", "pending", "sold"},
				},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &petArray},
				{Code: http.StatusBadRequest, Description: "Invalid status value"},
			},
			Security: petSecurity,
		},
	})
	petArrayByTag := arrayOf(api.Schema{Ref: "PetRes"})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/pet/findByTags", Produces: petProduces,
		Handler: c.findPetsByTags,
		Op: api.Operation{
			Name: "findPetsByTags", Tag: "pet-controller", ListedTagDescription: petTagDescription,
			Summary: "Finds Pets by tags",
			Notes: "Multiple tags can be provided with comma-separated strings. " +
				"Use tag1, tag2, tag3 for testing.",
			Produces: petProduces,
			Params: []api.Parameter{{
				Name: "tags", In: api.InQuery, Description: "Tags to filter by",
				Required: true, Schema: stringSchema(),
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &petArrayByTag},
				{Code: http.StatusBadRequest, Description: "Invalid tag value"},
			},
			Security:   petSecurity,
			Deprecated: true,
		},
	})
	// @ApiOperation(hidden = true): the route is served but never documented.
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/pet/findPetsHidden", Produces: petProduces,
		Handler: c.findPetsByTags,
		Op:      api.Operation{Name: "findPetsHidden", Tag: "pet-controller", Hidden: true},
	})
}

// petIDHeaders are the @ResponseHeader declarations on getPetById. The
// operation-level headers come first; the 400 response adds its own two.
func petIDHeaders(withResponseSpecific bool) []api.Header {
	headers := []api.Header{
		{Name: "header4", Type: api.TypeString},
		{Name: "header3", Type: api.TypeString},
	}
	if withResponseSpecific {
		headers = append(headers,
			api.Header{Name: "header2", Type: api.TypeString},
			api.Header{Name: "header1", Type: api.TypeString})
	}
	return headers
}

func (c *PetController) getPetByID(r *web.Request) (web.ResponseEntity, error) {
	// Long.valueOf on a non-numeric path variable throws, and nothing maps
	// NumberFormatException either, so the response is a 500.
	id, err := strconv.ParseInt(r.PathVar("petId"), 10, 64)
	if err != nil {
		return web.ResponseEntity{}, err
	}
	pet, ok := c.pets.get(id)
	if !ok {
		return web.ResponseEntity{}, notFound()
	}
	return web.OK(pet), nil
}

func (c *PetController) addPet(r *web.Request) (web.ResponseEntity, error) {
	var pet Pet
	if err := r.BindJSON(&pet); err != nil {
		return web.ResponseEntity{}, err
	}
	c.pets.add(pet)
	return web.OKText("SUCCESS"), nil
}

func (c *PetController) updatePet(r *web.Request) (web.ResponseEntity, error) {
	return c.addPet(r)
}

func (c *PetController) findPetsByStatus(r *web.Request) (web.ResponseEntity, error) {
	status, err := r.RequireQuery("status")
	if err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OK(c.pets.where(statusIs(status))), nil
}

func (c *PetController) findPetsByTags(r *web.Request) (web.ResponseEntity, error) {
	tags, err := r.RequireQuery("tags")
	if err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OK(c.pets.where(tagsContain(tags))), nil
}
