package jq

import (
	"encoding/json"
	"reflect"
	"testing"
)

func differentialPair(left, right reflect.Type, pointerMask int) reflect.Type {
	fields := []reflect.StructField{
		{Name: "Left", Type: left, Anonymous: true},
		{Name: "Right", Type: right, Anonymous: true},
	}
	for i := range fields {
		if pointerMask&(1<<i) != 0 {
			fields[i].Type = reflect.PointerTo(fields[i].Type)
		}
	}
	return reflect.StructOf(fields)
}

func populateDifferential(value reflect.Value, next *int64) {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.Pointer:
			field.Set(reflect.New(field.Type().Elem()))
			populateDifferential(field.Elem(), next)
		case reflect.Struct:
			populateDifferential(field, next)
		case reflect.Int:
			*next = *next + 1
			field.SetInt(*next)
		}
	}
}

func TestResolveStructFields(t *testing.T) {
	type leaf struct {
		Value   int `json:"value"`
		Ignored int `json:"-"`
	}
	type middle struct {
		leaf
	}
	type nested struct {
		middle
		Direct string `json:"direct"`
	}
	type left struct {
		Value int
	}
	type right struct {
		Value int
	}
	type ambiguous struct {
		left
		right
	}

	tests := []struct {
		name string
		typ  reflect.Type
		want map[string][]int
	}{
		{
			name: "nested",
			typ:  reflect.TypeFor[nested](),
			want: map[string][]int{
				"value":  {0, 0, 0},
				"direct": {1},
			},
		},
		{
			name: "ambiguous",
			typ:  reflect.TypeFor[ambiguous](),
			want: map[string][]int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveStructFields(tc.typ); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("resolveStructFields() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveStructFieldsMatchesEncodingJSON(t *testing.T) {
	intType := reflect.TypeFor[int]()
	leafFields := []reflect.StructField{
		{Name: "Value", Type: intType},
		{Name: "Tagged", Type: intType, Tag: `json:"Value"`},
		{Name: "OtherTagged", Type: intType, Tag: `json:"Value"`},
		{Name: "Lower", Type: intType, Tag: `json:"value"`},
		{Name: "Ignored", Type: intType, Tag: `json:"-"`},
	}
	leaves := make([]reflect.Type, len(leafFields))
	for i, field := range leafFields {
		leaves[i] = reflect.StructOf([]reflect.StructField{field})
	}

	nodes := append([]reflect.Type(nil), leaves...)
	for i, left := range leaves {
		for j, right := range leaves {
			nodes = append(nodes, differentialPair(left, right, (i+2*j)&3))
		}
	}

	for i, left := range nodes {
		for j, right := range nodes {
			typ := differentialPair(left, right, (i+2*j)&3)
			value := reflect.New(typ).Elem()
			var next int64
			populateDifferential(value, &next)

			data, err := json.Marshal(value.Interface())
			if err != nil {
				t.Fatalf("case (%d, %d): json.Marshal: %v", i, j, err)
			}
			want := make(map[string]int)
			if err = json.Unmarshal(data, &want); err != nil {
				t.Fatalf("case (%d, %d): json.Unmarshal(%s): %v", i, j, data, err)
			}

			got := resolveStructFields(typ)
			if len(got) != len(want) {
				t.Fatalf("case (%d, %d): fields = %v, JSON = %s", i, j, got, data)
			}
			for name, index := range got {
				field, _ := structFieldValue(value, index)
				if expected, ok := want[name]; !ok || !field.IsValid() || field.Kind() != reflect.Int || int(field.Int()) != expected {
					t.Fatalf("case (%d, %d): field %q at %v = %v, JSON = %s", i, j, name, index, field, data)
				}
			}
		}
	}
}
