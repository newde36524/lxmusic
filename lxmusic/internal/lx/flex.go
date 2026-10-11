package lx

import (
	"encoding/json"
	"fmt"
	"strconv"
)

/*
These platform APIs are inconsistent about types: ids come back as numbers in
one call and strings in another. The desktop app sidesteps that with plain JS
dynamic typing; Go needs explicit tolerant types.
*/

// FStr accepts JSON strings, numbers and booleans and normalizes them to a
// string.
type FStr string

func (s *FStr) UnmarshalJSON(b []byte) error {
	*s = ""
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	switch b[0] {
	case '"':
		var x string
		if err := json.Unmarshal(b, &x); err != nil {
			return err
		}
		*s = FStr(x)
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var x float64
		if err := json.Unmarshal(b, &x); err != nil {
			return err
		}
		*s = FStr(numberToString(x))
	default:
		switch string(b) {
		case "true":
			*s = "true"
		case "false":
			*s = "false"
		case "null":
			*s = ""
		default:
			*s = FStr(string(b))
		}
	}
	return nil
}

// FInt accepts JSON numbers and numeric strings.
type FInt int64

func (i *FInt) UnmarshalJSON(b []byte) error {
	*i = 0
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var x string
		if err := json.Unmarshal(b, &x); err != nil {
			return err
		}
		if x == "" {
			return nil
		}
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			*i = FInt(n)
			return nil
		}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			*i = FInt(int64(f))
			return nil
		}
		return nil
	}
	var x float64
	if err := json.Unmarshal(b, &x); err != nil {
		return nil
	}
	*i = FInt(int64(x))
	return nil
}

// FStrSlice tolerates either a JSON array of scalars or a single scalar.
type FStrSlice []string

func (s *FStrSlice) UnmarshalJSON(b []byte) error {
	*s = nil
	if len(b) == 0 || string(b) == "null" || b[0] == '"' {
		var x string
		if err := json.Unmarshal(b, &x); err == nil && x != "" {
			*s = []string{x}
		}
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil
	}
	for _, r := range raw {
		var st FStr
		if err := json.Unmarshal(r, &st); err == nil {
			*s = append(*s, string(st))
		}
	}
	return nil
}

func numberToString(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// FormatBytes renders a byte count the way the desktop app does (2 decimals,
// MB unless the value is under 1 MB).
func FormatBytes(n int64) string {
	if n <= 0 {
		return "0"
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.2fKB", float64(n)/1024)
	}
	return fmt.Sprintf("%.2fMB", float64(n)/(1024*1024))
}

// FormatDuration renders seconds as mm:ss.
func FormatDuration(sec int) string {
	if sec < 0 {
		sec = 0
	}
	m := sec / 60
	s := sec % 60
	if m < 10 {
		return fmt.Sprintf("0%d:%02d", m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
