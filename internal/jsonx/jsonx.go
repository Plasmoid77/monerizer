// Package jsonx reads upstream JSON tolerantly (spec DATA-03): unknown fields are
// ignored, and each wanted field is extracted on its own so one bad field does
// not discard its neighbours.
package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

// Object is a decoded top-level JSON object with numbers kept as json.Number.
type Object map[string]any

// Decode parses data, which must be a JSON object.
func Decode(data []byte) (Object, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("trailing data after JSON object")
	}
	o, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("top-level JSON value is not an object")
	}
	return o, nil
}

// Field describes one extraction problem; Missing distinguishes an absent or
// null field from a wrongly typed one (only the latter is FIELD_INVALID).
type FieldError struct {
	Path    string
	Missing bool
	Err     error
}

func (e *FieldError) Error() string { return fmt.Sprintf("%s: %v", e.Path, e.Err) }

func (o Object) walk(path ...string) (any, *FieldError) {
	var cur any = map[string]any(o)
	for i, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, &FieldError{Path: join(path[:i+1]), Err: errors.New("parent is not an object")}
		}
		cur, ok = m[p]
		if !ok || cur == nil {
			return nil, &FieldError{Path: join(path[:i+1]), Missing: true, Err: errors.New("missing")}
		}
	}
	return cur, nil
}

func join(p []string) string {
	s := ""
	for i, x := range p {
		if i > 0 {
			s += "."
		}
		s += x
	}
	return s
}

// Float returns a finite number at path.
func (o Object) Float(path ...string) (float64, *FieldError) {
	v, ferr := o.walk(path...)
	if ferr != nil {
		return 0, ferr
	}
	n, ok := v.(json.Number)
	if !ok {
		return 0, &FieldError{Path: join(path), Err: errors.New("not a number")}
	}
	f, err := strconv.ParseFloat(string(n), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return 0, &FieldError{Path: join(path), Err: errors.New("not a non-negative finite number")}
	}
	return f, nil
}

// Int returns a non-negative integer at path (counters, heights, ports).
func (o Object) Int(path ...string) (int64, *FieldError) {
	v, ferr := o.walk(path...)
	if ferr != nil {
		return 0, ferr
	}
	n, ok := v.(json.Number)
	if !ok {
		return 0, &FieldError{Path: join(path), Err: errors.New("not a number")}
	}
	i, err := n.Int64()
	if err != nil || i < 0 {
		return 0, &FieldError{Path: join(path), Err: errors.New("not a non-negative integer")}
	}
	return i, nil
}

// String returns a string at path.
func (o Object) String(path ...string) (string, *FieldError) {
	v, ferr := o.walk(path...)
	if ferr != nil {
		return "", ferr
	}
	s, ok := v.(string)
	if !ok {
		return "", &FieldError{Path: join(path), Err: errors.New("not a string")}
	}
	return s, nil
}

// Index returns element i of the array at path.
func (o Object) Index(i int, path ...string) (any, *FieldError) {
	v, ferr := o.walk(path...)
	if ferr != nil {
		return nil, ferr
	}
	a, ok := v.([]any)
	if !ok {
		return nil, &FieldError{Path: join(path), Err: errors.New("not an array")}
	}
	if i >= len(a) || a[i] == nil {
		return nil, &FieldError{Path: fmt.Sprintf("%s[%d]", join(path), i), Missing: true, Err: errors.New("missing")}
	}
	return a[i], nil
}

// FloatAt returns a finite number at array index i of path.
func (o Object) FloatAt(i int, path ...string) (float64, *FieldError) {
	v, ferr := o.Index(i, path...)
	if ferr != nil {
		return 0, ferr
	}
	p := fmt.Sprintf("%s[%d]", join(path), i)
	n, ok := v.(json.Number)
	if !ok {
		return 0, &FieldError{Path: p, Err: errors.New("not a number")}
	}
	f, err := strconv.ParseFloat(string(n), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < 0 {
		return 0, &FieldError{Path: p, Err: errors.New("not a non-negative finite number")}
	}
	return f, nil
}

// IntAt returns a non-negative integer at array index i of path.
func (o Object) IntAt(i int, path ...string) (int64, *FieldError) {
	v, ferr := o.Index(i, path...)
	if ferr != nil {
		return 0, ferr
	}
	p := fmt.Sprintf("%s[%d]", join(path), i)
	n, ok := v.(json.Number)
	if !ok {
		return 0, &FieldError{Path: p, Err: errors.New("not a number")}
	}
	x, err := n.Int64()
	if err != nil || x < 0 {
		return 0, &FieldError{Path: p, Err: errors.New("not a non-negative integer")}
	}
	return x, nil
}
