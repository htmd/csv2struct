package csv2struct

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	defaultCSVTagName     = "csv"
	defaultCSVTagFieldSep = ","
)

type DecodeStruct struct {
	cols       []*csvFieldInfo
	foundCols  []*csvFieldInfo
	recordType reflect.Type

	timeFormat     string
	csvTagName     string
	csvTagFieldSep string
}

type csvFieldInfo struct {
	Header           string
	Required         bool
	RecordIndex      int
	StructFieldIndex int
}

type option func(c *DecodeStruct)

// WithTimeFormat option to change time format
func WithTimeFormat(s string) option {
	return func(c *DecodeStruct) {
		c.timeFormat = s
	}
}

// WithCSVTagName option to change csv tag name
func WithCSVTagName(n string) option {
	return func(c *DecodeStruct) {
		c.csvTagName = n
	}
}

// WithCSVTagFieldSep option to change csv tag field separator
func WithCSVTagFieldSep(s string) option {
	return func(c *DecodeStruct) {
		c.csvTagFieldSep = s
	}
}

// NewDecodeStruct return pointer to DecodeStruct with given struct and options
func NewDecodeStruct(v interface{}, opts ...option) *DecodeStruct {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		panic("argument of NewDecodeStruct must be an struct or pointer to struct")
	}
	var (
		headerName, csvTag string
		required           bool
	)
	reader := &DecodeStruct{
		timeFormat:     time.RFC3339,
		csvTagName:     defaultCSVTagName,
		csvTagFieldSep: defaultCSVTagFieldSep,
	}
	for _, opt := range opts {
		opt(reader)
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !isSupportedField(f.Type) {
			panic(fmt.Sprintf("CSV struct reader does not support struct field with type: %s", f.Type.String()))
		}
		headerName = f.Name
		required = false
		csvTag = f.Tag.Get(reader.csvTagName)
		// ignore the field if tag is "-"
		if csvTag == "-" {
			continue
		}
		if len(csvTag) > 0 {
			tagFields := strings.Split(csvTag, reader.csvTagFieldSep)
			headerName = tagFields[0]
			if len(tagFields) > 1 && tagFields[1] == "required" {
				required = true
			}
		}
		reader.cols = append(
			reader.cols,
			&csvFieldInfo{
				Header:           strings.ToLower(headerName),
				Required:         required,
				StructFieldIndex: i,
				RecordIndex:      -1, // special flag to indicate un-initialized status
			},
		)
	}
	reader.recordType = rt
	return reader
}

func (r *DecodeStruct) ParseHeader(header []string) error {
	var found bool
	// must call reset() to prevent incorrect initialized status
	r.reset()
	for i, col := range header {
		found = false
		col = strings.ToLower(strings.TrimSpace(col))
		for _, f := range r.cols {
			if f.Header == col {
				f.RecordIndex = i
				found = true
				r.foundCols = append(r.foundCols, f)
			}
		}
		if !found {
			return NewIncorrectFileErr(fmt.Sprintf("Unexpected column %q", header[i]))
		}
	}
	// check if all required col is already in header
	for _, f := range r.cols {
		if f.Required && f.RecordIndex == -1 {
			return NewIncorrectFileErr(fmt.Sprintf("Mandatory column %q is missing", f.Header))
		}
	}
	return nil
}

// GetStruct create new struct pointer then unmarshal record to that struct
func (r *DecodeStruct) GetStruct(record []string) (v interface{}, err error) {
	v = reflect.New(r.recordType).Interface()
	if err = r.UnmarshalCSV(record, v); err != nil {
		return nil, err
	}
	return v, nil
}

// UnmarshalCSV convert csv row to container v
// v must be pointer to struct that have same type with struct in constructor function
func (r *DecodeStruct) UnmarshalCSV(record []string, v interface{}) (err error) {
	if len(record) != len(r.foundCols) {
		return errors.New("csv record must have same column with csv header")
	}
	rv := reflect.ValueOf(v).Elem()
	if rv.Type() != r.recordType {
		return fmt.Errorf("second argument of UnmarshalCSV function must be a pointer to %s", r.recordType.String())
	}

	return r.unmarshal(rv, record)
}

// reset reset reader to initial status
func (r *DecodeStruct) reset() *DecodeStruct {
	for _, col := range r.cols {
		col.RecordIndex = -1
	}
	r.foundCols = nil
	return r
}

func (r *DecodeStruct) unmarshal(rv reflect.Value, record []string) error {
	for _, c := range r.foundCols {
		s := record[c.RecordIndex]
		if !c.Required && len(s) == 0 {
			continue // ignore empty value for optional column
		}
		f := rv.Field(c.StructFieldIndex)
		if f.CanSet() {
			if err := r.setField(f, s); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *DecodeStruct) setField(f reflect.Value, s string) error {
	switch f.Kind() {
	case reflect.Ptr:
		z := reflect.New(f.Type().Elem())
		f.Set(z)
		f = reflect.Indirect(f)
		return r.setField(f, s)
	case reflect.String:
		f.SetString(s)
		return nil
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		f.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		f.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ui, err := strconv.ParseUint(s, 10, 0)
		if err != nil {
			return err
		}
		f.SetUint(ui)
		return nil
	case reflect.Float32, reflect.Float64:
		fv, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		f.SetFloat(fv)
		return nil
	case reflect.Struct:
		// support struct for only time.Time
		if s := f.Type().String(); s != "time.Time" {
			return fmt.Errorf("CSV struct reader does not support struct field with type: %s", s)
		}
		t, err := time.Parse(r.timeFormat, s)
		if err != nil {
			return err
		}
		f.Set(reflect.ValueOf(t))
		return nil
	default:
		return fmt.Errorf("CSV struct reader does not support struct field with type: %s", f.Type().String())
	}
}

// isSupportedField
func isSupportedField(f reflect.Type) bool {
	switch f.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		return true
	case reflect.Struct:
		if f.String() == "time.Time" {
			return true
		}
		return false
	case reflect.Ptr:
		return isSupportedField(f.Elem())
	default:
		return false
	}
}
