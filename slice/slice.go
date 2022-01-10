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

func FirstEqual[K comparable](a, b []K, fn func(K) K) *K {
	for _, aa := range a {
		for _, bb := range b {
			if fn(aa) == fn(bb) {
				return &aa
			}
		}
	}

	return nil
}

type Stringer interface {
	String() string
}

func Concat[T Stringer](a, b []T) []string {
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

func ConcatStr(a, b []string) []string {
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

type PartitionResult[T Stringer] struct {
	Element       T
	OriginalIndex int
}

func (p *PartitionResult[T]) String() string {
	return p.Element.String()
}

type PartitionStrResult struct {
	Element       string
	OriginalIndex int
}

func (p *PartitionStrResult) String() string {
	return p.Element
}

func PartitionStr[T Stringer, K comparable](haystack []T, fn func(T) K) map[K][]PartitionResult[T] {
	p := make(map[K][]PartitionResult[T])

	for i, e := range haystack {
		key := fn(e)
		p[key] = append(p[key], PartitionResult[T]{Element: e, OriginalIndex: i})
	}

	return p
}

// UnionButIntersectionFromA takes two slices (a and b) and combines the slices (incl. duplicates from 'b').
// If an element in 'a' is also present in 'b' (intersection) the value from 'b' (incl. duplicates) is used instead.
func UnionButIntersectionFromB[T any, K comparable](a, b []T, fn func(T) K) []T {
	union := make([]T, len(b), len(a)+len(b))
	bElComps := make([]K, len(b))

	// 'b' has priority over 'a' so keep all the elements (incl. duplicates)
	for i, bb := range b {
		union[i] = bb
		// convert elements from 'b' as comparable: O(n)
		bElComps[i] = fn(bb)
	}

	// convert elements from 'a' as comparable: O(m)
	for _, aa := range a {
		aElComp := fn(aa)
		isAlsoInB := false

		for _, bElComp := range bElComps {
			if aElComp == bElComp {
				isAlsoInB = true
				break
			}
		}

		if !isAlsoInB {
			union = append(union, aa)
		}
	}

	return union
}
