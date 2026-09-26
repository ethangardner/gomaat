package analysis

// formatRows renders a header row and data rows produced by rowFn for each item.
func formatRows[T any](header []string, items []T, rowFn func(T) []string) [][]string {
	out := make([][]string, len(items)+1)
	out[0] = header
	for i, item := range items {
		out[i+1] = rowFn(item)
	}
	return out
}
