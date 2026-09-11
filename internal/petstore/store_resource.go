package petstore

import (
	"net/http"
	"strconv"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// storeProduces is the class-level @RequestMapping(produces = …) of
// PetStoreResource.
var storeProduces = []string{web.MediaTypeJSON}

// storeTagDescription is the @Api(description = …) of PetStoreResource.
const storeTagDescription = "Operations about store"

// PetStoreResource is springfox.petstore.controller.PetStoreResource.
type PetStoreResource struct {
	orders *repository[int64, Order]
}

// NewPetStoreResource creates the controller. The Java field is static, so a
// single store is shared by every instance in a JVM; a Go application creates
// one controller, which has the same effect.
func NewPetStoreResource() *PetStoreResource {
	return &PetStoreResource{orders: newRepository(Order.Identifier)}
}

// orderBeanParams are the query parameters Spring derives from the `Order`
// command object of placeOrder, which is bound without @RequestBody and so
// binds from request parameters rather than from the body.
func orderBeanParams() []api.Parameter {
	return []api.Parameter{
		{Name: "complete", In: api.InQuery, Schema: api.Schema{Type: api.TypeBoolean}},
		{Name: "id", In: api.InQuery, Schema: int64Schema()},
		{Name: "identifier", In: api.InQuery, Schema: int64Schema()},
		{Name: "petId", In: api.InQuery, Schema: int64Schema()},
		{Name: "quantity", In: api.InQuery, Schema: int32Schema()},
		{Name: "shipDate", In: api.InQuery, Schema: api.Schema{Type: api.TypeString, Format: "date-time"}},
		{
			Name: "status", In: api.InQuery, Description: "Order Status",
			Schema: api.Schema{Type: api.TypeString, Enum: orderStatusValues},
		},
	}
}

// Register mounts the controller's request mappings.
func (c *PetStoreResource) Register(app *web.App) {
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/store/order/{orderId}", Produces: storeProduces,
		Handler: c.getOrderByID,
		Op: api.Operation{
			// @ApiOperation(tags = {"Pet Store"}) overrides the controller's
			// tag on the operation, but the document's tags array still lists
			// only the controller's own.
			Name: "getOrderById", Tag: "Pet Store",
			ListedTag: "pet-store-resource", ListedTagDescription: storeTagDescription,
			Summary: "Find purchase order by ID",
			Notes: "For valid response try integer IDs with value <= 5 or > 10. " +
				"Other values will generated exceptions",
			Produces: storeProduces,
			Params: []api.Parameter{{
				Name: "orderId", In: api.InPath, Description: "ID of pet that needs to be fetched",
				Required: true, Schema: stringSchema(), Range: &api.Range{Min: 1, Max: 5},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "Order"}},
				{Code: http.StatusBadRequest, Description: "Invalid ID supplied"},
				{Code: http.StatusNotFound, Description: "Order not found"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/api/store/order",
		Produces: storeProduces,
		Handler:  c.placeOrder,
		Op: api.Operation{
			Name: "placeOrder", Tag: "pet-store-resource", ListedTagDescription: storeTagDescription,
			Summary:  "Place an order for a pet",
			Consumes: []string{web.MediaTypeJSON}, Produces: storeProduces,
			Params: orderBeanParams(),
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid Order"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodDelete, Pattern: "/api/store/order/{orderId}", Produces: storeProduces,
		Handler: c.deleteOrder,
		Op: api.Operation{
			Name: "deleteOrder", Tag: "pet-store-resource", ListedTagDescription: storeTagDescription,
			Summary: "Delete purchase order by ID",
			Notes: "For valid response try integer IDs with value < 1000. " +
				"Anything above 1000 or non-integers will generate API errors",
			Produces: storeProduces,
			Params: []api.Parameter{{
				Name: "orderId", In: api.InPath, Description: "ID of the order that needs to be deleted",
				Required: true, Schema: stringSchema(), Range: &api.Range{Min: 1, Max: -1},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid ID supplied"},
				{Code: http.StatusNotFound, Description: "Order not found"},
			},
		},
	})

	// Two mappings share /api/store/search, distinguished only by a request
	// parameter condition, and both throw UnsupportedOperationException.
	// Springfox documents only one operation per path and method, and which of
	// the two it keeps depends on the order the JVM reports the class's
	// declared methods in — an order the language specification leaves
	// undefined. The original is not self-consistent about it: two of its three
	// documents that cover this path name getPetInCA and one names getPetInTx,
	// from a source file that declares getPetInTx first. The registration order
	// below follows the majority of the observed output rather than the source
	// text, because it is the output that is the contract.
	for _, search := range []struct{ method, state string }{
		{"getPetInCA", "CA"},
		{"getPetInTx", "TX"},
	} {
		state := search.state
		method := search.method
		app.Handle(web.Route{
			Method: http.MethodGet, Pattern: "/api/store/search", Produces: storeProduces,
			Params:  []web.ParamCondition{{Name: "x", Value: state}},
			Handler: func(*web.Request) (web.ResponseEntity, error) { return web.ResponseEntity{}, errUnsupported{} },
			Op: api.Operation{
				Name: method, Tag: "pet-store-resource", ListedTagDescription: storeTagDescription,
				Summary:  method,
				Produces: storeProduces,
				Params: []api.Parameter{{
					// A `params = "x=TX"` condition surfaces as a required
					// query parameter with the matched value as its default.
					Name: "x", In: api.InQuery, Required: true, FromCondition: true,
					Schema: api.Schema{Type: api.TypeString, Default: state, Enum: []string{state}},
				}},
				Responses: []api.Response{
					{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "Pet"}},
				},
			},
		})
	}
}

// errUnsupported is java.lang.UnsupportedOperationException, which is unmapped
// and therefore surfaces as a 500.
const unsupportedOperationExceptionName = "UnsupportedOperationException"

type errUnsupported struct{}

func (errUnsupported) Error() string { return unsupportedOperationExceptionName }

func (c *PetStoreResource) getOrderByID(r *web.Request) (web.ResponseEntity, error) {
	id, err := strconv.ParseInt(r.PathVar("orderId"), 10, 64)
	if err != nil {
		return web.ResponseEntity{}, err
	}
	order, ok := c.orders.get(id)
	if !ok {
		return web.ResponseEntity{}, notFound()
	}
	return web.OK(order), nil
}

// placeOrder binds its Order from request parameters, not from the body: the
// Java signature has no @RequestBody, so a JSON payload is ignored entirely and
// an all-defaults Order is stored under identifier 0.
func (c *PetStoreResource) placeOrder(r *web.Request) (web.ResponseEntity, error) {
	order := Order{}
	if v, ok := r.Query("id"); ok {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			order.ID = id
		}
	}
	if v, ok := r.Query("petId"); ok {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			order.PetID = id
		}
	}
	if v, ok := r.Query("quantity"); ok {
		if q, err := strconv.ParseInt(v, 10, 32); err == nil {
			order.Quantity = int32(q)
		}
	}
	if v, ok := r.Query("status"); ok {
		order.Status = &v
	}
	if v, ok := r.Query("complete"); ok {
		order.Complete = v == "true"
	}
	c.orders.add(order)
	return web.OKText(""), nil
}

func (c *PetStoreResource) deleteOrder(r *web.Request) (web.ResponseEntity, error) {
	id, err := strconv.ParseInt(r.PathVar("orderId"), 10, 64)
	if err != nil {
		return web.ResponseEntity{}, err
	}
	c.orders.delete(id)
	return web.OKText(""), nil
}
