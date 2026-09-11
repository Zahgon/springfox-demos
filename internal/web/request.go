package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/springfox/springfox-demos/internal/javalang"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// HTTPError is an error that carries the status the dispatcher must report.
// It is the Go stand-in for the Spring exceptions the DispatcherServlet
// translates into a status (MissingServletRequestParameterException,
// HttpMediaTypeNotSupportedException, and friends). Any error that is *not* an
// HTTPError becomes a 500, exactly as an unhandled Java exception does.
type HTTPError struct {
	Status int
	Reason string
}

func (e *HTTPError) Error() string { return fmt.Sprintf("%d %s", e.Status, e.Reason) }

// Errors the framework raises on the caller's behalf.
var (
	ErrBadRequest    = &HTTPError{Status: http.StatusBadRequest, Reason: "Bad Request"}
	ErrNotFound      = &HTTPError{Status: http.StatusNotFound, Reason: "Not Found"}
	ErrUnsupported   = &HTTPError{Status: http.StatusUnsupportedMediaType, Reason: "Unsupported Media Type"}
	ErrNotAllowed    = &HTTPError{Status: http.StatusMethodNotAllowed, Reason: "Method Not Allowed"}
	ErrInternalError = &HTTPError{Status: http.StatusInternalServerError, Reason: "Internal Server Error"}
)

// Request is the Go form of the arguments Spring binds into a handler method.
type Request struct {
	Raw      *http.Request
	PathVars map[string]string
	// Path is the request path with the servlet context path already removed.
	Path string

	query url.Values
	body  []byte
	form  *multipart.Form
}

// Method returns the HTTP method.
func (r *Request) Method() string { return r.Raw.Method }

// PathVar returns a URI template variable, or "" when absent.
func (r *Request) PathVar(name string) string { return r.PathVars[name] }

// Query returns the first value of a query parameter and whether it was present.
func (r *Request) Query(name string) (string, bool) {
	v, ok := r.query[name]
	if !ok || len(v) == 0 {
		return "", false
	}
	return v[0], true
}

// QueryOr returns the first value of a query parameter or the given default.
func (r *Request) QueryOr(name, def string) string {
	if v, ok := r.Query(name); ok {
		return v
	}
	return def
}

// QueryAll returns every value of a repeated query parameter.
func (r *Request) QueryAll(name string) []string { return r.query[name] }

// QueryMap collects every query parameter into a map, keeping the first value
// of each — the binding Spring performs for a `@RequestParam Map<String,String>`.
func (r *Request) QueryMap() map[string]string {
	out := map[string]string{}
	for k, v := range r.query {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out
}

// RequireQuery returns a query parameter, or ErrBadRequest when it is absent —
// Spring's MissingServletRequestParameterException.
func (r *Request) RequireQuery(name string) (string, error) {
	v, ok := r.Query(name)
	if !ok {
		return "", ErrBadRequest
	}
	return v, nil
}

// QueryDouble converts a query parameter with java.lang.Double semantics,
// falling back to def when the parameter is absent. A value that does not parse
// yields ErrBadRequest, matching Spring's type-conversion failure.
func (r *Request) QueryDouble(name string, def float64) (float64, error) {
	raw, ok := r.Query(name)
	if !ok {
		return def, nil
	}
	return parseJavaDouble(raw)
}

func parseJavaDouble(raw string) (float64, error) {
	s := strings.TrimSpace(raw)
	switch s {
	case javalang.NaNString:
		return math.NaN(), nil
	case javalang.PositiveInfinityString, "+" + javalang.PositiveInfinityString:
		return math.Inf(1), nil
	case javalang.NegativeInfinityString:
		return math.Inf(-1), nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, ErrBadRequest
	}
	return v, nil
}

// Body returns the raw request body, reading it once.
func (r *Request) Body() ([]byte, error) {
	if r.body != nil {
		return r.body, nil
	}
	b, err := io.ReadAll(r.Raw.Body)
	if err != nil {
		return nil, err
	}
	r.body = b
	return b, nil
}

// BodyText returns the request body as a string.
func (r *Request) BodyText() (string, error) {
	b, err := r.Body()
	return string(b), err
}

// BindJSON deserialises the request body into v. A malformed body is a 400,
// which is what Spring's HttpMessageNotReadableException produces.
func (r *Request) BindJSON(v any) error {
	b, err := r.Body()
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return ErrBadRequest
	}
	if err := json.Unmarshal(b, v); err != nil {
		return ErrBadRequest
	}
	return nil
}

// ContentType returns the request's media type without parameters.
func (r *Request) ContentType() string {
	ct := r.Raw.Header.Get("Content-Type")
	if ct == "" {
		return ""
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
	}
	return mt
}

// Accepts reports whether the request's Accept header admits the media type.
// An absent or */* Accept header admits everything, which is how
// RequestPredicates.accept behaves.
func (r *Request) Accepts(mediaType string) bool {
	accept := r.Raw.Header.Get("Accept")
	if accept == "" {
		return true
	}
	want := strings.SplitN(mediaType, "/", 2)
	for _, part := range strings.Split(accept, ",") {
		entry := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if entry == "" || entry == "*/*" || entry == mediaType {
			return true
		}
		got := strings.SplitN(entry, "/", 2)
		if len(got) == 2 && len(want) == 2 && got[0] == want[0] && got[1] == "*" {
			return true
		}
	}
	return false
}

// AcceptsExplicitly reports whether the Accept header names the media type
// without relying on a wildcard.
func (r *Request) AcceptsExplicitly(mediaType string) bool {
	for _, part := range strings.Split(r.Raw.Header.Get("Accept"), ",") {
		if strings.TrimSpace(strings.SplitN(part, ";", 2)[0]) == mediaType {
			return true
		}
	}
	return false
}

// Multipart parses a multipart/form-data body.
func (r *Request) Multipart() (*multipart.Form, error) {
	if r.form != nil {
		return r.form, nil
	}
	if err := r.Raw.ParseMultipartForm(32 << 20); err != nil {
		return nil, ErrBadRequest
	}
	r.form = r.Raw.MultipartForm
	return r.form, nil
}

// RequirePart returns a named multipart form field.
func (r *Request) RequirePart(name string) (string, error) {
	form, err := r.Multipart()
	if err != nil {
		return "", err
	}
	v := form.Value[name]
	if len(v) == 0 {
		return "", ErrBadRequest
	}
	return v[0], nil
}

// RequireFilePart returns a named multipart file field.
func (r *Request) RequireFilePart(name string) (*multipart.FileHeader, error) {
	form, err := r.Multipart()
	if err != nil {
		return nil, err
	}
	fh := form.File[name]
	if len(fh) == 0 {
		return nil, ErrBadRequest
	}
	return fh[0], nil
}

// AsHTTPError unwraps err into the HTTPError the dispatcher should report,
// reporting false when the error is an ordinary failure and must become a 500.
func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}
