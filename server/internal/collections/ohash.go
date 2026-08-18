package collections

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OhashSerialize ports ohash serialize() from unjs/ohash v2, used for
// collection hashing. It produces deterministic key-sorted single-quoted
// serialization: {key:'value',n:1}, arrays [..], Dates as Date(ISO).
func OhashSerialize(v any) string {
	return serialize(v)
}

func serialize(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return "'" + t + "'"
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return jsNumber(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case []any:
		var b strings.Builder
		b.WriteByte('[')
		for i, item := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(serialize(item))
		}
		b.WriteByte(']')
		return b.String()
	case map[string]any:
		return serializeMap(t)
	case time.Time:
		return "Date(" + t.UTC().Format("2006-01-02T15:04:05.000Z07:00") + ")"
	}
	return fmt.Sprintf("%v", v)
}

func serializeMap(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return compareStrings(keys[i], keys[j]) < 0 })
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(serialize(m[k]))
	}
	b.WriteByte('}')
	return b.String()
}

// jsNumber mirrors JS String(number).
func jsNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	// JS exponential format uses no leading zeros in exponent, e.g. 1e+21
	s := strconv.FormatFloat(f, 'e', -1, 64)
	// Go gives 1e+21 / 1e-07; JS gives 1e+21 / 1e-7 -> strip leading zeros
	if idx := strings.IndexByte(s, 'e'); idx != -1 {
		exp := s[idx+1:]
		sign := ""
		if strings.HasPrefix(exp, "+") || strings.HasPrefix(exp, "-") {
			sign = exp[:1]
			exp = exp[1:]
		}
		exp = strings.TrimLeft(exp, "0")
		if exp == "" {
			exp = "0"
		}
		s = s[:idx+1] + sign + exp
	}
	return s
}

// asciiOrder mirrors ohash's asciiOrder constant and the weight computation.
const asciiOrder = " _-,;:!?.'\"()[]{}@*/\\&#%`^+<=>|~$0123456789abcdefghijklmnopqrstuvwxyz"

var asciiWeights = func() [128]uint8 {
	var w [128]uint8
	for i := 0; i < 69; i++ {
		w[asciiOrder[i]] = uint8(i + 1)
	}
	for code := 65; code <= 90; code++ {
		w[code] = w[code+32]
	}
	return w
}()

// compareStrings mirrors ohash compareStrings.
func compareStrings(a, b string) int {
	if a == b {
		return 0
	}
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	tieBreaker := 0
	for i := 0; i < length; i++ {
		codeA := a[i]
		codeB := b[i]
		if codeA == codeB {
			continue
		}
		var weightA, weightB uint16
		if codeA < 128 && asciiWeights[codeA] != 0 {
			weightA = uint16(asciiWeights[codeA])
		} else {
			weightA = uint16(codeA) + 128
		}
		if codeB < 128 && asciiWeights[codeB] != 0 {
			weightB = uint16(asciiWeights[codeB])
		} else {
			weightB = uint16(codeB) + 128
		}
		if weightA != weightB {
			if weightA < weightB {
				return -1
			}
			return 1
		}
		if tieBreaker == 0 {
			if codeA > codeB {
				tieBreaker = -1
			} else {
				tieBreaker = 1
			}
		}
	}
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	return tieBreaker
}