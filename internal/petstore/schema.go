package petstore

import "github.com/springfox/springfox-demos/internal/api"

// Springfox derived these model definitions by reflecting over the petstore
// beans together with Jackson's serialisation rules, splitting a model into
// request and response variants wherever the two differ: the derived
// `identifier` property has a getter but no setter, so it appears in the
// response variant only, and Category's @JsonCreator marks `id` required in the
// request variant only. Go has no equivalent reflection, so the same
// definitions are declared as data.

func int64Schema() api.Schema  { return api.Schema{Type: api.TypeInteger, Format: "int64"} }
func int32Schema() api.Schema  { return api.Schema{Type: api.TypeInteger, Format: "int32"} }
func stringSchema() api.Schema { return api.Schema{Type: api.TypeString} }
func arrayOf(s api.Schema) api.Schema {
	return api.Schema{Type: api.TypeArray, Items: &s}
}

// orderStatusValues are the @ApiModelProperty allowable values of Order.status,
// split from the literal "placed, approved, delivered" and trimmed.
var orderStatusValues = []string{"placed", "approved", "delivered"}

// userStatusValues are the @ApiModelProperty allowable values of
// User.userStatus. Springfox keeps them on the query parameter but drops them
// from the model property, whose type is an int.
var userStatusValues = []string{"1-registered", "2-active", "3-closed"}

// petStatusSchema carries the @ApiModelProperty allowable values of Pet.status.
func petStatusSchema() api.Schema {
	return api.Schema{Type: api.TypeString, Enum: []string{"available", "pending", "sold"}}
}

func categoryModel(name string, requireID bool) api.Model {
	m := api.Model{
		Name: name,
		Properties: []api.Property{
			{Name: "id", Schema: int64Schema(), Required: requireID},
			{Name: "name", Schema: stringSchema()},
		},
	}
	return m
}

func tagModel() api.Model {
	return api.Model{
		Name: "Tag",
		Properties: []api.Property{
			{Name: "id", Schema: int64Schema()},
			{Name: "name", Schema: stringSchema()},
		},
	}
}

func petModel(name, categoryRef string, withIdentifier bool) api.Model {
	props := []api.Property{
		{Name: "category", Schema: api.Schema{Ref: categoryRef}},
		{Name: "id", Schema: int64Schema()},
	}
	if withIdentifier {
		props = append(props, api.Property{Name: "identifier", Schema: int64Schema()})
	}
	props = append(props,
		api.Property{Name: "name", Schema: stringSchema()},
		api.Property{Name: "photoUrls", Schema: arrayOf(stringSchema())},
		api.Property{
			Name:        "status",
			Description: "pet status in the store",
			Schema:      petStatusSchema(),
		},
		api.Property{Name: "tags", Schema: arrayOf(api.Schema{Ref: "Tag"})},
	)
	return api.Model{Name: name, Properties: props}
}

func orderModel() api.Model {
	return api.Model{
		Name: "Order",
		Properties: []api.Property{
			{Name: "complete", Schema: api.Schema{Type: api.TypeBoolean}},
			{Name: "id", Schema: int64Schema()},
			{Name: "identifier", Schema: int64Schema()},
			{Name: "petId", Schema: int64Schema()},
			{Name: "quantity", Schema: int32Schema()},
			{Name: "shipDate", Schema: api.Schema{Type: api.TypeString, Format: "date-time"}},
			{
				Name:        "status",
				Description: "Order Status",
				Schema:      api.Schema{Type: api.TypeString, Enum: orderStatusValues},
			},
		},
	}
}

func userModel(name string, withIdentifier bool) api.Model {
	props := []api.Property{
		{Name: "email", Schema: stringSchema()},
		{Name: "firstName", Schema: stringSchema()},
		{Name: "id", Schema: int64Schema()},
	}
	if withIdentifier {
		props = append(props, api.Property{Name: "identifier", Schema: stringSchema()})
	}
	props = append(props,
		api.Property{Name: "lastName", Schema: stringSchema()},
		api.Property{Name: "password", Schema: stringSchema()},
		api.Property{Name: "phone", Schema: stringSchema()},
		api.Property{
			Name:        "userStatus",
			Description: "User Status",
			Schema:      int32Schema(),
		},
		api.Property{Name: "username", Schema: stringSchema()},
	)
	return api.Model{Name: name, Properties: props}
}

// RegisterModels adds every petstore model, in both its request and response
// variants, to an application's registry.
func RegisterModels(reg *api.Registry) {
	reg.Add(categoryModel("Category", false))
	reg.Add(categoryModel("CategoryReq", true))
	reg.Add(categoryModel("CategoryRes", false))
	reg.Add(tagModel())
	reg.Add(petModel("Pet", "Category", true))
	reg.Add(petModel("PetReq", "CategoryReq", false))
	reg.Add(petModel("PetRes", "CategoryRes", true))
	reg.Add(orderModel())
	reg.Add(userModel("UserReq", false))
	reg.Add(userModel("UserRes", true))
}
