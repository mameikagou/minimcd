package main

import (
	"reflect"

	"golang.org/x/exp/constraints"
)

func To[T constraints.Ordered](x reflect.Value) T {
	ret, ok := x.Interface().(T)
	if !ok {
		panic("To(): doesn't work")
	} else {
		return ret
	}
}
