package jq_test

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/linkdata/jq"
)

type testSubType struct {
	S string
}

type testType struct {
	S     string
	I     int
	AI    []int
	T     testSubType
	PT    *testType
	PTnil *testType
	APT   []*testType
	AT    []testType
	M     map[string]any
	S_X   string `json:"sX"`
	IGN   int    `json:"-"`
}

var testInt = 1
var testString = "text"
var testIntArray = []int{1, 2, 3}
var testStringMatrix = [][]string{{"0.0", "0.1", "0.2"}, {"1.0", "1.1", "1.2"}, {"2.0", "2.1", "2.2"}}
var testStructVal = testType{
	S:  "string",
	I:  1,
	AI: testIntArray,
	T:  testSubType{S: "T_S"},
	PT: &testType{S: "PT_S"},
	APT: []*testType{
		{S: "PA.0"},
		{S: "PA.1",
			APT: []*testType{
				{S: "PA.1.0"},
				{S: "PA.1.1"},
			},
		},
	},
	AT: []testType{
		{S: "SA.0"},
		{S: "SA.1",
			AT: []testType{
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

func mustEqual(t *testing.T, a, b any) {
	t.Helper()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("not equal\n got %T %#v\nwant %T %#v\n", a, a, b, b)
	}
}

func TestGet_int(t *testing.T) {
	v, err := jq.Get(testInt, "")
	maybeError(t, err)
	mustEqual(t, v, testInt)
}

func TestGet_string(t *testing.T) {
	v, err := jq.Get(testString, "")
	maybeError(t, err)
	mustEqual(t, v, testString)
}

func TestGet_intArray(t *testing.T) {
	v, err := jq.Get(testIntArray, "")
	maybeError(t, err)
	mustEqual(t, v, testIntArray)
}

func TestGet_intArrayIndex(t *testing.T) {
	for i := range testIntArray {
		v, err := jq.Get(testIntArray, strconv.Itoa(i))
		maybeError(t, err)
		mustEqual(t, v, testIntArray[i])
	}
}

func TestGet_stringMatrix(t *testing.T) {
	v, err := jq.Get(testStringMatrix, "")
	maybeError(t, err)
	mustEqual(t, v, testStringMatrix)
}

func TestGet_stringMatrixIndex(t *testing.T) {
	for i, sl := range testStringMatrix {
		v, err := jq.Get(testStringMatrix, strconv.Itoa(i))
		maybeError(t, err)
		mustEqual(t, v, sl)
		for j, s := range sl {
			v, err := jq.Get(&testStringMatrix, fmt.Sprintf("%d.%d", i, j))
			maybeError(t, err)
			mustEqual(t, v, s)
		}
	}
}

func TestGet_rootStructVal(t *testing.T) {
	v, err := jq.Get(testStructVal, "")
	maybeError(t, err)
	mustEqual(t, v, testStructVal)
}

func TestGet_fieldStructVal(t *testing.T) {
	v, err := jq.Get(testStructVal, "T")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.T)
}

func TestGet_rootStructPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "")
	maybeError(t, err)
	mustEqual(t, v, &testStructVal)
}

func TestGet_fieldStructPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "T")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.T)
}

func TestGet_structValField(t *testing.T) {
	v, err := jq.Get(&testStructVal, "S")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.S)
}

func TestGet_structPtrField(t *testing.T) {
	v, err := jq.Get(&testStructVal, "S")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.S)
}

func TestGet_structPtrPtr(t *testing.T) {
	v, err := jq.Get(&testStructVal, "PT")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.PT)
}

func TestGet_structValPtrString(t *testing.T) {
	v, err := jq.Get(&testStructVal, "PT.S")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.PT.S)
}

func TestGet_structValFieldTag(t *testing.T) {
	v, err := jq.Get(&testStructVal, "sX")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.S_X)
}

func TestGet_structValFieldTagIGN(t *testing.T) {
	_, err := jq.Get(&testStructVal, "IGN")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
	if err != nil {
		mustEqual(t, err.Error(), "jq: \"IGN\" not found in jq_test.testType")
	}
}

func TestGet_intArrayOutOfBounds(t *testing.T) {
	_, err := jq.Get(&testIntArray, "4")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Error(err)
	}
}

func TestGet_intArrayNotNumber(t *testing.T) {
	_, err := jq.Get(&testIntArray, "foo")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Errorf("%q %T", err, err)
	}
}

func TestGet_structValPath(t *testing.T) {
	v, err := jq.Get(&testStructVal, "AT.1.AT.0.sX")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.AT[1].AT[0].S_X)
}

