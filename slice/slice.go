package slice

import (
	"strings"
)

func Contains[K comparable](haystack []K, needle K) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func FirstEqual[K comparable](a []K, b []K) *K {
	for _, aa := range a {
		for _, bb := range b {
			if aa == bb {
				return &aa
			}
		}
	}

	return nil
}

type Stringer interface {
	String() string
}

func Concat[T Stringer](a []T, b []T) []string {
	values := make([]string, len(a)*len(b))

	index := 0
	for _, aa := range a {
		for _, bb := range b {
			var sb strings.Builder
			sb.WriteString(aa.String())
			sb.WriteString(bb.String())

			values[index] = sb.String()
			index++
		}
	}

	return values
}

func ConcatStr(a []string, b []string) []string {
	values := make([]string, len(a)*len(b))

	index := 0
	for _, aa := range a {
		for _, bb := range b {
			var sb strings.Builder
			sb.WriteString(aa)
			sb.WriteString(bb)

			values[index] = sb.String()
			index++
		}
	}

	return values
}

func Map[T any, U any](list []T, fn func(T) U) []U {
	values := make([]U, len(list))
	for i, v := range list {
		values[i] = fn(v)
	}

	return values
}
