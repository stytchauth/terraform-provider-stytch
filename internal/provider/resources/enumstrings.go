package resources

func toStrings[T ~string](values []T) []string {
	strs := make([]string, len(values))
	for i, v := range values {
		strs[i] = string(v)
	}
	return strs
}
