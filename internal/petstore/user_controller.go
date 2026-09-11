package petstore

import (
	"net/http"

	"github.com/springfox/springfox-demos/internal/api"
	"github.com/springfox/springfox-demos/internal/web"
)

// userProduces is the class-level @RequestMapping(produces = …) of
// UserController.
var userProduces = []string{web.MediaTypeJSON}

// userTagDescription is the @Api(description = …) of UserController.
const userTagDescription = "Operations about user"

// UserController is springfox.petstore.controller.UserController.
type UserController struct {
	users *repository[string, User]
	// now supplies the millisecond clock loginUser stamps into its response.
	now func() int64
}

// NewUserController creates the controller with an empty repository.
func NewUserController(now func() int64) *UserController {
	return &UserController{
		users: newRepository(User.Identifier),
		now:   now,
	}
}

// userBeanParams are the query parameters Spring derives from the `User`
// command object of updateUser, which is bound without @RequestBody. The path
// variable is interleaved in the alphabetical position of its own name.
func userBeanParams() []api.Parameter {
	return []api.Parameter{
		{Name: "email", In: api.InQuery, Schema: stringSchema()},
		{Name: "firstName", In: api.InQuery, Schema: stringSchema()},
		{Name: "id", In: api.InQuery, Schema: int64Schema()},
		{Name: "identifier", In: api.InQuery, Schema: stringSchema()},
		{Name: "lastName", In: api.InQuery, Schema: stringSchema()},
		{Name: "password", In: api.InQuery, Schema: stringSchema()},
		{Name: "phone", In: api.InQuery, Schema: stringSchema()},
		{
			Name: "userStatus", In: api.InQuery, Description: "User Status",
			Schema: api.Schema{Type: api.TypeInteger, Format: "int32", Enum: userStatusValues},
		},
		{
			Name: "username", In: api.InPath, Description: "name that need to be deleted",
			Required: true, Schema: stringSchema(), AllowReserved: &allowReservedFalse,
		},
	}
}

// allowReservedFalse is the explicit `allowReserved: false` Springfox writes on
// the path variable of an operation whose other parameters were expanded from a
// command object.
var allowReservedFalse = false

