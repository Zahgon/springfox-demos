package springfox

import "testing"

// TestRegexIsAFullMatch covers PathSelectors.regex, which Springfox implements
// with String.matches — the pattern is anchored to the whole path, so a
// substring match is not enough.
func TestRegexIsAFullMatch(t *testing.T) {
	sel := Regex(".*/api/pet.*")
	cases := []struct {
		path string
		want bool
	}{
		{"/api/pet", true},
		{"/springfox/api/pet/{petId}", true},
		{"/api/petstore/anything", true},
		{"/api/store/order", false},
		{"/pet", false},
	}
	for _, c := range cases {
		if got := sel(c.path); got != c.want {
			t.Errorf("regex(.*/api/pet.*)(%q) = %v, want %v", c.path, got, c.want)
		}
	}

	// An unanchored pattern must not match a longer path.
	exact := Regex("/api/pet")
	if exact("/springfox/api/pet") {
		t.Error("regex(/api/pet) matched /springfox/api/pet, but it should be a full match")
	}
}

// TestPetstorePathsComposition covers boot-swagger's petstorePaths(), which
// composes three regexes with Predicate.or.
func TestPetstorePathsComposition(t *testing.T) {
	sel := Regex(".*/api/pet.*").
		Or(Regex(".*/api/user.*").
			Or(Regex(".*/api/store.*")))
	for _, path := range []string{
		"/springfox/api/pet", "/springfox/api/user/login", "/springfox/api/store/order",
	} {
		if !sel(path) {
			t.Errorf("petstorePaths did not select %q", path)
		}
	}
	for _, path := range []string{"/springfox/category/map", "/springfox/upload"} {
		if sel(path) {
			t.Errorf("petstorePaths selected %q", path)
		}
	}
}

// TestCategoryPathsComposition covers boot-swagger's categoryPaths().
func TestCategoryPathsComposition(t *testing.T) {
	sel := Regex(".*/category.*").
		Or(Regex(".*/category").
			Or(Regex(".*/categories")))
	for _, path := range []string{
		"/springfox/category/Resource", "/springfox/category", "/springfox/categories",
		"/springfox/category/{id}/map",
	} {
		if !sel(path) {
			t.Errorf("categoryPaths did not select %q", path)
		}
	}
	if sel("/springfox/api/pet") {
		t.Error("categoryPaths selected /springfox/api/pet")
	}
}

// TestAnt covers PathSelectors.ant, whose patterns are Ant paths rather than
// regular expressions: "." is a literal character, "*" spans one path segment
// and "**" spans any number.
func TestAnt(t *testing.T) {
	segments := Ant("/api/*/order")
	if !segments("/api/store/order") {
		t.Error("a single-star pattern did not match one segment")
	}
	if segments("/api/store/deep/order") {
		t.Error("a single-star pattern matched across a segment boundary")
	}
	if !Ant("/api/**")("/api/a/b/c") {
		t.Error("a double-star pattern did not match across segments")
	}
	if !Ant("/api/pet/*")("/api/pet/7") {
		t.Error("a trailing single-star pattern did not match")
	}
}

// TestBootSwaggerSecurityContextPathSelectorMatchesNothing pins a real quirk of
// the original: boot-swagger's securityContext bean is restricted with
//
//	.forPaths(ant(".*/api/pet.*"))
//
// which is a regular expression handed to an Ant matcher. Read as an Ant
// pattern it requires the path to begin with a literal "." and to contain a
// literal ".", so it matches none of the application's paths and the context is
// never applied. The pet operations carry a security block anyway, from their
// own @Authorization annotations.
func TestBootSwaggerSecurityContextPathSelectorMatchesNothing(t *testing.T) {
	sel := Ant(".*/api/pet.*")
	for _, path := range []string{
		"/springfox/api/pet", "/springfox/api/pet/{petId}", "/api/pet", "/springfox/api/user/login",
	} {
		if sel(path) {
			t.Errorf("ant(\".*/api/pet.*\") selected %q, but the pattern requires literal dots", path)
		}
	}
	// The equivalent Ant pattern would match.
	if !Ant("/**/api/pet/**")("/springfox/api/pet/7") {
		t.Error("the equivalent Ant pattern did not match")
	}
}

// TestContains covers boot-swagger's user-api selector, a plain substring test
// rather than a PathSelectors helper.
func TestContains(t *testing.T) {
	sel := Contains("user")
	for _, path := range []string{
		"/springfox/api/user/login", "/springfox/category/{id}/{userId}",
	} {
		if !sel(path) {
			t.Errorf("contains(user) did not select %q", path)
		}
	}
	if sel("/springfox/api/pet") {
		t.Error("contains(user) selected /springfox/api/pet")
	}
}

// TestAnyAndNone covers the two trivial selectors.
func TestAnyAndNone(t *testing.T) {
	if !Any()("/anything") {
		t.Error("any() rejected a path")
	}
	if None()("/anything") {
		t.Error("none() accepted a path")
	}
	if Regex(".*").Negate()("/anything") {
		t.Error("negate did not invert the selector")
	}
	if !Regex("/a").And(Regex(".*"))("/a") {
		t.Error("and did not combine the selectors")
	}
}
