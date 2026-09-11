package springfox

import (
	"net/http"
	"sort"

	"github.com/springfox/springfox-demos/internal/api"
)

// defaultResponses are the response messages Springfox adds to every operation
// unless a Docket sets useDefaultResponseMessages(false). The set depends on
// the HTTP method.
func defaultResponses(method string) []api.Response {
	ok := api.Response{Code: http.StatusOK, Description: "OK"}
	created := api.Response{Code: http.StatusCreated, Description: "Created"}
	noContent := api.Response{Code: http.StatusNoContent, Description: "No Content"}
	unauthorized := api.Response{Code: http.StatusUnauthorized, Description: "Unauthorized"}
	forbidden := api.Response{Code: http.StatusForbidden, Description: "Forbidden"}
	notFound := api.Response{Code: http.StatusNotFound, Description: "Not Found"}

	switch method {
	case http.MethodGet:
		return []api.Response{ok, unauthorized, forbidden, notFound}
	case http.MethodPost, http.MethodPut:
		return []api.Response{ok, created, unauthorized, forbidden, notFound}
	default:
		return []api.Response{ok, noContent, unauthorized, forbidden}
	}
}

// mergeResponses overlays an operation's declared @ApiResponse entries on the
// default response messages for its HTTP method: a declared code replaces the
// default with the same code, and any other declared code is added. The result
// is ordered by status code, which is the order Springfox emits.
func mergeResponses(method string, declared []api.Response) []api.Response {
	merged := map[int]api.Response{}
	for _, r := range defaultResponses(method) {
		merged[r.Code] = r
	}
	for _, r := range declared {
		if existing, ok := merged[r.Code]; ok && r.Description == "" {
			r.Description = existing.Description
		}
		merged[r.Code] = r
	}
	codes := make([]int, 0, len(merged))
	for c := range merged {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	out := make([]api.Response, 0, len(codes))
	for _, c := range codes {
		out = append(out, merged[c])
	}
	return out
}
