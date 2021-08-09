package slice

import (
	"fmt"
	"reflect"
)

// Map maps the slice using the mapper function fn. Map panics if slice is not a
// slice, fn is not a function that accepts a single argument of the slice type
// or fn does not have a single return value of the slice type. The returned
// value is a slice with same type as the return value of fn.
//
// Examples
//
//	names := []string{"foo", "bar", "baz"}
//	uppercase := Map(names, strings.ToUpper).([]string)
//	// uppercase == []string{"FOO", "BAR", "BAZ"}
//
//	numbers := []int{1, 2, 4, 8}
//	doubled := Map(numbers, func(num int) int { return num*2 }).([]int)
//	// doubled == []int{2, 4, 8, 16}
func Map(slice interface{}, fn interface{}) interface{} {
	sliceType := reflect.TypeOf(slice)
	fnType := reflect.TypeOf(fn)

	if sliceType.Kind() != reflect.Slice {
		panic("slice.Map: not a slice")
	}

	if fnType.Kind() != reflect.Func {
		panic("slice.Map: not a function")
	}

	elemType := sliceType.Elem()

	if in := fnType.NumIn(); in != 1 {
		panic(fmt.Sprintf("slice.Map: function must have exactly 1 parameter; has %d", in))
	}

	if out := fnType.NumOut(); out != 1 {
		panic(fmt.Sprintf("slice.Map: function must have exactly 1 return value; has %d", out))
	}

	if in := fnType.In(0); in != elemType {
		panic("slice.Map: function parameter must be of same type as slice")
	}

	sliceVal := reflect.ValueOf(slice)
	fnVal := reflect.ValueOf(fn)

	outType := fnType.Out(0)
	out := reflect.MakeSlice(reflect.SliceOf(outType), 0, sliceVal.Cap())

	l := sliceVal.Len()
	for i := 0; i < l; i++ {
		val := sliceVal.Index(i)
		result := fnVal.Call([]reflect.Value{val})
		out = reflect.Append(out, result[0])
	}

	return out.Interface()
}
