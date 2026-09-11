package petstore

import (
	"net/http"
	"testing"

	"github.com/springfox/springfox-demos/internal/web"
)

// fixedClock is the millisecond clock loginUser stamps into its response.
func fixedClock() int64 { return 1788414439628 }

func newTestMvc(t *testing.T) *web.MockMvc {
	t.Helper()
	app := web.NewApp("petstore", web.StackServlet)
	Register(app, fixedClock)
	return web.NewMockMvc(app)
}

const doggie = `{"id":1,"name":"doggie","category":{"id":1,"name":"dog"},` +
	`"photoUrls":["u1"],"tags":[{"id":1,"name":"t1"}],"status":"available"}`

// TestPetLifecycle covers PetController's storage semantics, including the
// derived `identifier` property Jackson adds from getIdentifier().
func TestPetLifecycle(t *testing.T) {
	mvc := newTestMvc(t)

	added := mvc.Perform(web.Post("/api/pet").ContentType(web.MediaTypeJSON).Content(doggie))
	if added.Status != http.StatusOK || added.Body != "SUCCESS" {
		t.Fatalf("addPet = %d %q, want 200 SUCCESS", added.Status, added.Body)
	}
	// The class-level `produces` wins over the string body's default type.
	if added.ContentType != web.MediaTypeJSON {
		t.Errorf("addPet Content-Type = %q, want %q", added.ContentType, web.MediaTypeJSON)
	}

	got := mvc.Perform(web.Get("/api/pet/1"))
	want := `{"id":1,"category":{"id":1,"name":"dog"},"name":"doggie","photoUrls":["u1"],` +
		`"tags":[{"id":1,"name":"t1"}],"status":"available","identifier":1}`
	if got.Body != want {
		t.Errorf("getPetById = %s, want %s", got.Body, want)
	}

	// updatePet replaces the stored pet wholesale, so the fields a partial body
	// omits fall back to the Java field initialisers rather than being merged.
	if res := mvc.Perform(web.Put("/api/pet").ContentType(web.MediaTypeJSON).
		Content(`{"id":1,"name":"rex","status":"sold"}`)); res.Body != "SUCCESS" {
		t.Fatalf("updatePet = %q, want SUCCESS", res.Body)
	}
	replaced := mvc.Perform(web.Get("/api/pet/1"))
	wantReplaced := `{"id":1,"category":null,"name":"rex","photoUrls":[],"tags":[],` +
		`"status":"sold","identifier":1}`
	if replaced.Body != wantReplaced {
		t.Errorf("after updatePet = %s, want %s", replaced.Body, wantReplaced)
	}
}

// TestPetLookupFailures covers the two ways getPetById fails, both of which the
// original reports as a 500 because nothing maps NotFoundException or
// NumberFormatException.
func TestPetLookupFailures(t *testing.T) {
	mvc := newTestMvc(t)
	for _, path := range []string{"/api/pet/99", "/api/pet/abc"} {
		if res := mvc.Perform(web.Get(path)); res.Status != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusInternalServerError)
		}
	}
}

// TestFindPets covers the two search endpoints and their required parameters.
func TestFindPets(t *testing.T) {
	mvc := newTestMvc(t)
	mvc.Perform(web.Post("/api/pet").ContentType(web.MediaTypeJSON).Content(doggie))

	cases := []struct {
		path      string
		wantEmpty bool
	}{
		{"/api/pet/findByStatus?status=available", false},
		{"/api/pet/findByStatus?status=bogus", true},
		{"/api/pet/findByTags?tags=t1", false},
		{"/api/pet/findByTags?tags=nope", true},
	}
	for _, c := range cases {
		res := mvc.Perform(web.Get(c.path))
		if res.Status != http.StatusOK {
			t.Fatalf("%s: status = %d, want %d", c.path, res.Status, http.StatusOK)
		}
		if (res.Body == "[]") != c.wantEmpty {
			t.Errorf("%s: body = %s, wantEmpty = %v", c.path, res.Body, c.wantEmpty)
		}
	}
	for _, path := range []string{"/api/pet/findByStatus", "/api/pet/findByTags"} {
		if res := mvc.Perform(web.Get(path)); res.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusBadRequest)
		}
	}
}

