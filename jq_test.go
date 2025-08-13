package jq_test

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/linkdata/jq"
)

type testType struct {
	S    string
	N    int
	IA   []int
	P    *testType
	Pnil *testType
	PA   []*testType
	SA   []testType
	M    map[string]any
	S_X  string `json:"sX"`
	IGN  int    `json:"-"`
}

var testInt = 1
var testString = "text"
var testIntArray = []int{1, 2, 3}
var testStringMatrix = [][]string{{"0.0", "0.1", "0.2"}, {"1.0", "1.1", "1.2"}, {"2.0", "2.1", "2.2"}}
var testStructVal = testType{
	S:  "string",
	N:  1,
	IA: testIntArray,
	P:  &testType{S: "p"},
	PA: []*testType{
		{S: "PA.0"},
		{S: "PA.1",
			PA: []*testType{
				{S: "PA.1.0"},
				{S: "PA.1.1"},
			},
		},
	},
	SA: []testType{
		{S: "SA.0"},
		{S: "SA.1",
			SA: []testType{
				{S: "SA.1.0"},
				{S: "SA.1.1"},
			},
		},
	},
	M: map[string]any{
		"MS": "string",
		"MI": 1,
		"MM": map[string]any{
			"MMS": "string2",
			"MMI": 2,
		},
	},
	S_X: "sX",
	IGN: 123,
}

func maybeError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Error(err)
	}
}

func TestGet_int(t *testing.T) {
	v, err := jq.Get(testInt, "")
	maybeError(t, err)
	if v != testInt {
		t.Error(v)
	}
}

func TestGet_string(t *testing.T) {
	v, err := jq.Get(testString, "")
	maybeError(t, err)
	if v != testString {
		t.Error(v)
	}
}

func TestGet_intArray(t *testing.T) {
	v, err := jq.Get(testIntArray, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testIntArray) {
		t.Error(v)
	}
}

func TestGet_intArrayIndex(t *testing.T) {
	for i := range testIntArray {
		v, err := jq.Get(testIntArray, strconv.Itoa(i))
		maybeError(t, err)
		if !reflect.DeepEqual(v, testIntArray[i]) {
			t.Error(v)
		}
	}
}

func TestGet_stringMatrix(t *testing.T) {
	v, err := jq.Get(testStringMatrix, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStringMatrix) {
		t.Error(v)
	}
}

func TestGet_stringMatrixIndex(t *testing.T) {
	for i, sl := range testStringMatrix {
		v, err := jq.Get(testStringMatrix, strconv.Itoa(i))
		maybeError(t, err)
		if !reflect.DeepEqual(v, sl) {
			t.Error(i, v)
		}
		for j, s := range sl {
			v, err := jq.Get(testStringMatrix, fmt.Sprintf("%d.%d", i, j))
			maybeError(t, err)
			if !reflect.DeepEqual(v, s) {
				t.Error(i, j, v)
			}
		}
	}
}

func TestGet_structVal(t *testing.T) {
	v, err := jq.Get(testStructVal, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal) {
		t.Error(v)
	}
}

func TestGet_structPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, &testStructVal) {
		t.Error(v)
	}
}

func TestGet_structValField(t *testing.T) {
	v, err := jq.Get(testStructVal, "S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S) {
		t.Error(v)
	}
}

func TestGet_structPtrField(t *testing.T) {
	v, err := jq.Get(&testStructVal, "S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S) {
		t.Error(v)
	}
}

func TestGet_structPtrPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "P")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.P) {
		t.Error(v)
	}
}

func TestGet_structValPtrString(t *testing.T) {
	v, err := jq.Get(testStructVal, "P.S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.P.S) {
		t.Error(v)
	}
}

func TestGet_structValFieldTag(t *testing.T) {
	v, err := jq.Get(testStructVal, "sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S_X) {
		t.Error(v)
	}
}

func TestGet_structValFieldTagIGN(t *testing.T) {
	_, err := jq.Get(testStructVal, "IGN")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
}

func TestGet_intArrayOutOfBounds(t *testing.T) {
	_, err := jq.Get(testIntArray, "4")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
	t.Log(err)
}

