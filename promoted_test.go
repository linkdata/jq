package jq_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/linkdata/jq"
)

type promotedEmbedded struct {
	Value int `json:"value"`
}

type promotedValue struct {
	promotedEmbedded
}

type promotedPointer struct {
	*promotedEmbedded
}

type promotedLeaf struct {
	Leaf int `json:"leaf"`
}

type promotedMiddle struct {
	promotedLeaf
}

type promotedDeep struct {
	promotedMiddle
}

type promotedUntagged struct {
	Bare int
}

type promotedUntaggedOuter struct {
	promotedUntagged
}

func jsonSnapshot(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPromotedFieldPaths(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		path     string
		snapshot string
		want     int
	}{
		{"tagged", &promotedValue{promotedEmbedded{Value: 1}}, "value", `{"value":1}`, 1},
		{"pointer", &promotedPointer{&promotedEmbedded{Value: 2}}, "value", `{"value":2}`, 2},
		{"deep", &promotedDeep{promotedMiddle{promotedLeaf{Leaf: 3}}}, "leaf", `{"leaf":3}`, 3},
		{"untagged", &promotedUntaggedOuter{promotedUntagged{Bare: 4}}, "Bare", `{"Bare":4}`, 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := jsonSnapshot(t, tc.value); got != tc.snapshot {
				t.Fatalf("json.Marshal = %s, want %s", got, tc.snapshot)
			}
			got, err := jq.Get(tc.value, tc.path)
			if err != nil || got != tc.want {
				t.Fatalf("Get(%q) = %#v, %v; want %d, nil", tc.path, got, err, tc.want)
			}

			changed, err := jq.Set(tc.value, tc.path, tc.want+10)
			if err != nil || !changed {
				t.Fatalf("Set(%q) = %t, %v; want true, nil", tc.path, changed, err)
			}
			got, err = jq.Get(tc.value, tc.path)
			if err != nil || got != tc.want+10 {
				t.Fatalf("Get(%q) after Set = %#v, %v; want %d, nil", tc.path, got, err, tc.want+10)
			}
			changed, err = jq.Set(tc.value, tc.path, tc.want+10)
			if err != nil || changed {
				t.Fatalf("no-op Set(%q) = %t, %v; want false, nil", tc.path, changed, err)
			}
		})
	}
}

func TestPromotedFieldMapAssignment(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		value := promotedValue{promotedEmbedded{Value: 1}}
		changed, err := jq.Set(&value, "", map[string]any{"value": 42})
		if err != nil || !changed || value.Value != 42 {
			t.Fatalf("Set = %t, %v; Value = %d; want true, nil, 42", changed, err, value.Value)
		}
		changed, err = jq.Set(&value, "", map[string]any{"value": 42})
		if err != nil || changed {
			t.Fatalf("no-op Set = %t, %v; want false, nil", changed, err)
		}
	})

	t.Run("pointer preserves identity", func(t *testing.T) {
		embedded := &promotedEmbedded{Value: 1}
		value := promotedPointer{promotedEmbedded: embedded}
		changed, err := jq.Set(&value, "", map[string]any{"value": 42})
		if err != nil || !changed {
			t.Fatalf("Set = %t, %v; want true, nil", changed, err)
		}
		if value.promotedEmbedded != embedded || embedded.Value != 42 {
			t.Fatalf("pointer/value = %p/%d; want %p/42", value.promotedEmbedded, embedded.Value, embedded)
		}
	})
}

