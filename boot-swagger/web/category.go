// Package web holds boot-swagger's own controllers, ported from
// springfoxdemo.boot.swagger.web.
package web

// Category is the enum springfoxdemo.boot.swagger.web.Category, with its three
// constants in declaration order.
type Category string

// The Category constants.
const (
	CategoryOne   Category = "ONE"
	CategoryTwo   Category = "TWO"
	CategoryThree Category = "THREE"
)

// CategoryValues is Category.values(), in declaration order.
var CategoryValues = []Category{CategoryOne, CategoryTwo, CategoryThree}

// CategoryNames is the constants' names, in declaration order.
var CategoryNames = []string{string(CategoryOne), string(CategoryTwo), string(CategoryThree)}

// ParseCategory is Enum.valueOf. It reports false for a value that names no
// constant, which Spring surfaces as a 400.
func ParseCategory(s string) (Category, bool) {
	for _, c := range CategoryValues {
		if string(c) == s {
			return c, true
		}
	}
	return "", false
}

// Name is Enum.name().
func (c Category) Name() string { return string(c) }
