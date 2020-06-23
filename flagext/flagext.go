package flagext

import (
	"flag"
	"fmt"
	"strconv"
)

// Based on: https://stackoverflow.com/questions/35809252/check-if-flag-was-provided-in-go

// IntPtrFlag wraps an integer flag
type IntPtrFlag struct {
	ptr **int
}

// NewIntPtrFlag holds a flag that if prodived holds the value, if undefined just nil
func NewIntPtrFlag(i **int) *IntPtrFlag {
	return &IntPtrFlag{
		ptr: i,
	}
}

func (f IntPtrFlag) String() string {
	if f.ptr == nil || *f.ptr == nil {
		return ""
	}

	return strconv.Itoa(**f.ptr)
}

// Set applies the flag value
func (f IntPtrFlag) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("not a number")
	}

	*f.ptr = &v

	return nil
}

// StringPtrFlag wraps a string flag
type StringPtrFlag struct {
	ptr **string
}

// NewStringPtrFlag holds a flag that if prodived holds the value, if undefined just nil
func NewStringPtrFlag(s **string) *StringPtrFlag {
	return &StringPtrFlag{
		ptr: s,
	}
}

func (f StringPtrFlag) String() string {
	if f.ptr == nil || *f.ptr == nil {
		return ""
	}

	return **f.ptr
}

// Set applies the flag value
func (f StringPtrFlag) Set(s string) error {
	*f.ptr = &s
	return nil
}

func StringPtr(flagSet *flag.FlagSet, name, usage string) *string {
	s := new(string)
	flag.Var(NewStringPtrFlag(&s), name, usage)

	return s
}
