package commons

import (
	"encoding/json"
	"fmt"
	"testing"
)

type Field1 struct {
	Value        string
	AnotherValue int
}

func (f *Field1) UnmarshalJSON(input []byte) error {
	type _real struct {
		Value string
	}
	var real _real
	err := json.Unmarshal(input, &real)
	if err != nil {
		return err
	}
	f.Value = "Field1" + real.Value
	return nil
}

type A struct {
	Field        Field1
	AnotherField string
}

func fixFieldLazy[T any](input []byte, path []string, fix func(*T) error) ([]byte, error) {
	//fmt.Printf("fixFieldLazy: %v, %v\n", string(input), path)
	// unmarshal to raw maps
	var raw map[string]json.RawMessage
	err := json.Unmarshal(input, &raw)
	if err != nil {
		return nil, err
	}
	//path := strings.Split(field, ".")

	next, ok := raw[path[0]]
	if !ok {
		return nil, fmt.Errorf("field %s not found", path[0])
	}

	if len(path) == 1 {
		var fieldTyped T
		err = json.Unmarshal(next, &fieldTyped)
		if err != nil {
			return nil, err
		}
		err = fix(&fieldTyped)
		if err != nil {
			return nil, err
		}
		fieldRaw, err := json.Marshal(fieldTyped)
		if err != nil {
			return nil, err
		}
		raw[path[0]] = json.RawMessage(fieldRaw)
		return json.Marshal(raw)
	} else {
		// recurse to submap
		submapRaw, err := fixFieldLazy[T](next, path[1:], fix)
		if err != nil {
			return nil, err
		}
		raw[path[0]] = json.RawMessage(submapRaw)
		return json.Marshal(raw)
	}
}

type AnyMapFixWrap A

func (b *AnyMapFixWrap) UnmarshalJSON(input []byte) error {

	raw, err := FixJsonFields(input, true, []string{"Field", "Value"}, func(s *string) error {
		*s = "FIXED" + *s
		return nil
	})

	if err != nil {
		return err
	}

	var a A
	err = json.Unmarshal(raw, &a)
	*b = AnyMapFixWrap(a)
	return err
}

type LazyUnmarshalFixWrap A

func (b *LazyUnmarshalFixWrap) UnmarshalJSON(input []byte) error {
	fixedInput, err := fixFieldLazy[string](input, []string{"Field", "Value"}, func(s *string) error {
		*s = "FIXED2" + *s
		return nil
	})

	if err != nil {
		return err
	}

	//fmt.Printf("fixedInput: %v\n", string(fixedInput))

	var a A
	err = json.Unmarshal(fixedInput, &a)
	*b = LazyUnmarshalFixWrap(a)
	return err
}

func TestFixJsonFields(t *testing.T) {
	input := []byte("{\"Field\":{\"Value\":\"xxx\"}}")
	fmt.Printf("Original: %v\n", string(input))
	var a AnyMapFixWrap
	err := json.Unmarshal(input, &a)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fixed: %v\n", a)

	if a.Field.Value != "Field1FIXEDxxx" {
		t.Errorf("Field.Value is not FIXEDxxx")
	}
}

func TestFixJsonFieldsIgnoreNonExisting(t *testing.T) {
	input := []byte("{\"AnotherField\":\"somevalue\"}")
	fmt.Printf("Original: %v\n", string(input))
	var a AnyMapFixWrap
	err := json.Unmarshal(input, &a)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Fixed: %v\n", a)

	if a.Field.Value != "" {
		t.Errorf("Field.Value is not empty")
	}

	if a.AnotherField != "somevalue" {
		t.Errorf("AnotherField is not somevalue")
	}
}

func BenchmarkFixAnyMap(b *testing.B) {
	input := []byte("{\"Field\":{\"Value\":\"xxx\"}}")
	fmt.Printf("Original: %v\n", string(input))

	for i := 0; i < b.N; i++ {
		var a AnyMapFixWrap
		err := json.Unmarshal(input, &a)
		if err != nil {
			panic(err)
		}

		var anyMap map[string]any
		err = json.Unmarshal(input, &anyMap)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkLazyUnmarshal(b *testing.B) {
	input := []byte("{\"Field\":{\"Value\":\"xxx\"}}")
	fmt.Printf("Original: %v\n", string(input))

	for i := 0; i < b.N; i++ {
		var a LazyUnmarshalFixWrap
		err := json.Unmarshal(input, &a)
		if err != nil {
			panic(err)
		}

		var anyMap map[string]any
		err = json.Unmarshal(input, &anyMap)
		if err != nil {
			panic(err)
		}
	}
}