func TestGet_intArrayNotNumber(t *testing.T) {
	_, err := jq.Get(testIntArray, "foo")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Errorf("%q %T", err, err)
	}
}

func TestGet_structValPath(t *testing.T) {
	v, err := jq.Get(testStructVal, "SA.1.SA.0.sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.SA[1].SA[0].S_X) {
		t.Error(v)
	}
}

func TestGet_structPtrPath(t *testing.T) {
	v, err := jq.Get(&testStructVal, "PA.1.PA.0.sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.PA[1].PA[0].S_X) {
		t.Error(v)
	}
}

func TestSet_int(t *testing.T) {
	var x int
	err := jq.Set(&x, "", 2)
	maybeError(t, err)
	if x != 2 {
		t.Error(x)
	}
}

func TestSet_string(t *testing.T) {
	var x string
	err := jq.Set(&x, "", "foo")
	maybeError(t, err)
	if !reflect.DeepEqual(x, "foo") {
		t.Error(x)
	}
}

func TestSet_intArray(t *testing.T) {
	x := []int{1, 2, 3}
	err := jq.Set(&x, "1", 4)
	maybeError(t, err)
	if !reflect.DeepEqual(x, []int{1, 4, 3}) {
		t.Error(x)
	}
}

func TestSet_intArrayExpand(t *testing.T) {
	x := []int{1, 2, 3}
	err := jq.Set(&x, "3", 4)
	maybeError(t, err)
	if !reflect.DeepEqual(x, []int{1, 2, 3, 4}) {
		t.Error(x)
	}
}

func TestSet_structField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "sX", "foo!")
	maybeError(t, err)
	if x.S_X != "foo!" {
		t.Error(x)
	}
}

func TestSet_structValArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "SA.0.S", "foo!")
	maybeError(t, err)
	if x.SA[0].S != "foo!" {
		t.Error(x)
	}
}

func TestSet_structPrtArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "PA.1.S", "foo!")
	maybeError(t, err)
	if x.PA[1].S != "foo!" {
		t.Error(x)
	}
}

func TestGet_map(t *testing.T) {
	v, err := jq.Get(testStructVal, "M")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.M) {
		t.Error(v)
	}
}

func TestGet_mapInt(t *testing.T) {
	v, err := jq.Get(testStructVal, "M.MI")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.M["MI"]) {
		t.Error(v)
	}
}

func TestGet_mapMapString(t *testing.T) {
	v, err := jq.Get(testStructVal, "M.MM.MMS")
	maybeError(t, err)
	if !reflect.DeepEqual(v, "string2") {
		t.Error(v)
	}
}

func TestGet_mapMapStringNotFound(t *testing.T) {
	_, err := jq.Get(testStructVal, "M.MM.MMX")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Errorf("%q %T", err, err)
	}
}

func TestSet_mapInt(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "M.MI", 3)
	maybeError(t, err)
	if y := x.M["MI"]; !reflect.DeepEqual(y, 3) {
		t.Error(y)
	}
}

func TestSet_mapMapInt(t *testing.T) {
	x := testStructVal

	err := jq.Set(&x, "M.MM.MMI", 33)
	maybeError(t, err)

	got, err := jq.Get(x, "M.MM.MMI")
	maybeError(t, err)

	if !reflect.DeepEqual(33, got) {
		t.Error(got)
	}
}

func TestGetAs(t *testing.T) {
	x, err := jq.GetAs[int](testStructVal, "N")
	maybeError(t, err)
	if x != testStructVal.N {
		t.Error(x)
	}
}

func TestGetAsTypeMismatch(t *testing.T) {
	_, err := jq.GetAs[int](testStructVal, "S")
	if !errors.Is(err, jq.ErrTypeMismatch) {
		t.Fatal(err)
	}
	if x := err.Error(); x != "jq: expected int, not string" {
		t.Error(x)
	}
}

func TestSetTypeMismatch(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "N", "foo")
	if !errors.Is(err, jq.ErrTypeMismatch) {
		t.Fatal(err)
	}
	if x := err.Error(); x != "jq: expected int, not string" {
		t.Error(x)
	}
}
