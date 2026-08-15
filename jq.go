package jq

import (
	"errors"
	"reflect"
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

func assignMap(from, into reflect.Value) (changed bool, err error) {
	tp := into.Type()
	iter := from.MapRange()
	for iter.Next() {
		if iter.Key().Kind() == reflect.String {
			keystring := iter.Key().String()
			for i := range tp.NumField() {
				if sf := tp.Field(i); sf.IsExported() && matchField(sf, keystring) {
					var change bool
					field := into.Field(i)
					value := iter.Value()
					switch value.Kind() {
					case reflect.Interface:
						if value.IsNil() {
							value = reflect.Zero(field.Type())
						} else {
							value = value.Elem()
						}
					case reflect.Pointer:
						if field.Kind() == reflect.Pointer {
							if value.IsNil() {
								value = reflect.Zero(field.Type())
							}
						} else {
							if value.IsNil() {
								value = reflect.Zero(field.Type())
							} else {
								value = value.Elem()
							}
						}
					}
					if change, err = assign(value, field, nil); err != nil {
						return
					}
					changed = changed || change
				}
			}
		}
	}
	return
}

// prepareAssignment returns a candidate value assignable to into's type
// without writing to into.
//
// It intentionally mirrors the dispatch in [assign] rather than sharing code:
// assign needs per-case change detection, while this must stay free of
// comparisons and extra allocations for the slice-append paths.
func prepareAssignment(from, into reflect.Value) (candidate reflect.Value, err error) {
	if err = assignable(from, into); err == nil {
		candidate = from
		return
	}
	if from.Kind() == reflect.Map && into.Kind() == reflect.Struct {
		candidate = cloneValue(into)
		_, err = assignMap(from, candidate)
	} else if isNumber(from.Kind()) && isNumber(into.Kind()) {
		if from.Type().ConvertibleTo(into.Type()) {
			err = nil
			candidate = from.Convert(into.Type())
		}
	}
	return
}

func assign(from, into reflect.Value, log *undoLog) (changed bool, err error) {
	if err = assignable(from, into); err == nil {
		if changed = !reflect.DeepEqual(into.Interface(), from.Interface()); changed {
			if log == nil {
				into.Set(from)
			} else {
				log.set(into, from)
			}
		}
		return
	}
	if from.Kind() == reflect.Map && into.Kind() == reflect.Struct {
		candidate := cloneValue(into)
		if changed, err = assignMap(from, candidate); err == nil {
			if changed {
				if log == nil {
					into.Set(candidate)
				} else {
					log.set(into, candidate)
				}
			}
		} else {
			changed = false
		}
	} else if isNumber(from.Kind()) && isNumber(into.Kind()) {
		if from.Type().ConvertibleTo(into.Type()) {
			err = nil
			converted := from.Convert(into.Type())
			if changed = !into.Equal(converted); changed {
				if log == nil {
					into.Set(converted)
				} else {
					log.set(into, converted)
				}
			}
		}
	}
	return
}

type assignment struct {
	value reflect.Value
	log   *undoLog
}

func getSet(obj reflect.Value, jspath string, setting *assignment) (v reflect.Value, changed bool, err error) {
	v = obj
	if !v.IsValid() {
		err = errors.Join(err, errPathNotFound{jspath, "<nil>"})
		return
	}
	set := setting != nil
	elem, tail, hasDot := strings.Cut(jspath, ".")
	if elem == "" {
		if hasDot {
			return getSet(v, tail, setting)
		}
		if set {
			if !v.CanSet() {
				if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && !v.IsNil() {
					v = v.Elem()
				}
			}
			if !v.CanSet() {
				err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
				return
			}
			value := setting.value
			if !value.IsValid() {
				value = reflect.Zero(v.Type())
			}
			changed, err = assign(value, v, setting.log)
		}
		return
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		if idx, ok := parseArrayIndex(elem); ok {
			if set && v.Kind() == reflect.Slice && idx == v.Len() {
				if !v.CanSet() {
					err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
					return
				}
				// Allow expanding slices by one each time. Prepare the element before
				// changing the real slice so an invalid assignment leaves it untouched.
				var candidate reflect.Value
				if tail == "" {
					zero := reflect.Zero(v.Type().Elem())
					value := setting.value
					if !value.IsValid() {
						value = zero
					}
					if candidate, err = prepareAssignment(value, zero); err != nil {
						return
					}
				} else {
					candidate = reflect.New(v.Type().Elem()).Elem()
					candidateSetting := assignment{value: setting.value}
					if _, _, err = getSet(candidate, tail, &candidateSetting); err != nil {
						return
					}
				}
				if setting.log == nil {
					if idx >= v.Cap() {
						v.Grow(1)
					}
					v.SetLen(idx + 1)
					v.Index(idx).Set(candidate)
				} else {
					setting.log.append(v, candidate)
				}
				v = v.Index(idx)
				changed = true
				return
			}
			if idx >= 0 && idx < v.Len() {
				return getSet(v.Index(idx), tail, setting)
			}
		}
	case reflect.Map:
		keyType := v.Type().Key()
		if keyType.Kind() == reflect.String {
			key := reflect.ValueOf(elem)
			if key.Type() != keyType {
				key = key.Convert(keyType)
			}
			mapped := v.MapIndex(key)
			if mapped.IsValid() {
				if tail == "" {
					if set {
						value := setting.value
						if !value.IsValid() {
							value = reflect.Zero(mapped.Type())
						}
						if err = assignable(value, mapped); err == nil {
							var change bool
							if change = !reflect.DeepEqual(mapped.Interface(), value.Interface()); change {
								if setting.log == nil {
									v.SetMapIndex(key, value)
								} else {
									setting.log.setMapIndex(v, key, value)
								}
								v = value
								changed = true
							} else {
								v = mapped
							}
						}
					} else {
						v = mapped
					}
					return
				}
				if !set {
					return getSet(mapped, tail, setting)
				}
				// Map values are not settable, so recurse via a writable copy.
				value := reflect.New(mapped.Type()).Elem()
				value.Set(mapped)
				if _, changed, err = getSet(value, tail, setting); err != nil {
					return
				}
				if changed {
					if setting.log == nil {
						v.SetMapIndex(key, value)
					} else {
						setting.log.setMapIndex(v, key, value)
					}
				}
				v = value
				return
			}
		}
	case reflect.Pointer:
		if v.IsNil() {
			err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
			return
		}
		return getSet(v.Elem(), jspath, setting)
	case reflect.Interface:
		if v.IsNil() {
			err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
			return
		}
		concrete := v.Elem()
		if set && jspath != "" && concrete.Kind() == reflect.Struct && !concrete.CanSet() {
			err = errors.Join(err, errPathNotFound{jspath, concrete.Type().String()})
			return
		}
		return getSet(concrete, jspath, setting)
	case reflect.Struct:
		tp := v.Type()
		for i := 0; i < tp.NumField(); i++ {
			if sf := tp.Field(i); sf.IsExported() && matchField(sf, elem) {
				f := v.Field(i)
				return getSet(f, tail, setting)
			}
		}
	}
	err = errors.Join(err, errPathNotFound{elem, v.Type().String()})
	return
}

// GetAs returns the value at jspath in obj as T.
//
// It returns [ErrTypeMismatch] when the resolved value is not assignable to T.
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

// Get returns the value at jspath in obj.
//
// An empty path returns obj itself unless obj is nil. Values containing maps,
// slices, or pointers may share their backing data with obj.
//
// When traversal reaches an array or slice, a component is a valid index only
// if it is "0" or begins with an ASCII digit from '1' through '9' followed by
// zero or more ASCII decimal digits. The index must be at most 4294967294 and
// representable as int; otherwise the error matches [ErrPathNotFound].
func Get(obj any, jspath string) (val any, err error) {
	rv := reflect.ValueOf(obj)
	if rv, _, err = getSet(rv, jspath, nil); err == nil {
		err = ErrPathNotFound
		if rv.CanInterface() {
			val = rv.Interface()
			err = nil
		}
	}
	return
}

func set(obj any, jspath string, val any, log *undoLog) (changed bool, err error) {
	err = ErrInvalidReceiver
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		setting := assignment{value: reflect.ValueOf(val), log: log}
		_, changed, err = getSet(rv, jspath, &setting)
	}
	return
}

