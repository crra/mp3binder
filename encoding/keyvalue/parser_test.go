package keyvalue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()
	singleValue := map[string]string{"key": "value"}
	twoValues := map[string]string{"key1": "value1", "key2": "value2"}
	threeValues := map[string]string{"key1": "value1", "key2": "value2", "key3": "value3"}

	for _, f := range []struct {
		title    string
		input    string
		expected map[string]string
	}{
		// key value
		{title: "equal", input: "key=value", expected: singleValue},
		{title: "quoted", input: `"key"="value"`, expected: singleValue},
		{title: "quoted in quote", input: `"key"="their's value"`, expected: map[string]string{"key": "their's value"}},
		{title: "colon", input: "key:value", expected: singleValue},

		// whitespace
		{title: "Space prefix", input: " key = value ", expected: singleValue},
		{title: "FF prefix", input: "\fkey=value", expected: singleValue},
		{title: "Tab prefix", input: "\tkey=value", expected: singleValue},
		{title: "Mix prefix", input: " \f\tkey=value", expected: singleValue},

		// quoted
		{title: "Key quoted >\"<", input: `"space key"=value`, expected: map[string]string{"space key": "value"}},
		{title: "Value quoted >\"<", input: `key="space value"`, expected: map[string]string{"key": "space value"}},
		{title: "Key and value quoted >\"<", input: `"key space"="space value"`, expected: map[string]string{"key space": "space value"}},
		{title: "Key quoted >'<", input: `'space key'=value`, expected: map[string]string{"space key": "value"}},
		{title: "Value quoted >'<", input: `key='space value'`, expected: map[string]string{"key": "space value"}},

		// multiple keys
		{title: "comma separated", input: "key1=value1,key2=value2", expected: twoValues},
		{title: "space separated", input: "key1=value1, key2=value2", expected: twoValues},
		{title: "comma separated", input: "key1=value1,key2=value2,key3=value3", expected: threeValues},
		{title: "space separated", input: "key1=value1, key2=value2, key3=value3", expected: threeValues},
		{title: "Space between", input: "key1=value1,  key2=value2", expected: twoValues},
		{title: "quoted with comma", input: `key1="value1", key2=value2`, expected: twoValues},
		{title: "quoted with space", input: `key1="value1", key2=value2`, expected: twoValues},
		{title: "keys only", input: "key1,  key2 ", expected: map[string]string{"key1": "", "key2": ""}},
		{title: "keys only with equals and colon", input: "    key1=,  key2 : ", expected: map[string]string{"key1": "", "key2": ""}},

		// separator in key/value
		{title: "comma in key", input: `"key,1"="value",key2=value2`, expected: map[string]string{"key,1": "value", "key2": "value2"}},
		{title: "comma in value", input: `key="value,1",key2=value2`, expected: map[string]string{"key": "value,1", "key2": "value2"}},

		// equal in key/value
		{title: "equal in key", input: `"key=1"="value",key2=value2`, expected: map[string]string{"key=1": "value", "key2": "value2"}},
		{title: "equal in value", input: `key="value=1",key2=value2`, expected: map[string]string{"key": "value=1", "key2": "value2"}},
	} {
		f := f // pin
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()

			result, err := StringAsStringMap(f.input)
			if assert.NoError(t, err) {
				assert.Equal(t, f.expected, result, "Input: '%s'", f.input)
			}
		})
	}
}