func TestPromotedFieldSetChecked(t *testing.T) {
	t.Run("direct rejection", func(t *testing.T) {
		value := promotedValue{promotedEmbedded{Value: 1}}
		rejected := errors.New("rejected")
		changed, err := jq.SetChecked(&value, "value", 2, func() error {
			if value.Value != 2 {
				t.Fatalf("checker saw Value = %d, want 2", value.Value)
			}
			return rejected
		})
		if changed || err != rejected || value.Value != 1 {
			t.Fatalf("SetChecked = %t, %v; Value = %d; want false, rejection, 1", changed, err, value.Value)
		}
	})

	t.Run("map rejection preserves pointer", func(t *testing.T) {
		embedded := &promotedEmbedded{Value: 1}
		value := promotedPointer{promotedEmbedded: embedded}
		rejected := errors.New("rejected")
		changed, err := jq.SetChecked(&value, "", map[string]any{"value": 2}, func() error {
			if value.promotedEmbedded != embedded || embedded.Value != 2 {
				t.Fatalf("checker saw pointer/value = %p/%d, want %p/2", value.promotedEmbedded, embedded.Value, embedded)
			}
			return rejected
		})
		if changed || err != rejected {
			t.Fatalf("SetChecked = %t, %v; want false, rejection", changed, err)
		}
		if value.promotedEmbedded != embedded || embedded.Value != 1 {
			t.Fatalf("pointer/value = %p/%d, want %p/1", value.promotedEmbedded, embedded.Value, embedded)
		}
	})

	t.Run("nested map rejection preserves pointer", func(t *testing.T) {
		type child struct {
			*promotedEmbedded
		}
		type outer struct {
			Child child `json:"child"`
		}

		embedded := &promotedEmbedded{Value: 1}
		value := outer{Child: child{promotedEmbedded: embedded}}
		rejected := errors.New("rejected")
		input := map[string]any{"child": map[string]any{"value": 2}}
		changed, err := jq.SetChecked(&value, "", input, func() error {
			if value.Child.promotedEmbedded != embedded || embedded.Value != 2 {
				t.Fatalf("checker saw pointer/value = %p/%d, want %p/2", value.Child.promotedEmbedded, embedded.Value, embedded)
			}
			return rejected
		})
		if changed || err != rejected {
			t.Fatalf("SetChecked = %t, %v; want false, rejection", changed, err)
		}
		if value.Child.promotedEmbedded != embedded || embedded.Value != 1 {
			t.Fatalf("pointer/value = %p/%d, want %p/1", value.Child.promotedEmbedded, embedded.Value, embedded)
		}
	})

	t.Run("map assignment error preserves pointer", func(t *testing.T) {
		type embedded struct {
			Number int    `json:"number"`
			Text   string `json:"text"`
		}
		type outer struct {
			*embedded
		}

		for range 200 {
			inner := &embedded{Number: 1, Text: "original"}
			value := outer{embedded: inner}
			checked := false
			changed, err := jq.SetChecked(&value, "", map[string]any{"number": 2, "text": 3}, func() error {
				checked = true
				return nil
			})
			if checked {
				t.Fatal("checker called after an assignment error")
			}
			if changed || !errors.Is(err, jq.ErrTypeMismatch) {
				t.Fatalf("SetChecked = %t, %v; want false, ErrTypeMismatch", changed, err)
			}
			if value.embedded != inner || inner.Number != 1 || inner.Text != "original" {
				t.Fatalf("value = %#v, pointee = %#v; want original pointer and value", value, inner)
			}
		}
	})
}

func TestPromotedFieldMapAssignmentFailureIsAtomic(t *testing.T) {
	type embedded struct {
		Number int    `json:"number"`
		Text   string `json:"text"`
	}

	t.Run("promoted pointer", func(t *testing.T) {
		type outer struct {
			*embedded
		}

		for range 200 {
			inner := &embedded{Number: 1, Text: "original"}
			value := outer{embedded: inner}
			changed, err := jq.Set(&value, "", map[string]any{"number": 2, "text": 3})
			if changed || !errors.Is(err, jq.ErrTypeMismatch) {
				t.Fatalf("Set = %t, %v; want false, ErrTypeMismatch", changed, err)
			}
			if value.embedded != inner || inner.Number != 1 || inner.Text != "original" {
				t.Fatalf("value = %#v, pointee = %#v; want original pointer and value", value, inner)
			}
		}
	})

	t.Run("nested promoted pointer", func(t *testing.T) {
		type child struct {
			*embedded
		}
		type outer struct {
			Child child  `json:"child"`
			Text  string `json:"text"`
		}

		for range 200 {
			inner := &embedded{Number: 1}
			value := outer{Child: child{embedded: inner}, Text: "original"}
			input := map[string]any{
				"child": map[string]any{"number": 2},
				"text":  3,
			}
			changed, err := jq.Set(&value, "", input)
			if changed || !errors.Is(err, jq.ErrTypeMismatch) {
				t.Fatalf("Set = %t, %v; want false, ErrTypeMismatch", changed, err)
			}
			if value.Child.embedded != inner || inner.Number != 1 || value.Text != "original" {
				t.Fatalf("value = %#v, pointee = %#v; want original pointers and values", value, inner)
			}
		}
	})
}

