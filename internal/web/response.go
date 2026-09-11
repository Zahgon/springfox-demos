package web

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Media types used across the demos, spelled exactly as the original emits them.
const (
	MediaTypeJSON      = "application/json"
	MediaTypeXML       = "application/xml"
	MediaTypeTextPlain = "text/plain"
	MediaTypeTextHTML  = "text/html"
	MediaTypeMultipart = "multipart/form-data"
	MediaTypeAll       = "*/*"

	// CharsetTextPlain is what Spring's StringHttpMessageConverter writes when
	// a handler returns a String and the route declares no `produces`.
	CharsetTextPlain = "text/plain;charset=UTF-8"
	// CharsetTextPlainLatin1 is what Spring MVC writes when the request carried
	// no Content-Type to negotiate a charset from.
	CharsetTextPlainLatin1 = "text/plain;charset=ISO-8859-1"
	// CharsetJSON is what Spring MVC writes for a JSON body produced by a
	// Spring Integration gateway.
	CharsetJSON = "application/json;charset=UTF-8"
)

// bodyKind distinguishes the three shapes Spring's message converters produce.
type bodyKind int

const (
	bodyNone bodyKind = iota // ResponseEntity with no body: no Content-Type at all
	bodyText                 // a String body: text/plain unless `produces` says otherwise
	bodyJSON                 // an object body: application/json
)

// ResponseEntity is the Go form of org.springframework.http.ResponseEntity.
type ResponseEntity struct {
	Status  int
	kind    bodyKind
	text    string
	value   any
	Headers http.Header
	// contentType, when set, wins over both `produces` and the body kind.
	contentType string
}

// OK returns 200 with a JSON-serialised body.
func OK(v any) ResponseEntity { return ResponseEntity{Status: http.StatusOK, kind: bodyJSON, value: v} }

// OKText returns 200 with a string body.
func OKText(s string) ResponseEntity {
	return ResponseEntity{Status: http.StatusOK, kind: bodyText, text: s}
}

// OKEmpty returns 200 with no body, like new ResponseEntity(HttpStatus.OK).
func OKEmpty() ResponseEntity { return ResponseEntity{Status: http.StatusOK, kind: bodyNone} }

// Status returns the given status with no body.
func Status(code int) ResponseEntity { return ResponseEntity{Status: code, kind: bodyNone} }

// WithContentType pins the Content-Type, overriding the route's `produces`.
func (r ResponseEntity) WithContentType(ct string) ResponseEntity {
	r.contentType = ct
	return r
}

// render resolves the entity into wire bytes plus a Content-Type, applying the
// route's `produces` list the way Spring's content negotiation does: the first
// producible media type wins for every body kind, and a body-less entity gets
// no Content-Type at all.
func (r ResponseEntity) render(produces []string, charset string) (body []byte, contentType string, err error) {
	// "*/*" is how Springfox spells "this mapping declares no produces"; it is
	// not a concrete media type and never reaches the wire.
	produces = concreteMediaTypes(produces)
	switch r.kind {
	case bodyNone:
		return nil, "", nil
	case bodyText:
		ct := r.contentType
		if ct == "" {
			if len(produces) > 0 {
				ct = produces[0]
			} else {
				ct = MediaTypeTextPlain + ";charset=" + charset
			}
		}
		return []byte(r.text), ct, nil
	default:
		ct := r.contentType
		if ct == "" {
			if len(produces) > 0 {
				ct = produces[0]
			} else {
				ct = MediaTypeJSON
			}
		}
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(r.value); err != nil {
			return nil, "", err
		}
		// Jackson does not append a newline; encoding/json does.
		return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), ct, nil
	}
}

// concreteMediaTypes drops the wildcard entries from a produces list.
func concreteMediaTypes(produces []string) []string {
	out := make([]string, 0, len(produces))
	for _, mt := range produces {
		if mt != MediaTypeAll {
			out = append(out, mt)
		}
	}
	return out
}

// MarshalJSON serialises a value the way the application's message converters
// do: no HTML escaping and no trailing newline.
func MarshalJSON(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
