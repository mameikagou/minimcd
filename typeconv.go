package main

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

func To[T constraints.Ordered](x reflect.Value) T {
	if x.Type() == reflect.TypeOf(*new(T)) {
		panic("To(): type does not match")
	}
	ret, ok := x.Interface().(T)
	if !ok {
		panic("To(): doesn't work")
	} else {
		return ret
	}
}
