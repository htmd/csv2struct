# csv2struct.DecodeStruct
An decoder to convert CSV record ([]string) to struct.
You can define CSV header in struct tag. The decoder will
validate if CSV file has correct header or not.

Decoder support only scala types (and pointer to them): string, boolean, int, uint, float
and time. You can define time format in constructor option

# Usage

```
import (
    "time"
    "github.com/htmd/csv2struct"
)

type Record struct {
	StringField string  `csv:"String Field;required"`
	IntField    int     `csv:"Integer Field;required"`
	UintField   uint    `csv:"Unsigned Integer Field;required"`
	BoolField   bool    `csv:"Boolean Field;required"`
	FloatField  float64 `csv:"Float Field;required"`

	OptionalTimeField  time.Time `csv:"Optional Time Field"`
	OptionalIntField   int
	OptionalIntPointer *int
}

decoder := csv2struct.NewDecodeStruct(
    &Record{},
    csv2struct.WithTimeFormat(time.RFC850),//optional
    csv2struct.WithCSVTagName("csv"), // optional
)

// use decoder to validate CSV file header
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
// process error
...

// start convert csv record data to Record struct
v := new(Record)
rec := []string{
    "field 1",
    "100",
    "-30",
    "-200",
    "true",
    "50",
    "-10",
    "2017-10-21T12:00:00Z07:00",
}
err = decoder.UnmarshalCSV(rec, v)
if err != nil {
    return err
}
fmt.Printf("got struct data: %+v", v)
...

```