package cli

import (
	"testing"

	"github.com/carolynvs/aferox"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCopyIndex(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	numberOfFiles := 2
	afero.WriteFile(fs, "/"+validFileName1, []byte("1"), 0644)
	afero.WriteFile(fs, "/"+validFileName2, []byte("2"), 0644)

	for i := 1; i <= numberOfFiles+1; i++ {
		a := &application{
			fs:                aferox.NewAferox("/", fs),
			copyTagsFromIndex: i,
		}
		err := a.args(nil, []string{"."})
		if i <= numberOfFiles {
			assert.NoError(t, err)
		} else {
			assert.ErrorIs(t, err, ErrInvalidIndex)
		}
	}
}