func TestPromotedFieldDominance(t *testing.T) {
	t.Run("outer shadows promoted", func(t *testing.T) {
		type outer struct {
			promotedEmbedded
			Value string `json:"value"`
		}
		value := outer{promotedEmbedded: promotedEmbedded{Value: 1}, Value: "outer"}
		if got := jsonSnapshot(t, value); got != `{"value":"outer"}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(&value, "value")
		if err != nil || got != "outer" {
			t.Fatalf("Get = %#v, %v; want outer, nil", got, err)
		}
		changed, err := jq.Set(&value, "value", "updated")
		if err != nil || !changed || value.Value != "updated" || value.promotedEmbedded.Value != 1 {
			t.Fatalf("Set = %t, %v; value = %#v", changed, err, value)
		}
	})

	t.Run("shallower promoted field wins", func(t *testing.T) {
		type shallow struct {
			Leaf float64 `json:"leaf"`
		}
		type deepLeaf struct {
			Leaf int `json:"leaf"`
		}
		type middle struct {
			deepLeaf
		}
		type outer struct {
			shallow
			middle
		}
		value := outer{shallow: shallow{Leaf: 1.5}, middle: middle{deepLeaf{Leaf: 2}}}
		if got := jsonSnapshot(t, value); got != `{"leaf":1.5}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(&value, "leaf")
		if err != nil || got != 1.5 {
			t.Fatalf("Get = %#v, %v; want 1.5, nil", got, err)
		}
	})

	t.Run("tagged field wins equal depth", func(t *testing.T) {
		type untagged struct {
			Value int
		}
		type tagged struct {
			Other int `json:"Value"`
		}
		type outer struct {
			untagged
			tagged
		}
		value := outer{untagged: untagged{Value: 1}, tagged: tagged{Other: 2}}
		if got := jsonSnapshot(t, value); got != `{"Value":2}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(&value, "Value")
		if err != nil || got != 2 {
			t.Fatalf("Get = %#v, %v; want 2, nil", got, err)
		}
	})

	t.Run("tagged field resolves untagged ambiguity", func(t *testing.T) {
		type left struct {
			Value int
		}
		type right struct {
			Value int
		}
		type tagged struct {
			Other int `json:"Value"`
		}
		type outer struct {
			left
			right
			tagged
		}
		value := outer{left: left{Value: 1}, right: right{Value: 2}, tagged: tagged{Other: 3}}
		if got := jsonSnapshot(t, value); got != `{"Value":3}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(&value, "Value")
		if err != nil || got != 3 {
			t.Fatalf("Get = %#v, %v; want 3, nil", got, err)
		}
	})
}

func TestPromotedFieldAmbiguity(t *testing.T) {
	type PromotedLeft struct {
		Value int `json:"value"`
	}
	type PromotedRight struct {
		Value int `json:"value"`
	}
	typeOf := reflect.StructOf([]reflect.StructField{
		{Name: "PromotedLeft", Type: reflect.TypeFor[PromotedLeft](), Anonymous: true},
		{Name: "PromotedRight", Type: reflect.TypeFor[PromotedRight](), Anonymous: true},
	})
	value := reflect.New(typeOf)
	value.Elem().Field(0).Set(reflect.ValueOf(PromotedLeft{Value: 1}))
	value.Elem().Field(1).Set(reflect.ValueOf(PromotedRight{Value: 2}))
	obj := value.Interface()

	if got := jsonSnapshot(t, obj); got != `{}` {
		t.Fatalf("json.Marshal = %s, want {}", got)
	}
	_, err := jq.Get(obj, "value")
	requirePathNotFound(t, err)
	changed, err := jq.Set(obj, "value", 3)
	if changed {
		t.Fatal("Set reported a change for an ambiguous field")
	}
	requirePathNotFound(t, err)
	changed, err = jq.Set(obj, "", map[string]any{"value": 3})
	left := value.Elem().Field(0).Interface().(PromotedLeft)
	right := value.Elem().Field(1).Interface().(PromotedRight)
	if err != nil || changed || left.Value != 1 || right.Value != 2 {
		t.Fatalf("map Set = %t, %v; values = %#v, %#v; want unchanged", changed, err, left, right)
	}
}

