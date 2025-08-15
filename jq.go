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
	return name == want
}

func assignable(from, into reflect.Value) (err error) {
	if !from.Type().AssignableTo(into.Type()) {
		err = errTypeMismatch{into.Type(), from.Type()}
	}
	return
}

func isNumber(k reflect.Kind) bool {
	switch k {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func assign(from, into reflect.Value) (changed bool, err error) {
	if err = assignable(from, into); err == nil {
		if changed = !reflect.DeepEqual(into.Interface(), from.Interface()); changed {
			into.Set(from)
		}
		return
	}
	if from.Kind() == reflect.Map && into.Kind() == reflect.Struct {
		err = nil
		tp := into.Type()
		iter := from.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String {
				keystring := iter.Key().String()
				for i := range tp.NumField() {
					if matchField(tp.Field(i), keystring) {
						var change bool
						if change, err = assign(iter.Value().Elem(), into.Field(i)); err != nil {
							return
						}
						changed = changed || change
					}
				}
			}
		}
	}
	if isNumber(from.Kind()) && isNumber(into.Kind()) {
		if from.Type().ConvertibleTo(into.Type()) {
			err = nil
			converted := from.Convert(into.Type())
			if changed = !into.Equal(converted); changed {
				into.Set(converted)
			}
		}
	}
	return
}

func getSet(obj reflect.Value, jspath string, setting reflect.Value) (v reflect.Value, changed bool, err error) {
	v = obj
	elem, tail, _ := strings.Cut(jspath, ".")
	if elem == "" {
		if setting.IsValid() {
			if !v.CanAddr() {
				v = v.Elem()
			}
			changed, err = assign(setting, v)
		}
		return
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		var idx int
		if idx, err = strconv.Atoi(elem); err == nil {
			if setting.IsValid() && v.Kind() == reflect.Slice && idx == v.Len() {
				// allow expanding slices by one each time
				if idx >= v.Cap() {
					v.Grow(1)
				}
				v.SetLen(idx + 1)
			}
			if idx >= 0 && idx < v.Len() {
				return getSet(v.Index(idx), tail, setting)
			}
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if iter.Key().String() == elem {
				if tail == "" {
					if setting.IsValid() {
						if err = assignable(setting, iter.Value()); err == nil {
							var change bool
							if change = !reflect.DeepEqual(v.MapIndex(iter.Key()).Interface(), setting.Interface()); change {
								v.SetMapIndex(iter.Key(), setting)
								v = setting
								changed = true
							}
						}
					} else {
						v = v.MapIndex(iter.Key())
					}
					return
				}
				return getSet(iter.Value(), tail, setting)
			}
		}
	case reflect.Interface, reflect.Pointer:
		if !(v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr()) {
			v = v.Elem()
		}
		return getSet(v, jspath, setting)
	case reflect.Struct:
		tp := v.Type()
		for i := 0; i < tp.NumField(); i++ {
			if matchField(tp.Field(i), elem) {
				f := v.Field(i)
				return getSet(f, tail, setting)
			}
		}
	}
	err = errors.Join(err, errPathNotFound{elem, v.Type().String()})
	return
}

func GetAs[T any](obj any, jspath string) (val T, err error) {
	var x any
	if x, err = Get(obj, jspath); err == nil {
		var ok bool
		if val, ok = x.(T); !ok {
			err = errTypeMismatch{reflect.TypeOf(val), reflect.TypeOf(x)}
		}
	}
	return
}

func Get(obj any, jspath string) (val any, err error) {
	rv := reflect.ValueOf(obj)
	if rv, _, err = getSet(rv, jspath, reflect.Value{}); err == nil {
		err = ErrPathNotFound
		if rv.CanInterface() {
			val = rv.Interface()
			err = nil
		}
	}
	return
}

func Set(obj any, jspath string, val any) (changed bool, err error) {
	err = ErrInvalidReceiver
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		_, changed, err = getSet(rv, jspath, reflect.ValueOf(val))
	}
	return
}