// Set updates jspath in obj and reports whether it performed a write.
//
// obj must be a non-nil pointer. An empty path replaces the pointed-to value.
// Array and slice components follow the index syntax documented by [Get]. A
// path into a settable slice may append one element by using an index equal to
// the slice's current length. Map paths address existing string-keyed entries
// and do not create new entries. A nil val stores the destination type's zero
// value.
//
// Set leaves obj unchanged when it returns an error. It does not synchronize
// access to obj; callers must prevent concurrent reads and writes.
func Set(obj any, jspath string, val any) (changed bool, err error) {
	return set(obj, jspath, val, nil)
}

func setChecked(obj any, jspath string, val any, check func() error) (changed bool, err error) {
	var log undoLog
	committed := false
	defer func() {
		if !committed {
			log.rollback()
			changed = false
		}
	}()

	if changed, err = set(obj, jspath, val, &log); err == nil && changed {
		if err = check(); err == nil {
			log.commit()
			committed = true
		}
	}
	return
}

// SetChecked updates jspath only when check accepts the resulting object.
//
// SetChecked first applies the same operation as [Set]. If Set reports a write,
// SetChecked calls check exactly once while obj contains the tentative result. A
// nil error commits the change. If check returns an error, SetChecked restores
// obj, returns false, and returns that error unchanged. If check panics,
// SetChecked restores obj before the panic continues. A nil check behaves like
// [Set].
//
// check is not called for an invalid operation or when Set reports no write. It
// may inspect obj, including by calling [Get], but must not mutate obj or values
// reachable from it, call [Set] or SetChecked on them, or retain references into
// a rejected tentative value. Rollback covers only mutations made by SetChecked
// itself.
//
// SetChecked does not synchronize access to obj. Callers must prevent concurrent
// access for the entire call, including while check runs.
func SetChecked(obj any, jspath string, val any, check func() error) (changed bool, err error) {
	if check != nil {
		return setChecked(obj, jspath, val, check)
	}
	return set(obj, jspath, val, nil)
}
