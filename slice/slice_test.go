package slice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const arbitraryInterlaceElement = "_"

func TestInterlaceString(t *testing.T) {
	for _, f := range []struct {
		name     string
		slice    []string
		expected []string
		element  string
		index    int
	}{
		{
			name:     "Empty slice",
			slice:    []string{},
			expected: []string{},
			element:  arbitraryInterlaceElement,
			index:    -1,
		},
		{
			name:     "One element",
			slice:    []string{"one"},
			expected: []string{"one"},
			element:  arbitraryInterlaceElement,
			index:    0,
		},
		{
			name:     "Two elements first",
			slice:    []string{"one", "two"},
			expected: []string{"one", arbitraryInterlaceElement, "two"},
			element:  arbitraryInterlaceElement,
			index:    0,
		},
		{
			name:     "Two elements last",
			slice:    []string{"one", "two"},
			expected: []string{"one", arbitraryInterlaceElement, "two"},
			element:  arbitraryInterlaceElement,
			index:    1,
		},
	} {
		f := f
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			interlaced := Interlace(f.slice, f.element)
			assert.Equal(t, f.expected, interlaced)

			if f.index >= 0 {
				newIndex := IndexAfterInterlace(len(interlaced), f.index)
				assert.Equal(t, f.slice[f.index], interlaced[newIndex])
			}
		})
	}
}

func TestInterlaceBool(t *testing.T) {
	for _, f := range []struct {
		name     string
		slice    []bool
		expected []bool
		element  bool
		index    int
	}{
		{
			name:     "Empty slice",
			slice:    []bool{},
			expected: []bool{},
			element:  false,
			index:    -1,
		},
		{
			name:     "One element",
			slice:    []bool{true},
			expected: []bool{true},
			element:  false,
			index:    0,
		},
		{
			name:     "Two elements first",
			slice:    []bool{true, true},
			expected: []bool{true, false, true},
			element:  false,
			index:    0,
		},
		{
			name:     "Two elements last",
			slice:    []bool{true, true},
			expected: []bool{true, false, true},
			element:  false,
			index:    1,
		},
	} {
		f := f
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			interlaced := Interlace(f.slice, f.element)
			assert.Equal(t, f.expected, interlaced)

			if f.index >= 0 {
				newIndex := IndexAfterInterlace(len(interlaced), f.index)
				assert.Equal(t, f.slice[f.index], interlaced[newIndex])
			}
		})
	}
}
