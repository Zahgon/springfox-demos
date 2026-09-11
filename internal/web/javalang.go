package web

import "github.com/springfox/springfox-demos/internal/javalang"

// JavaDoubleToString reproduces java.lang.Double.toString for handlers that
// concatenate a boxed Double into a response body.
func JavaDoubleToString(d float64) string { return javalang.DoubleToString(d) }

// JavaLongToString reproduces java.lang.Long.toString.
func JavaLongToString(v int64) string { return javalang.LongToString(v) }
