package csv2struct_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/htmd/csv2struct"
)

type Record struct {
	StringField string  `csv:"String Field,required"`
	IntField    int     `csv:"Integer Field,required"`
	UintField   uint    `csv:"Unsigned Integer Field,required"`
	BoolField   bool    `csv:"Boolean Field,required"`
	FloatField  float64 `csv:"Float Field,required"`

	OptionalTimeField  time.Time `csv:"Optional Time Field"`
	OptionalIntField   int
	OptionalIntPointer *int
}

func TestNewDecodeStruct(t *testing.T) {
	s := struct {
		FieldA map[string]string
	}{}

	err := runCatchPanic(func() {
		var _ = csv2struct.NewDecodeStruct(s)
	})
	if err == nil {
		t.Errorf("using map data type for struct field must cause panic")
	}
	err = runCatchPanic(func() {
		var _ = csv2struct.NewDecodeStruct(
			struct {
				FieldA struct {
					FieldB string
				}
			}{},
		)
	})
	if err == nil {
		t.Errorf("using struct data type for struct field must cause panic")
	}
}

func TestDecodeStruct_ParseHeader(t *testing.T) {
	decoder := csv2struct.NewDecodeStruct(&Record{})
	header := []string{
		"String Field",
		"Unsigned Integer Field",
		"Integer Field",
		"Float Field",
		"Boolean Field",
		"OptionalIntField",
	}

	err := decoder.ParseHeader(header)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	header = []string{
		"Unsigned Integer Field",
		"Integer Field",
		"Float Field",
		"Boolean Field",
		"OptionalIntField",
	}
	err = decoder.ParseHeader(header)
	expect := csv2struct.NewIncorrectFileErr("Mandatory column \"string field\" is missing")
	if err != expect {
		t.Errorf("expecting error: %s but got: %s", expect, err)
	}

	header = []string{
		"Unsigned Integer Field",
		"Integer Field",
		"Float Field",
		"Boolean Field",
		"String Field",
	}
	err = decoder.ParseHeader(header)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestDecodeStruct_GetStruct(t *testing.T) {
	decoder := csv2struct.NewDecodeStruct(&Record{})
	header := []string{
		"String Field",
		"Unsigned Integer Field",
		"Integer Field",
		"Float Field",
		"Boolean Field",
		"OptionalIntField",
		"OptionalIntPointer",
		"Optional Time Field",
	}
	err := decoder.ParseHeader(header)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	now := time.Date(2017, 10, 9, 12, 30, 30, 0, time.Local)
	rec := []string{
		"field 1",
		"100",
		"-30",
		"-200",
		"true",
		"50",
		"-10",
		now.Format(time.RFC3339),
	}

	v, err := decoder.GetStruct(rec)
	got, ok := v.(*Record)
	if !ok {
		t.Errorf("unexpected struct type: %T", v)
	}
	pInt := -10
	expect := &Record{
		StringField:        "field 1",
		IntField:           -30,
		UintField:          100,
		BoolField:          true,
		FloatField:         -200.0,
		OptionalIntField:   50,
		OptionalIntPointer: &pInt,
		OptionalTimeField:  now,
	}
	if !reflect.DeepEqual(got, expect) {
		t.Errorf("expecting: %+v \nbut got: %+v", expect, got)
	}
}

func TestDecodeStruct_UnmarshalCSV(t *testing.T) {
	decoder := csv2struct.NewDecodeStruct(&Record{})
	header := []string{
		"String Field",
		"Unsigned Integer Field",
		"Integer Field",
		"Float Field",
		"Boolean Field",
		"OptionalIntField",
		"OptionalIntPointer",
		"Optional Time Field",
	}
	err := decoder.ParseHeader(header)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	now := time.Date(2017, 10, 9, 12, 30, 30, 0, time.Local)
	rec := []string{
		"xxx",
	}
	v := &struct {
		Field1 int
	}{}
	err = decoder.UnmarshalCSV(rec, v)
	expect := errors.New("csv record must have same column with csv header")
	if !reflect.DeepEqual(err, expect) {
		t.Errorf("expecting error: %s but got: %+v", expect, err)
	}
	rec = []string{
		"field 1",
		"100",
		"-30",
		"-200",
		"true",
		"50",
		"-10",
		now.Format(time.RFC3339),
	}
	err = decoder.UnmarshalCSV(rec, v)
	expect = fmt.Errorf("second argument of UnmarshalCSV function must be a pointer to %s", reflect.TypeOf(Record{}).String())
	if !reflect.DeepEqual(err, expect) {
		t.Errorf("expecting error: %s but got %+v", expect, err)
	}

	correctValue := &Record{}
	err = decoder.UnmarshalCSV(rec, correctValue)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	pInt := -10
	expectValue := &Record{
		StringField:        "field 1",
		IntField:           -30,
		UintField:          100,
		BoolField:          true,
		FloatField:         -200.0,
		OptionalIntField:   50,
		OptionalIntPointer: &pInt,
		OptionalTimeField:  now,
	}
	if !reflect.DeepEqual(correctValue, expectValue) {
		t.Errorf("expecting: %+v \nbut got: %+v", expectValue, correctValue)
	}
}

func runCatchPanic(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	fn()
	return
}
