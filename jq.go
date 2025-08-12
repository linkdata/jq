package jq

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

func matchField(f reflect.StructField, want string) (yes bool) {
	name := f.Name
	if tag, ok := f.Tag.Lookup("json"); ok {
		if tag, _, _ = strings.Cut(tag, ","); tag != "" {
			if tag == "-" {
				return false
			}
			name = tag
		}
	}
	return strings.EqualFold(name, want)
}

func getRef(obj reflect.Value, jspath string) (v reflect.Value, mapkey reflect.Value, err error) {
	v = obj
	elem, tail, _ := strings.Cut(jspath, ".")
	if elem != "" {
		switch v.Kind() {
		case reflect.Array, reflect.Slice:
			var idx int
			if idx, err = strconv.Atoi(elem); err == nil {
				if idx >= 0 && idx < v.Len() {
					return getRef(v.Index(idx), tail)
				}
			}
		case reflect.Struct:
			tp := v.Type()
			for i := 0; i < tp.NumField(); i++ {
				if matchField(tp.Field(i), elem) {
					f := v.Field(i)
					return getRef(f, tail)
				}
			}
		case reflect.Map:
			iter := v.MapRange()
			for iter.Next() {
				if iter.Key().String() == elem {
					if tail == "" {
						mapkey = iter.Key()
						return
					}
					return getRef(iter.Value(), tail)
				}
			}
		case reflect.Pointer, reflect.Interface:
			if !(v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr()) {
				v = v.Elem()
			}
			return getRef(v, jspath)
		}
		err = errors.Join(err, errPathNotFound{elem, v.Type().String()})
	}
	return
}

func Get(obj any, jspath string) (val any, err error) {
	var mk reflect.Value
	rv := reflect.ValueOf(obj)
	if rv, mk, err = getRef(rv, jspath); err == nil {
		if mk.IsValid() {
			rv = rv.MapIndex(mk)
		}
		val = rv.Interface()
	}
	return
}

func Set(obj any, jspath string, val any) (err error) {
	var mk reflect.Value
	rv := reflect.ValueOf(obj)
	if rv, mk, err = getRef(rv, jspath); err == nil {
		if mk.IsValid() {
			rv.SetMapIndex(mk, reflect.ValueOf(val))
		} else {
			if !rv.CanAddr() {
				rv = rv.Elem()
			}
			rv.Set(reflect.ValueOf(val))
		}
	}
	return
}
