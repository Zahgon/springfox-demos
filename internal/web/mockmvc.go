package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// MockMvc drives an App in-process without binding a socket, replacing
// org.springframework.test.web.servlet.MockMvc.
type MockMvc struct {
	app     *App
	handler http.Handler
}

// NewMockMvc wraps an application, like
// MockMvcBuilders.webAppContextSetup(context).build().
func NewMockMvc(app *App) *MockMvc {
	return &MockMvc{app: app, handler: app.Handler()}
}

// MockResult is the outcome of a simulated request.
type MockResult struct {
	Status      int
	Headers     http.Header
	Body        string
	ContentType string
}

// RequestBuilder accumulates the parts of a simulated request.
type RequestBuilder struct {
	method  string
	url     string
	headers http.Header
	body    string
}

// Get starts a GET request builder.
func Get(url string) *RequestBuilder { return newBuilder(http.MethodGet, url) }

// Post starts a POST request builder.
func Post(url string) *RequestBuilder { return newBuilder(http.MethodPost, url) }

// Put starts a PUT request builder.
func Put(url string) *RequestBuilder { return newBuilder(http.MethodPut, url) }

// Delete starts a DELETE request builder.
func Delete(url string) *RequestBuilder { return newBuilder(http.MethodDelete, url) }

// RequestFor starts a builder for an arbitrary method.
func RequestFor(method, url string) *RequestBuilder { return newBuilder(method, url) }

func newBuilder(method, url string) *RequestBuilder {
	return &RequestBuilder{method: method, url: url, headers: http.Header{}}
}

// ContentType sets the request's Content-Type.
func (b *RequestBuilder) ContentType(ct string) *RequestBuilder {
	b.headers.Set("Content-Type", ct)
	return b
}

// Accept sets the request's Accept header.
func (b *RequestBuilder) Accept(a string) *RequestBuilder {
	b.headers.Set("Accept", a)
	return b
}

// Header sets an arbitrary request header.
func (b *RequestBuilder) Header(name, value string) *RequestBuilder {
	b.headers.Set(name, value)
	return b
}

// Content sets the request body.
func (b *RequestBuilder) Content(body string) *RequestBuilder {
	b.body = body
	return b
}

// Perform executes the request against the application.
func (m *MockMvc) Perform(b *RequestBuilder) *MockResult {
	var body io.Reader
	if b.body != "" {
		body = strings.NewReader(b.body)
	}
	req := httptest.NewRequest(b.method, b.url, body)
	for k, vs := range b.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	m.handler.ServeHTTP(rec, req)
	res := rec.Result()
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	return &MockResult{
		Status:      res.StatusCode,
		Headers:     res.Header,
		Body:        string(raw),
		ContentType: res.Header.Get("Content-Type"),
	}
}