func TestPromotedFieldSelectionDetails(t *testing.T) {
	t.Run("explicit anonymous name", func(t *testing.T) {
		type outer struct {
			promotedEmbedded `json:"inner"`
		}
		value := outer{promotedEmbedded{Value: 7}}
		if got := jsonSnapshot(t, value); got != `{"inner":{"value":7}}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		_, err := jq.Get(&value, "inner")
		if !errors.Is(err, jq.ErrPathNotFound) {
			t.Fatalf("Get(inner) error = %v, want ErrPathNotFound", err)
		}
		if got, want := err.Error(), `jq: "inner" not found in *jq_test.outer`; got != want {
			t.Fatalf("Get(inner) error = %q, want %q", got, want)
		}
		for _, path := range []string{"inner.", "inner.."} {
			_, err = jq.Get(&value, path)
			requirePathNotFound(t, err)
		}
		got, err := jq.Get(&value, "inner.value")
		if err != nil || got != 7 {
			t.Fatalf("Get(inner.value) = %#v, %v; want 7, nil", got, err)
		}
		_, err = jq.Get(&value, "value")
		requirePathNotFound(t, err)
	})

	t.Run("ignored promoted field", func(t *testing.T) {
		type ignored struct {
			Secret int `json:"-"`
		}
		type outer struct {
			ignored
		}
		value := outer{ignored{Secret: 7}}
		if got := jsonSnapshot(t, value); got != `{}` {
			t.Fatalf("json.Marshal = %s, want {}", got)
		}
		_, err := jq.Get(&value, "Secret")
		requirePathNotFound(t, err)
		changed, err := jq.Set(&value, "Secret", 8)
		if changed || value.Secret != 7 {
			t.Fatalf("Set = %t, %v; Secret = %d; want unchanged", changed, err, value.Secret)
		}
		requirePathNotFound(t, err)
	})

	t.Run("nil anonymous pointer", func(t *testing.T) {
		value := promotedPointer{}
		if got := jsonSnapshot(t, value); got != `{}` {
			t.Fatalf("json.Marshal = %s, want {}", got)
		}
		_, err := jq.Get(&value, "value")
		requirePathNotFound(t, err)
		changed, err := jq.Set(&value, "value", 1)
		if changed || value.promotedEmbedded != nil {
			t.Fatalf("Set = %t, %v; value = %#v; want unchanged", changed, err, value)
		}
		requirePathNotFound(t, err)
		changed, err = jq.Set(&value, "", map[string]any{"value": 1})
		if changed || value.promotedEmbedded != nil {
			t.Fatalf("map Set = %t, %v; value = %#v; want unchanged", changed, err, value)
		}
		requirePathNotFound(t, err)
	})

	t.Run("anonymous container is not a JSON field", func(t *testing.T) {
		value := promotedValue{promotedEmbedded{Value: 1}}
		_, err := jq.Get(&value, "promotedEmbedded.value")
		requirePathNotFound(t, err)
	})

	t.Run("exact case", func(t *testing.T) {
		value := promotedValue{promotedEmbedded{Value: 1}}
		_, err := jq.Get(&value, "Value")
		requirePathNotFound(t, err)
	})

	t.Run("dash with options is a name", func(t *testing.T) {
		typeOf := reflect.StructOf([]reflect.StructField{
			{Name: "Value", Type: reflect.TypeFor[int](), Tag: `json:"-,omitempty"`},
		})
		value := reflect.New(typeOf)
		value.Elem().Field(0).SetInt(1)
		if got := jsonSnapshot(t, value.Interface()); got != `{"-":1}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(value.Interface(), "-")
		if err != nil || got != 1 {
			t.Fatalf("Get = %#v, %v; want 1, nil", got, err)
		}
	})

	t.Run("invalid tag falls back to Go name", func(t *testing.T) {
		typeOf := reflect.StructOf([]reflect.StructField{
			{Name: "Value", Type: reflect.TypeFor[int](), Tag: `json:"bad\\name"`},
		})
		value := reflect.New(typeOf)
		value.Elem().Field(0).SetInt(1)
		if got := jsonSnapshot(t, value.Interface()); got != `{"Value":1}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(value.Interface(), "Value")
		if err != nil || got != 1 {
			t.Fatalf("Get = %#v, %v; want 1, nil", got, err)
		}
	})

	t.Run("named string map key", func(t *testing.T) {
		type key string
		value := promotedValue{promotedEmbedded{Value: 1}}
		changed, err := jq.Set(&value, "", map[key]any{"value": 2})
		if err != nil || !changed || value.Value != 2 {
			t.Fatalf("Set = %t, %v; Value = %d; want true, nil, 2", changed, err, value.Value)
		}
	})

	t.Run("non-string map key", func(t *testing.T) {
		value := promotedValue{promotedEmbedded{Value: 1}}
		changed, err := jq.Set(&value, "", map[int]any{1: 2})
		if err != nil || changed || value.Value != 1 {
			t.Fatalf("Set = %t, %v; Value = %d; want false, nil, 1", changed, err, value.Value)
		}
	})

	t.Run("unexported anonymous scalar", func(t *testing.T) {
		type hidden int
		type outer struct {
			hidden
		}
		value := outer{hidden: 1}
		if got := jsonSnapshot(t, value); got != `{}` {
			t.Fatalf("json.Marshal = %s, want {}", got)
		}
		_, err := jq.Get(&value, "hidden")
		requirePathNotFound(t, err)
	})

	t.Run("anonymous named pointer", func(t *testing.T) {
		type PromotedNamedPointerTarget struct {
			Nested int
		}
		type PromotedNamedPointer *PromotedNamedPointerTarget
		typeOf := reflect.StructOf([]reflect.StructField{
			{Name: "PromotedNamedPointer", Type: reflect.TypeFor[PromotedNamedPointer](), Anonymous: true},
		})
		target := &PromotedNamedPointerTarget{Nested: 1}
		pointer := PromotedNamedPointer(target)
		value := reflect.New(typeOf)
		value.Elem().Field(0).Set(reflect.ValueOf(pointer))
		obj := value.Interface()

		if got := jsonSnapshot(t, obj); got != `{"PromotedNamedPointer":{"Nested":1}}` {
			t.Fatalf("json.Marshal = %s", got)
		}
		got, err := jq.Get(obj, "PromotedNamedPointer.Nested")
		if err != nil || got != 1 {
			t.Fatalf("Get = %#v, %v; want 1, nil", got, err)
		}
		_, err = jq.Get(obj, "Nested")
		requirePathNotFound(t, err)
	})
}

func TestPromotedFieldRepeatedAndRecursiveTypes(t *testing.T) {
	t.Run("repeated type is ambiguous", func(t *testing.T) {
		type leaf struct {
			Value int `json:"value"`
		}
		type left struct {
			leaf
		}
		type right struct {
			leaf
		}
		type outer struct {
			left
			right
		}
		value := outer{left: left{leaf{Value: 1}}, right: right{leaf{Value: 2}}}
		if got := jsonSnapshot(t, value); got != `{}` {
			t.Fatalf("json.Marshal = %s, want {}", got)
		}
		_, err := jq.Get(&value, "value")
		requirePathNotFound(t, err)
	})

	t.Run("repeated type does not make a deeper field ambiguous", func(t *testing.T) {
		type leaf struct {
			Value int `json:"value"`
		}
		type shared struct {
			leaf
		}
		type left struct {
			shared
		}
		type right struct {
			shared
		}
		type outer struct {
			left
			right
		}
		value := outer{
			left:  left{shared{leaf{Value: 1}}},
			right: right{shared{leaf{Value: 2}}},
		}
		if got := jsonSnapshot(t, value); got != `{"value":1}` {
			t.Fatalf("json.Marshal = %s, want {\"value\":1}", got)
		}
		got, err := jq.Get(&value, "value")
		if err != nil || got != 1 {
			t.Fatalf("Get = %#v, %v; want 1, nil", got, err)
		}
	})

	t.Run("recursive embedding terminates", func(t *testing.T) {
		type recursive struct {
			*recursive
			Value int `json:"value"`
		}
		value := recursive{Value: 1}
		if value.recursive != nil {
			t.Fatal("recursive pointer is unexpectedly non-nil")
		}
		got, err := jq.Get(&value, "value")
		if err != nil || got != 1 {
			t.Fatalf("Get = %#v, %v; want 1, nil", got, err)
		}
	})
}
