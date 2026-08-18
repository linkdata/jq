package jq

import (
	"errors"
	"reflect"
	"strings"
)

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

func assignMap(from, into reflect.Value, log *undoLog) (changed bool, err error) {
	tp := into.Type()
	if from.Type().Key().Kind() != reflect.String {
		return
	}
	// A staged struct still shares data through pointers. Use a local log when
	// the caller has no transaction, so failed staging can restore those values.
	var local undoLog
	ownsLog := log == nil
	if ownsLog {
		log = &local
	}
	fields := cachedStructFields(tp)
	iter := from.MapRange()
	for iter.Next() {
		name := iter.Key().String()
		index, ok := fields[name]
		if !ok {
			continue
		}
		value := iter.Value()
		field, throughPointer := structFieldValue(into, index)
		if !field.IsValid() || !field.CanSet() {
			err = errPathNotFound{name, tp.String()}
			break
		}
		if value.Kind() == reflect.Interface {
			if value.IsNil() {
				value = reflect.Zero(field.Type())
			} else {
				value = value.Elem()
			}
		}
		// Unwrapping an interface can expose a pointer.
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				value = reflect.Zero(field.Type())
			} else if field.Kind() != reflect.Pointer {
				value = value.Elem()
			}
		}
		var candidate reflect.Value
		var change bool
		if candidate, change, err = stageValue(value, field, log); err != nil {
			break
		}
		if change {
			if throughPointer {
				log.set(field, candidate)
			} else {
				field.Set(candidate)
			}
			changed = true
		}
	}
	if err == nil {
		return
	}
	changed = false
	if ownsLog {
		log.rollback()
	}
	return
}

// stageNewElement returns a candidate for a new slice element.
func stageNewElement(from, into reflect.Value) (candidate reflect.Value, err error) {
	// Keep this dispatch separate from stageValue. The caller supplies a zero
	// destination that cannot be observed before append, so change detection is
	// unnecessary.
	if err = assignable(from, into); err == nil {
		candidate = from
		return
	}
	if from.Kind() == reflect.Map && into.Kind() == reflect.Struct {
		candidate = cloneValue(into)
		_, err = assignMap(from, candidate, nil)
		return
	}
	if isNumber(from.Kind()) && isNumber(into.Kind()) {
		if from.Type().ConvertibleTo(into.Type()) {
			err = nil
			candidate = from.Convert(into.Type())
		}
	}
	return
}

// stageValue prepares a replacement for into and reports a change only on success.
func stageValue(from, into reflect.Value, log *undoLog) (candidate reflect.Value, changed bool, err error) {
	// Map candidates are shallow, so assignMap logs writes through shared
	// pointers until the update succeeds.
	if err = assignable(from, into); err == nil {
		candidate = from
		changed = !reflect.DeepEqual(into.Interface(), from.Interface())
		return
	}
	if from.Kind() == reflect.Map && into.Kind() == reflect.Struct {
		candidate = cloneValue(into)
		changed, err = assignMap(from, candidate, log)
		return
	}
	if isNumber(from.Kind()) && isNumber(into.Kind()) {
		if from.Type().ConvertibleTo(into.Type()) {
			err = nil
			candidate = from.Convert(into.Type())
			changed = !into.Equal(candidate)
		}
	}
	return
}

func assign(from, into reflect.Value, log *undoLog) (changed bool, err error) {
	var candidate reflect.Value
	if candidate, changed, err = stageValue(from, into, log); err == nil && changed {
		if log == nil {
			into.Set(candidate)
		} else {
			log.set(into, candidate)
		}
	}
	return
}

type assignment struct {
	value reflect.Value
	log   *undoLog
}

type pointerIdentity struct {
	typeOf  reflect.Type // the pointer type is part of the traversal state
	pointer any          // a boxed unsafe.Pointer remains comparable and visible to the GC
}

