package entity

import (
	"testing"
)

type sample struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJSONFieldScanFromString(t *testing.T) {
	var field JSONField[sample]
	if err := field.Scan(`{"name":"alpha","age":42}`); err != nil {
		t.Fatalf("scan string: %v", err)
	}
	if field.Data.Name != "alpha" || field.Data.Age != 42 {
		t.Fatalf("got %+v, want {Name:alpha, Age:42}", field.Data)
	}
}

func TestJSONFieldScanFromBytes(t *testing.T) {
	var field JSONField[sample]
	if err := field.Scan([]byte(`{"name":"beta","age":7}`)); err != nil {
		t.Fatalf("scan bytes: %v", err)
	}
	if field.Data.Name != "beta" || field.Data.Age != 7 {
		t.Fatalf("got %+v, want {Name:beta, Age:7}", field.Data)
	}
}

func TestJSONFieldScanReturnsErrorForUnsupportedType(t *testing.T) {
	var field JSONField[sample]
	if err := field.Scan(123); err == nil {
		t.Fatal("got nil error, want error for unsupported scan source")
	}
}

func TestJSONFieldScanReturnsErrorForInvalidJSON(t *testing.T) {
	var field JSONField[sample]
	if err := field.Scan(`{"name":`); err == nil {
		t.Fatal("got nil error, want JSON parse error")
	}
}
