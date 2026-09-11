package springfox

import (
	"bytes"
	"encoding/json"
)

// object is a JSON object that keeps its insertion order. Swagger documents are
// compared byte for byte by tooling and by this project's differential tests,
// and Springfox emits a specific key order that a Go map cannot preserve.
type object struct {
	keys []string
	vals map[string]any
}

func obj() *object { return &object{vals: map[string]any{}} }

// set appends or replaces a key, keeping first-insertion order.
func (o *object) set(key string, value any) *object {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = value
	return o
}

// setIf appends the key only when cond holds.
func (o *object) setIf(cond bool, key string, value any) *object {
	if cond {
		o.set(key, value)
	}
	return o
}

// setNonEmpty appends a string key only when the value is not empty.
func (o *object) setNonEmpty(key, value string) *object {
	return o.setIf(value != "", key, value)
}

func (o *object) empty() bool { return len(o.keys) == 0 }

// MarshalJSON writes the object in insertion order.
func (o *object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		val, err := marshal(o.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// marshal serialises a value without HTML escaping, which Jackson also omits.
func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// Marshal renders a generated document to its wire bytes.
func Marshal(v any) ([]byte, error) { return marshal(v) }