// pointerCycleDetector uses Brent's algorithm for transparent pointer traversal.
// Each pointer has one successor, so one moving anchor can detect a cycle.
type pointerCycleDetector struct {
	anchor   pointerIdentity
	power    int
	distance int
}

func (d *pointerCycleDetector) visit(v reflect.Value) (cycle bool) {
	visit := pointerIdentity{typeOf: v.Type(), pointer: v.UnsafePointer()}
	if d.power == 0 {
		d.anchor = visit
		d.power = 1
		return
	}
	if visit == d.anchor {
		cycle = true
		return
	}
	d.distance++
	if d.distance == d.power {
		d.anchor = visit
		d.power *= 2
		d.distance = 0
	}
	return
}

func getSet(obj reflect.Value, jspath string, setting *assignment) (v reflect.Value, changed bool, err error) {
	v = obj
	if !v.IsValid() {
		err = errors.Join(err, errPathNotFound{jspath, "<nil>"})
		return
	}
	set := setting != nil
	jspath = strings.TrimLeft(jspath, ".")
	elem, tail, _ := strings.Cut(jspath, ".")
	if elem == "" {
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
	var cycleDetector pointerCycleDetector
	var followedPointer bool
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
			return
		}
		if v.Kind() == reflect.Pointer {
			// A cycle repeats, so omitting the first pointer still detects it while
			// avoiding detector work for ordinary one-pointer traversal.
			if followedPointer {
				if cycleDetector.visit(v) {
					err = errors.Join(err, errPathNotFound{jspath, v.Type().String()})
					return
				}
			}
			followedPointer = true
			v = v.Elem()
			continue
		}
		concrete := v.Elem()
		if set && concrete.Kind() == reflect.Struct && !concrete.CanSet() {
			err = errors.Join(err, errPathNotFound{jspath, concrete.Type().String()})
			return
		}
		v = concrete
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		if idx, ok := parseArrayIndex(elem); ok {
			if set && v.Kind() == reflect.Slice && idx == v.Len() {
				if !v.CanSet() {
					err = errPathNotFound{jspath, v.Type().String()}
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
					if candidate, err = stageNewElement(value, zero); err != nil {
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
			if idx < v.Len() {
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
						var candidate reflect.Value
						if candidate, changed, err = stageValue(value, mapped, setting.log); err == nil {
							if changed {
								if setting.log == nil {
									v.SetMapIndex(key, candidate)
								} else {
									setting.log.setMapIndex(v, key, candidate)
								}
								v = candidate
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
	case reflect.Struct:
		if index, ok := cachedStructFields(v.Type())[elem]; ok {
			if field, _ := structFieldValue(v, index); field.IsValid() {
				// CanInterface is how Get determines whether it can return a field. A dots-only
				// tail also ends here; break reaches the shared error without hiding deeper errors.
				if set && !field.CanInterface() && strings.Trim(tail, ".") == "" {
					break
				}
				return getSet(field, tail, setting)
			}
		}
	}
	err = errPathNotFound{elem, v.Type().String()}
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
// Struct components exactly match names selected by encoding/json's default
// struct-field rules, including JSON tag names and unambiguous promoted fields.
// An untagged anonymous struct field contributes its promoted fields directly to
// the containing struct's namespace; it does not add its Go type name as a path
// component. A valid explicit JSON name on the field replaces that promotion
// with a component named by the tag.
//
// Exported fields promoted through unexported embedded structs are readable with
// Get and writable with [Set], including through map-to-struct assignments. An
// exact `json:"-"` tag excludes a field from the path namespace. Get and [Set]
// can traverse an explicitly named unexported embedded field on paths to
// reachable exported fields, but a path ending at the embedded field returns an
// error matching [ErrPathNotFound]. Traversing a nil pointer or an unresolved
// pointer/interface cycle returns the same error.
//
// When traversal reaches an array or slice, a component is a valid index only
// if it is "0" or begins with an ASCII digit from '1' through '9' followed by
// zero or more ASCII decimal digits. The index must be at most 4294967294 and
// representable as int; otherwise the error matches [ErrPathNotFound].
func Get(obj any, jspath string) (val any, err error) {
	rv := reflect.ValueOf(obj)
	if rv, _, err = getSet(rv, jspath, nil); err == nil {
		if rv.CanInterface() {
			val = rv.Interface()
		} else {
			err = errPathNotFound{jspath, reflect.TypeOf(obj).String()}
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
// obj must be a non-nil pointer. An empty path targets the pointed-to value.
// Array and slice components follow the index syntax documented by [Get]. A
// path into a settable slice may append one element by using an index equal to
// the slice's current length. Map paths address existing string-keyed entries
// and do not create new entries. A nil val stores the destination type's zero
// value.
//
// Set converts among integer kinds other than uintptr and floating-point kinds
// using [Go numeric conversion rules]. These conversions can truncate, wrap, or
// lose precision and do not report overflow.
//
// Struct components and string keys in map-to-struct assignments follow [Get]'s
// field-selection rules. Only entries with matching string keys update fields;
// all other entries are ignored. For an existing struct, unselected fields are
// retained and Set reports no write if no selected field changes; an appended
// struct starts from zero. Existing overlays are shallow: preserved pointers
// retain identity, and successful updates to promoted fields reached through
// embedded pointers are visible through other aliases.
//
// Set can traverse an explicitly named unexported embedded field to update a
// reachable exported field. A path ending at the embedded field, or a
// map-to-struct key selecting it, returns an error matching [ErrPathNotFound]. Set
// does not allocate nil pointers. Traversing a nil pointer or an unresolved
// pointer/interface cycle returns the same error.
//
// For an existing destination value, Set skips an assignable replacement when
// [reflect.DeepEqual] reports that the current and replacement values are equal.
// This is Go deep equality, not equality of serialized JSON; Set does not marshal
// values to make this decision. Set can therefore report no write and retain an
// existing pointer, map, or slice, including its aliasing, when a distinct
// replacement is deeply equal.
//
// Set leaves obj unchanged when it returns an error. It does not synchronize
// access to obj; callers must prevent concurrent reads and writes.
//
// [Go numeric conversion rules]: https://go.dev/ref/spec#Conversions_between_numeric_types
func Set(obj any, jspath string, val any) (changed bool, err error) {
	return set(obj, jspath, val, nil)
}

// SetChecked updates jspath only when check accepts the tentative result.
//
// It performs the same update as [Set], including updates to reachable exported
// fields through explicitly named unexported embedded fields. If the update
// reports a write, SetChecked calls check exactly once while obj contains the
// tentative result. A nil error commits the update. If check returns an error,
// SetChecked restores obj, returns false, and returns the error unchanged. If
// check panics, SetChecked restores obj before the panic continues. A nil check
// behaves like [Set].
//
// check is not called if the update is invalid or reports no write, including
// when [Set] skips a deeply equal assignable replacement. When check uses [Get]
// to inspect obj, a path ending at an explicitly named unexported embedded field
// returns an error matching [ErrPathNotFound], even after a tentative update
// beneath it; longer paths to reachable exported fields remain readable.
//
// check must not mutate obj or values reachable from it, call [Set] or SetChecked
// on them, or retain references into a rejected value. Rollback restores only
// changes made by SetChecked.
//
// SetChecked does not synchronize access to obj. Callers must prevent concurrent
// access for the entire call, including while check runs.
func SetChecked(obj any, jspath string, val any, check func() error) (changed bool, err error) {
	if check == nil {
		return Set(obj, jspath, val)
	}

	var log undoLog
	defer log.rollback()
	if changed, err = set(obj, jspath, val, &log); err == nil && changed {
		if err = check(); err == nil {
			log.commit()
			return
		}
	}
	changed = false
	return
}
