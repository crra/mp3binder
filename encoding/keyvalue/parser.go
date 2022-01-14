package keyvalue

import (
	"strings"
	"unicode"
)

const (
	keyValueSeparator = ":="
	pairSeparator     = ", "
)

// StringAsStringMap takes a string (e.g "key=value,key2=value") and puts the keys
// and values in a map of strings.
func StringAsStringMap(input string) (map[string]string, error) {
	kv := make(map[string]string)

	// separate buffers for key and value
	keyB := strings.Builder{}
	valueB := strings.Builder{}
	// pointer to current buffer
	b := &keyB

	// rune of the last quote. Allows nesting "'" and '"'
	lastQuote := rune(0)
	finalIndexOfInput := len(input) - 1

	for i, r := range input {
		outsideOfQuote := lastQuote == rune(0)
		endOfPair := i == finalIndexOfInput

		switch {
		case r == lastQuote:
			// closing quote, swallow
			lastQuote = rune(0)
		case outsideOfQuote && unicode.In(r, unicode.Quotation_Mark):
			// opening quote, swallow
			lastQuote = r
		case outsideOfQuote && strings.ContainsRune(keyValueSeparator, r):
			// start value, swallow
			// switch buffer
			b = &valueB
		case outsideOfQuote && unicode.In(r, unicode.White_Space):
			// whitespace outside of quote, swallow
		case outsideOfQuote && strings.ContainsRune(pairSeparator, r):
			// pair separator, swallow
			endOfPair = true
		default:
			// collect in current buffer (key or value) inside or outside of quotes
			b.WriteRune(r)
		}

		if endOfPair {
			kv[keyB.String()] = valueB.String()
			keyB.Reset()
			valueB.Reset()
			b = &keyB
		}
	}

	return kv, nil
}
