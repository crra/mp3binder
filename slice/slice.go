package slice

import (
	"strings"
)

const (
	interlaceScaleFactor = 2
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

// A partition result wraps an element with its original index before the partitioning to be able
// preserve the original order if unified again.
type PartitionResult[T Stringer] struct {
	Element       T
	OriginalIndex int
}

func (p PartitionResult[T]) String() string {
	return (p.Element).String()
}

// String 'unboxes' any type that impelements the `Stringer` interface to a string which also happens to implement `comparable`.
func String[T Stringer](t T) string { return t.String() }

// Partition takes a slice of generic types and applies a partition function to it. It wraps the partition results with an index
// so that it can be sorted to preserve the initial order.
func Partition[T Stringer, K comparable](haystack []T, fn func(T) K) map[K][]PartitionResult[T] {
	p := make(map[K][]PartitionResult[T])

	for i, e := range haystack {
		key := fn(e)
		p[key] = append(p[key], PartitionResult[T]{Element: e, OriginalIndex: i})
	}

	return p
}

// UnionButIntersectionFromA takes two slices (a and b) and combines the slices (incl. duplicates from 'b').
// If an element in 'a' is also present in 'b' (intersection) the value from 'b' (incl. duplicates) is used instead.
func UnionButIntersectionFromB[T Stringer, K comparable](a, b []T, fn func(T) K) []T {
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

// Interlace works similar to `strings.Join`, it takes a slice and adds an interlace between each element.
func Interlace[T any](l []T, interlace T) []T {
	// interlacing makes sense for more than one elements
	if len(l) <= 1 {
		return l
	}

	// -1 because there is no interlace at the end
	interlaced := make([]T, len(l)*interlaceScaleFactor-1)
	interlaced[0] = l[0]

	for i, e := range l[1:] {
		eI := i*interlaceScaleFactor + 1
		interlaced[eI] = interlace
		interlaced[eI+1] = e
	}

	return interlaced
}

func IndexAfterInterlace(interlacedLength, oldIndex int) int {
	if interlacedLength <= 1 {
		return oldIndex
	}

	return oldIndex * interlaceScaleFactor
}