// Register mounts the controller's request mappings.
func (c *UserController) Register(app *web.App) {
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/api/user",
		Produces: userProduces,
		Handler:  c.createUser,
		Op: api.Operation{
			Name: "createUser", Tag: "user-controller", ListedTagDescription: userTagDescription,
			Summary:  "Create user",
			Notes:    "This can only be done by the logged in user.",
			Consumes: []string{web.MediaTypeJSON}, Produces: userProduces,
			Params: []api.Parameter{{
				Name: "user", In: api.InBody, Description: "Created user object",
				Required: true, Schema: api.Schema{Ref: "UserReq"},
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "UserRes"}},
			},
		},
	})
	// createUsersWithArrayInput binds a User[] without @RequestBody, so the
	// argument resolver produces null and the handler's for-loop throws.
	// Springfox cannot map the array either and documents its body as a string.
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/api/user/createWithArray",
		Produces: userProduces,
		Handler:  func(*web.Request) (web.ResponseEntity, error) { return web.ResponseEntity{}, errNullPointer{} },
		Op: api.Operation{
			Name: "createUsersWithArrayInput", Tag: "user-controller",
			ListedTagDescription: userTagDescription,
			Summary:              "Creates list of users with given input array",
			Consumes:             []string{web.MediaTypeJSON}, Produces: userProduces,
			Params: []api.Parameter{{
				Name: "users", In: api.InBody, Description: "List of user object",
				Required: true, Schema: stringSchema(),
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "UserRes"}},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodPost, Pattern: "/api/user/createWithList",
		Produces: userProduces,
		Handler:  func(*web.Request) (web.ResponseEntity, error) { return web.ResponseEntity{}, errNullPointer{} },
		Op: api.Operation{
			Name: "createUsersWithListInput", Tag: "user-controller",
			ListedTagDescription: userTagDescription,
			Summary:              "Creates list of users with given input array",
			Consumes:             []string{web.MediaTypeJSON}, Produces: userProduces,
			Params: []api.Parameter{{
				Name: "users", In: api.InBody, Description: "List of user object",
				Required: true, Schema: stringSchema(),
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodPut, Pattern: "/api/user/{username}",
		Produces: userProduces,
		Handler:  c.updateUser,
		Op: api.Operation{
			Name: "updateUser", Tag: "user-controller", ListedTagDescription: userTagDescription,
			Summary:  "Updated user",
			Notes:    "This can only be done by the logged in user.",
			Consumes: []string{web.MediaTypeJSON}, Produces: userProduces,
			Params: userBeanParams(),
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid user supplied"},
				{Code: http.StatusNotFound, Description: "User not found"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodDelete, Pattern: "/api/user/{username}", Produces: userProduces,
		Handler: c.deleteUser,
		Op: api.Operation{
			Name: "deleteUser", Tag: "user-controller", ListedTagDescription: userTagDescription,
			Summary:  "Delete user",
			Notes:    "This can only be done by the logged in user.",
			Produces: userProduces,
			Params: []api.Parameter{{
				Name: "username", In: api.InPath, Description: "The name that needs to be deleted",
				Required: true, Schema: stringSchema(),
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid username supplied"},
				{Code: http.StatusNotFound, Description: "User not found"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/user/{username}", Produces: userProduces,
		Handler: c.getUserByName,
		Op: api.Operation{
			Name: "getUserByName", Tag: "user-controller", ListedTagDescription: userTagDescription,
			Summary:  "Get user by user name",
			Produces: userProduces,
			Params: []api.Parameter{{
				Name: "username", In: api.InPath,
				Description: "The name that needs to be fetched. Use user1 for testing. ",
				Required:    true, Schema: stringSchema(),
			}},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Ref: "UserRes"}},
				{Code: http.StatusBadRequest, Description: "Invalid username supplied"},
				{Code: http.StatusNotFound, Description: "User not found"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/user/login", Produces: userProduces,
		Handler: c.loginUser,
		Op: api.Operation{
			Name: "loginUser", Tag: "user-controller", ListedTagDescription: userTagDescription,
			Summary:  "Logs user into the system",
			Produces: userProduces,
			Params: []api.Parameter{
				{
					Name: "username", In: api.InQuery, Description: "The user name for login",
					Required: true, Schema: stringSchema(),
				},
				{
					Name: "password", In: api.InQuery, Description: "The password for login in clear text",
					Required: true, Schema: stringSchema(),
				},
			},
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
				{Code: http.StatusBadRequest, Description: "Invalid username/password supplied"},
			},
		},
	})
	app.Handle(web.Route{
		Method: http.MethodGet, Pattern: "/api/user/logout", Produces: userProduces,
		Handler: func(*web.Request) (web.ResponseEntity, error) { return web.OKEmpty(), nil },
		Op: api.Operation{
			Name: "logoutUser", Tag: "user-controller",
			ListedTagDescription: userTagDescription,
			Summary:              "Logs out current logged in user session",
			Produces:             userProduces,
			Responses: []api.Response{
				{Code: http.StatusOK, Description: "OK", Schema: &api.Schema{Type: api.TypeString}},
			},
		},
	})
}

// errNullPointer is java.lang.NullPointerException, which is unmapped and
// therefore surfaces as a 500.
const nullPointerExceptionName = "NullPointerException"

type errNullPointer struct{}

func (errNullPointer) Error() string { return nullPointerExceptionName }

func (c *UserController) createUser(r *web.Request) (web.ResponseEntity, error) {
	var user User
	if err := r.BindJSON(&user); err != nil {
		return web.ResponseEntity{}, err
	}
	c.users.add(user)
	return web.OK(user), nil
}

// updateUser binds its User from request parameters and from the URI template
// variable, not from the body: Spring's model-attribute binding includes URI
// template variables, so the stored user keeps its username and loses
// everything the client sent in the JSON body.
func (c *UserController) updateUser(r *web.Request) (web.ResponseEntity, error) {
	username := r.PathVar("username")
	if _, ok := c.users.get(username); !ok {
		return web.Status(http.StatusNotFound), nil
	}
	user := User{Username: &username}
	bindUserParams(r, &user)
	c.users.add(user)
	return web.OKEmpty(), nil
}

func bindUserParams(r *web.Request, user *User) {
	assign := func(name string, target **string) {
		if v, ok := r.Query(name); ok {
			*target = &v
		}
	}
	assign("firstName", &user.FirstName)
	assign("lastName", &user.LastName)
	assign("email", &user.Email)
	assign("password", &user.Password)
	assign("phone", &user.Phone)
}

func (c *UserController) deleteUser(r *web.Request) (web.ResponseEntity, error) {
	username := r.PathVar("username")
	if !c.users.exists(username) {
		return web.Status(http.StatusNotFound), nil
	}
	c.users.delete(username)
	return web.OKEmpty(), nil
}

func (c *UserController) getUserByName(r *web.Request) (web.ResponseEntity, error) {
	user, ok := c.users.get(r.PathVar("username"))
	if !ok {
		return web.ResponseEntity{}, notFound()
	}
	return web.OK(user), nil
}

func (c *UserController) loginUser(r *web.Request) (web.ResponseEntity, error) {
	if _, err := r.RequireQuery("username"); err != nil {
		return web.ResponseEntity{}, err
	}
	if _, err := r.RequireQuery("password"); err != nil {
		return web.ResponseEntity{}, err
	}
	return web.OKText("logged in user session:" + web.JavaLongToString(c.now())), nil
}

// Register mounts all three petstore controllers and their models.
func Register(app *web.App, now func() int64) {
	RegisterModels(app.Models)
	NewPetController().Register(app)
	NewPetStoreResource().Register(app)
	NewUserController(now).Register(app)
}
