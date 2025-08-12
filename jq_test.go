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
	S_X: "sX",
	IGN: 123,
}

func maybeError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Error(err)
	}
}

func TestGetSimple_int(t *testing.T) {
	v, err := jq.Get(testInt, "")
	maybeError(t, err)
	if v != testInt {
		t.Error(v)
	}
}

func TestGetSimple_string(t *testing.T) {
	v, err := jq.Get(testString, "")
	maybeError(t, err)
	if v != testString {
		t.Error(v)
	}
}

func TestGetSimple_intArray(t *testing.T) {
	v, err := jq.Get(testIntArray, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testIntArray) {
		t.Error(v)
	}
}

func TestGetSimple_intArrayIndex(t *testing.T) {
	for i := range testIntArray {
		v, err := jq.Get(testIntArray, strconv.Itoa(i))
		maybeError(t, err)
		if !reflect.DeepEqual(v, testIntArray[i]) {
			t.Error(v)
		}
	}
}

func TestGetSimple_stringMatrix(t *testing.T) {
	v, err := jq.Get(testStringMatrix, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStringMatrix) {
		t.Error(v)
	}
}

func TestGetSimple_stringMatrixIndex(t *testing.T) {
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

func TestGetSimple_structVal(t *testing.T) {
	v, err := jq.Get(testStructVal, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal) {
		t.Error(v)
	}
}

func TestGetSimple_structPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "")
	maybeError(t, err)
	if !reflect.DeepEqual(v, &testStructVal) {
		t.Error(v)
	}
}

func TestGetSimple_structValField(t *testing.T) {
	v, err := jq.Get(testStructVal, "S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S) {
		t.Error(v)
	}
}

func TestGetSimple_structPtrField(t *testing.T) {
	v, err := jq.Get(&testStructVal, "S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S) {
		t.Error(v)
	}
}

func TestGetSimple_structPtrPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "P")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.P) {
		t.Error(v)
	}
}

func TestGetSimple_structValPtrString(t *testing.T) {
	v, err := jq.Get(testStructVal, "P.S")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.P.S) {
		t.Error(v)
	}
}

func TestGetSimple_structValFieldTag(t *testing.T) {
	v, err := jq.Get(testStructVal, "sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.S_X) {
		t.Error(v)
	}
}

func TestGetSimple_structValFieldTagIGN(t *testing.T) {
	_, err := jq.Get(testStructVal, "IGN")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
	t.Log(err)
}

func TestGetSimple_intArrayOutOfBounds(t *testing.T) {
	_, err := jq.Get(testIntArray, "4")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
	t.Log(err)
}

func TestGetSimple_intArrayNotNumber(t *testing.T) {
	_, err := jq.Get(testIntArray, "foo")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Errorf("%q %T", err, err)
	}
	t.Log(err)
}

func TestGetSimple_structValPath(t *testing.T) {
	v, err := jq.Get(testStructVal, "sa.1.sa.0.sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.SA[1].SA[0].S_X) {
		t.Error(v)
	}
}

func TestGetSimple_structPtrPath(t *testing.T) {
	v, err := jq.Get(&testStructVal, "pa.1.pa.0.sX")
	maybeError(t, err)
	if !reflect.DeepEqual(v, testStructVal.PA[1].PA[0].S_X) {
		t.Error(v)
	}
}

func TestSetSimple_int(t *testing.T) {
	var x int
	err := jq.Set(&x, "", 2)
	maybeError(t, err)
	if x != 2 {
		t.Error(x)
	}
}

func TestSetSimple_string(t *testing.T) {
	var x string
	err := jq.Set(&x, "", "foo")
	maybeError(t, err)
	if !reflect.DeepEqual(x, "foo") {
		t.Error(x)
	}
}

func TestSetSimple_intArray(t *testing.T) {
	x := []int{1, 2, 3}

	/*e := json.Unmarshal([]byte("[1,2,3]"), &x)
	maybeError(t, e)*/
	err := jq.Set(&x, "1", 4)
	maybeError(t, err)
	if !reflect.DeepEqual(x, []int{1, 4, 3}) {
		t.Error(x)
	}
}

func TestSetSimple_structField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "sX", "foo!")
	maybeError(t, err)
	if x.S_X != "foo!" {
		t.Error(x)
	}
}

func TestSetSimple_structValArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "SA.0.S", "foo!")
	maybeError(t, err)
	if x.SA[0].S != "foo!" {
		t.Error(x)
	}
}

func TestSetSimple_structPrtArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "PA.1.S", "foo!")
	maybeError(t, err)
	if x.PA[1].S != "foo!" {
		t.Error(x)
	}
}
