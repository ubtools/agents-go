package commons

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ErrFieldNotExist struct {
	Path []string
}

func (e *ErrFieldNotExist) Error() string {
	return fmt.Sprintf("field %s not found", strings.Join(e.Path, "."))
}

// FixJsonFields applies a series of fixes to the JSON fields in the input byte slice.
// To inject into UnmarshalJSON method of wrapper struct
// It takes a variadic number of fixes, where each fix is a pair of path and fix function.
// The path is a slice of strings representing the path to the field in the JSON structure.
// If ignoreMissing is set to true, the function will ignore fields that do not exist in the JSON structure.
// The function returns the modified JSON byte slice and an error if any occurred during the fixing process.
func FixJsonFields(input []byte, ignoreMissing bool, fixes ...any) ([]byte, error) {
	if len(fixes)%2 != 0 {
		return nil, fmt.Errorf("fixes must be pairs of path and fix function")
	}
	// unmarshal to raw maps
	var raw map[string]any
	err := json.Unmarshal(input, &raw)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(fixes); i += 2 {
		path, ok := fixes[i].([]string)
		if !ok {
			return nil, fmt.Errorf("fixes[%d] is not a string slice", i)
		}
		fix, ok := fixes[i+1].(func(*string) error)
		if !ok {
			return nil, fmt.Errorf("fixes[%d] is not a fix function", i+1)
		}
		err = FixField(raw, path, fix)
		if err != nil {
			if _, ok := err.(*ErrFieldNotExist); ok && ignoreMissing {
				continue
			}
			return nil, err
		}
	}

	return json.Marshal(raw)
}

func FixField[T any](raw map[string]any, path []string, fix func(*T) error) error {
	var prevMap map[string]any = raw
	var cur any = raw
	for idx, p := range path {
		switch curTyped := cur.(type) {
		case map[string]any:
			next, ok := curTyped[p]
			if !ok {
				return &ErrFieldNotExist{Path: path[:idx+1]}
			}
			cur = next
			prevMap = curTyped
		default:
			if idx < len(path)-1 {
				return fmt.Errorf("field %s is not a map", strings.Join(path[:idx+1], "."))
			}
		}
	}

	rawField, ok := cur.(T)
	if !ok {
		return fmt.Errorf("field %s is invalid type", strings.Join(path, "."))
	}

	err := fix(&rawField)
	if err != nil {
		return err
	}

	prevMap[path[len(path)-1]] = rawField

	return nil
}

func FixerZeroHash(s *string) error {
	if s != nil {
		if *s == "0x" {
			*s = "0x0000000000000000000000000000000000000000000000000000000000000000"
		}
	}
	return nil
}

func FixerZeroUint64(s *string) error {
	if s != nil && *s == "0x" {
		*s = "0x0"
	}
	return nil
}
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func TrimLeftStrZeros(s string) string {
	idx := 0
	for ; idx < len(s); idx++ {
		if s[idx] != '0' {
			break
		}
	}
	return s[idx:]
}

func FixerHexStripLeadingZeros(s *string) error {
	if s != nil && has0xPrefix(*s) {
		*s = "0x" + TrimLeftStrZeros((*s)[2:])
		return FixerZeroUint64(s)
	}
	return nil
}
