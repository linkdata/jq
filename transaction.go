package jq

import "reflect"

type undoLog struct {
	undos []func()
}

func cloneValue(value reflect.Value) (clone reflect.Value) {
	clone = reflect.New(value.Type()).Elem()
	clone.Set(value)
	return
}

func (log *undoLog) add(undo func()) {
	log.undos = append(log.undos, undo)
}

func (log *undoLog) set(dst, src reflect.Value) {
	old := cloneValue(dst)
	log.add(func() {
		dst.Set(old)
	})
	dst.Set(src)
}

func (log *undoLog) setMapIndex(value, key, elem reflect.Value) {
	old := value.MapIndex(key)
	log.add(func() {
		value.SetMapIndex(key, old)
	})
	value.SetMapIndex(key, elem)
}

func (log *undoLog) append(value, elem reflect.Value) {
	old := cloneValue(value)
	var oldTail, tail reflect.Value
	if value.Len() < value.Cap() {
		tail = value.Slice(0, value.Len()+1).Index(value.Len())
		oldTail = cloneValue(tail)
	}
	log.add(func() {
		if tail.IsValid() {
			tail.Set(oldTail)
		}
		value.Set(old)
	})
	value.Set(reflect.Append(value, elem))
}

func (log *undoLog) commit() {
	log.undos = nil
}

func (log *undoLog) rollback() {
	for i := len(log.undos) - 1; i >= 0; i-- {
		log.undos[i]()
	}
	log.undos = nil
}