// TestHiddenOperationIsStillServed covers @ApiOperation(hidden = true): the
// route works, it simply never appears in a document.
func TestHiddenOperationIsStillServed(t *testing.T) {
	res := newTestMvc(t).Perform(web.Get("/api/pet/findPetsHidden?tags=t1"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
}

// TestPlaceOrderBindsFromRequestParameters covers the consequence of
// placeOrder's missing @RequestBody: the JSON payload is ignored entirely and
// an all-defaults Order is stored under identifier 0.
func TestPlaceOrderBindsFromRequestParameters(t *testing.T) {
	mvc := newTestMvc(t)

	placed := mvc.Perform(web.Post("/api/store/order").ContentType(web.MediaTypeJSON).
		Content(`{"id":5,"petId":1,"quantity":2,"status":"placed","complete":false}`))
	if placed.Status != http.StatusOK || placed.Body != "" {
		t.Fatalf("placeOrder = %d %q, want 200 and an empty body", placed.Status, placed.Body)
	}

	// The order the client asked for was never stored.
	if res := mvc.Perform(web.Get("/api/store/order/5")); res.Status != http.StatusInternalServerError {
		t.Errorf("order 5: status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
	// An all-defaults order was stored under identifier 0 instead.
	zero := mvc.Perform(web.Get("/api/store/order/0"))
	want := `{"id":0,"petId":0,"quantity":0,"shipDate":null,"status":null,"complete":false,"identifier":0}`
	if zero.Body != want {
		t.Errorf("order 0 = %s, want %s", zero.Body, want)
	}

	// The same binding does work through request parameters.
	mvc.Perform(web.Post("/api/store/order?id=5&petId=1&quantity=2&status=placed"))
	if res := mvc.Perform(web.Get("/api/store/order/5")); res.Status != http.StatusOK {
		t.Errorf("after a parameter-bound order, status = %d, want %d", res.Status, http.StatusOK)
	}
	mvc.Perform(web.Delete("/api/store/order/5"))
	if res := mvc.Perform(web.Get("/api/store/order/5")); res.Status != http.StatusInternalServerError {
		t.Errorf("after deleteOrder, status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
}

// TestStoreSearchRequiresAKnownState covers the two mappings that share
// /api/store/search under a request-parameter condition and both throw.
func TestStoreSearchRequiresAKnownState(t *testing.T) {
	mvc := newTestMvc(t)
	if res := mvc.Perform(web.Get("/api/store/search")); res.Status != http.StatusBadRequest {
		t.Errorf("without x, status = %d, want %d", res.Status, http.StatusBadRequest)
	}
	for _, state := range []string{"TX", "CA"} {
		res := mvc.Perform(web.Get("/api/store/search?x=" + state))
		if res.Status != http.StatusInternalServerError {
			t.Errorf("x=%s: status = %d, want %d", state, res.Status, http.StatusInternalServerError)
		}
	}
}

// TestUserLifecycle covers UserController's storage semantics, including the
// URI-template binding that keeps the username on an otherwise empty update.
func TestUserLifecycle(t *testing.T) {
	mvc := newTestMvc(t)
	bob := `{"id":1,"username":"bob","firstName":"Bob","lastName":"B","email":"b@x",` +
		`"password":"p","phone":"1","userStatus":1}`

	created := mvc.Perform(web.Post("/api/user").ContentType(web.MediaTypeJSON).Content(bob))
	want := `{"id":1,"username":"bob","firstName":"Bob","lastName":"B","email":"b@x",` +
		`"password":"p","phone":"1","userStatus":1,"identifier":"bob"}`
	if created.Body != want {
		t.Fatalf("createUser = %s, want %s", created.Body, want)
	}
	if got := mvc.Perform(web.Get("/api/user/bob")); got.Body != want {
		t.Errorf("getUserByName = %s, want %s", got.Body, want)
	}

	// updateUser has no @RequestBody either, so the JSON payload is discarded
	// and only the URI template variable survives.
	updated := mvc.Perform(web.Put("/api/user/bob").ContentType(web.MediaTypeJSON).
		Content(`{"id":1,"username":"bob","firstName":"Bobby"}`))
	if updated.Status != http.StatusOK || updated.Body != "" {
		t.Fatalf("updateUser = %d %q, want 200 and an empty body", updated.Status, updated.Body)
	}
	after := mvc.Perform(web.Get("/api/user/bob"))
	wantAfter := `{"id":0,"username":"bob","firstName":null,"lastName":null,"email":null,` +
		`"password":null,"phone":null,"userStatus":0,"identifier":"bob"}`
	if after.Body != wantAfter {
		t.Errorf("after updateUser = %s, want %s", after.Body, wantAfter)
	}

	if res := mvc.Perform(web.Delete("/api/user/bob")); res.Status != http.StatusOK {
		t.Errorf("deleteUser = %d, want 200", res.Status)
	}
	if res := mvc.Perform(web.Get("/api/user/bob")); res.Status != http.StatusInternalServerError {
		t.Errorf("after deleteUser, status = %d, want %d", res.Status, http.StatusInternalServerError)
	}
	if res := mvc.Perform(web.Delete("/api/user/nobody")); res.Status != http.StatusNotFound {
		t.Errorf("deleting an absent user = %d, want %d", res.Status, http.StatusNotFound)
	}
}

// TestBulkUserCreationFails covers the defect the original carries:
// createUsersWithArrayInput and createUsersWithListInput bind their collection
// without @RequestBody, so the argument is null and the handler throws.
func TestBulkUserCreationFails(t *testing.T) {
	mvc := newTestMvc(t)
	for _, path := range []string{"/api/user/createWithArray", "/api/user/createWithList"} {
		res := mvc.Perform(web.Post(path).ContentType(web.MediaTypeJSON).
			Content(`[{"id":2,"username":"u2"}]`))
		if res.Status != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want %d", path, res.Status, http.StatusInternalServerError)
		}
	}
}

// TestLoginAndLogout covers the two session endpoints.
func TestLoginAndLogout(t *testing.T) {
	mvc := newTestMvc(t)
	res := mvc.Perform(web.Get("/api/user/login?username=a&password=b"))
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Status, http.StatusOK)
	}
	if res.Body != "logged in user session:1788414439628" {
		t.Errorf("body = %q, want the clock's value appended", res.Body)
	}
	if res.ContentType != web.MediaTypeJSON {
		t.Errorf("Content-Type = %q, want %q", res.ContentType, web.MediaTypeJSON)
	}
	for _, path := range []string{
		"/api/user/login?username=a", "/api/user/login?password=b", "/api/user/login",
	} {
		if bad := mvc.Perform(web.Get(path)); bad.Status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", path, bad.Status, http.StatusBadRequest)
		}
	}
	// logoutUser returns a body-less 200, so it carries no Content-Type.
	out := mvc.Perform(web.Get("/api/user/logout"))
	if out.Status != http.StatusOK || out.Body != "" || out.ContentType != "" {
		t.Errorf("logoutUser = %d %q %q, want 200 with no body and no Content-Type",
			out.Status, out.Body, out.ContentType)
	}
}

// TestUnmappedExceptionMessages covers the three Java exceptions the petstore
// raises. None of them is mapped to a status, which is why each surfaces as a
// 500; their messages name the Java type so a stack trace stays recognisable.
func TestUnmappedExceptionMessages(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{notFound(), "NotFoundException"},
		{errUnsupported{}, "UnsupportedOperationException"},
		{errNullPointer{}, "NullPointerException"},
	}
	for _, c := range cases {
		if got := c.err.Error(); got != c.want {
			t.Errorf("Error() = %q, want %q", got, c.want)
		}
		if _, ok := web.AsHTTPError(c.err); ok {
			t.Errorf("%s was classified as an HTTP error; it must become a 500", c.want)
		}
	}
}

// TestIdentifierIsTheRepositoryKey covers Identifiable.getIdentifier(), which
// is both what the repositories key on and a property Jackson adds to the
// response.
func TestIdentifierIsTheRepositoryKey(t *testing.T) {
	if got := (Pet{ID: 7}).Identifier(); got != 7 {
		t.Errorf("Pet.Identifier() = %d, want 7", got)
	}
	if got := (Order{ID: 5}).Identifier(); got != 5 {
		t.Errorf("Order.Identifier() = %d, want 5", got)
	}
	name := "bob"
	if got := (User{Username: &name}).Identifier(); got != "bob" {
		t.Errorf("User.Identifier() = %q, want %q", got, "bob")
	}
	// A user with no username keys on the empty string, which is what
	// getIdentifier() returning null degrades to as a map key.
	if got := (User{}).Identifier(); got != "" {
		t.Errorf("User.Identifier() with no username = %q, want the empty string", got)
	}
}