func TestGet_structPtrPath(t *testing.T) {
	v, err := jq.Get(&testStructVal, "APT.1.APT.0.sX")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.APT[1].APT[0].S_X)
}

func TestSet_int(t *testing.T) {
	var x int
	err := jq.Set(&x, "", 2)
	maybeError(t, err)
	mustEqual(t, x, 2)
}

func TestSet_string(t *testing.T) {
	var x string
	err := jq.Set(&x, "", "foo")
	maybeError(t, err)
	mustEqual(t, x, "foo")
}

func TestSet_intArray(t *testing.T) {
	x := []int{1, 2, 3}
	err := jq.Set(&x, "1", 4)
	maybeError(t, err)
	mustEqual(t, x, []int{1, 4, 3})
}

func TestSet_intArrayExpand(t *testing.T) {
	x := []int{1, 2, 3}
	err := jq.Set(&x, "3", 4)
	maybeError(t, err)
	mustEqual(t, x, []int{1, 2, 3, 4})
}

func TestSet_structField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "sX", "foo!")
	maybeError(t, err)
	mustEqual(t, x.S_X, "foo!")
}

func TestSet_structValArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "AT.0.S", "foo!")
	maybeError(t, err)
	mustEqual(t, x.AT[0].S, "foo!")
}

func TestSet_structPrtArrayField(t *testing.T) {
	var x testType = testStructVal
	err := jq.Set(&x, "APT.1.S", "foo!")
	maybeError(t, err)
	mustEqual(t, x.APT[1].S, "foo!")
}

func TestGet_map(t *testing.T) {
	v, err := jq.Get(&testStructVal, "M")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.M)
}

func TestGet_mapInt(t *testing.T) {
	v, err := jq.Get(&testStructVal, "M.MI")
	maybeError(t, err)
	mustEqual(t, v, testStructVal.M["MI"])
}

func TestGet_mapMapString(t *testing.T) {
	v, err := jq.Get(&testStructVal, "M.MM.MMS")
	maybeError(t, err)
	if !reflect.DeepEqual(v, "string2") {
		t.Error(v)
	}
}

func TestGet_mapMapStringNotFound(t *testing.T) {
	_, err := jq.Get(&testStructVal, "M.MM.MMX")
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Errorf("%q %T", err, err)
	}
}

func TestSet_mapInt(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "M.MI", 3)
	maybeError(t, err)
	mustEqual(t, x.M["MI"], 3)
}

func TestSet_mapMapInt(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "M.MM.MMI", 33)
	maybeError(t, err)
	got, err := jq.Get(x, "M.MM.MMI")
	maybeError(t, err)
	mustEqual(t, got, 33)
}

func TestGetAs(t *testing.T) {
	x, err := jq.GetAs[int](testStructVal, "I")
	maybeError(t, err)
	if x != testStructVal.I {
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
	err := jq.Set(&x, "I", "foo")
	if !errors.Is(err, jq.ErrTypeMismatch) {
		t.Fatal(err)
	}
	if x := err.Error(); x != "jq: expected int, not string" {
		t.Error(x)
	}
}

func TestSetAcceptsGet(t *testing.T) {
	x := testStructVal
	y, err := jq.Get(x, "")
	maybeError(t, err)
	err = jq.Set(&x, "", y)
	maybeError(t, err)
	mustEqual(t, x, y)
}

func TestSetRootStructAcceptsMap(t *testing.T) {
	var x testSubType
	err := jq.Set(&x, "", map[string]any{"S": "foo"})
	maybeError(t, err)
	mustEqual(t, x, testSubType{S: "foo"})
}

func TestSetFieldStructAcceptsMap(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "T", map[string]any{"S": "foo!"})
	maybeError(t, err)
	mustEqual(t, x.T, testSubType{S: "foo!"})
}

func TestSetRootStructAcceptsNestedMap(t *testing.T) {
	x := testStructVal
	err := jq.Set(&x, "", map[string]any{"T": map[string]any{"S": "foo!"}})
	maybeError(t, err)
	mustEqual(t, x.T, testSubType{S: "foo!"})
}

func TestSetRootStructMapWrongType(t *testing.T) {
	var x testSubType
	err := jq.Set(&x, "", map[string]any{"S": 1})
	if !errors.Is(err, jq.ErrTypeMismatch) {
		t.Error(err)
	}
}

func TestSetNumberConversion(t *testing.T) {
	var x int
	err := jq.Set(&x, "", 1.1)
	maybeError(t, err)
	mustEqual(t, x, 1)
}
