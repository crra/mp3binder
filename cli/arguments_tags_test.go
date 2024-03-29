package cli

import (
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/stretchr/testify/assert"
)

func TestCopyIndex(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	files := withTwoValidFiles(fs, root)
	numberOfFiles := len(files)

	for i := 1; i <= numberOfFiles+1; i++ {
		a := newDefaultApplication(aferox.NewAferox(root, fs))
		a.copyTagsFromIndex = i

		err := a.args(nil, []string{"."})
		if i <= numberOfFiles {
			assert.NoError(t, err)
		} else {
			assert.ErrorIs(t, err, ErrInvalidIndex)
		}
	}
}

func TestApplyTags(t *testing.T) {
	t.Parallel()
	root, fs := newTestFilesystem()
	withTwoValidFiles(fs, root)

	for _, f := range []struct {
		title    string
		primed   map[string]string
		args     string
		expected map[string]string
	}{
		{
			title:    "Empty",
			primed:   map[string]string{},
			args:     "",
			expected: map[string]string{},
		},
		{
			title:    "Empty with primed",
			primed:   map[string]string{"foo": "bar"},
			args:     "",
			expected: map[string]string{"foo": "bar"},
		},
		{
			title:    "Removing primed",
			primed:   map[string]string{"foo": "bar"},
			args:     "foo=''",
			expected: map[string]string{"foo": ""},
		},
		{
			title:    "Simple form",
			primed:   map[string]string{},
			args:     "foo=bar",
			expected: map[string]string{"foo": "bar"},
		},
		{
			title:    "Simple form with spaces quoted",
			primed:   map[string]string{},
			args:     "foo='bar baz'",
			expected: map[string]string{"foo": "bar baz"},
		},
		{
			title:    "Simple form with spaces unquoted",
			primed:   map[string]string{},
			args:     "foo=bar baz ",
			expected: map[string]string{"foo": "bar baz"},
		},
	} {
		f := f // pin
		t.Run(f.title, func(t *testing.T) {
			t.Parallel()

			a := newDefaultApplication(aferox.NewAferox(root, fs))
			a.applyTags = f.args
			a.tags = f.primed

			err := a.args(nil, []string{"."})
			if assert.NoError(t, err) {
				assert.Equal(t, f.expected, a.tags)
			}
		})
	}
}
