package value

func OrDefaultStr(v string, defaultV string) string {
	if v == "" {
		return defaultV
	}

	return v
}

func PointerOf[T any](t T) *T { return &t }
