package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringEmpty(t *testing.T) {
	value := ""
	defaultV := "random"
	assert.Equal(t, defaultV, OrDefaultStr(value, defaultV))
}

func TestNonEmpty(t *testing.T) {
	value := "value"
	defaultV := "random"
	assert.Equal(t, value, OrDefaultStr(value, defaultV))
}

func TestPointerOfString(t *testing.T) {
	value := "a string"
	r := PointerOf(value)
	if assert.NotNil(t, r) {
		assert.Equal(t, value, *r)
	}
}
