package markdown

import "testing"

func TestTable(t *testing.T) {
	rows := [][]string{{"entity", "n-revs"}, {"a|b.go", "3"}, {"c.go", "1"}}
	tests := []struct {
		name  string
		rows  [][]string
		limit int
		want  string
	}{
		{"all rows, markup escaped", rows, 0, "| entity | n-revs |\n| --- | --- |\n| a\\|b.go | 3 |\n| c.go | 1 |\n"},
		{"limited", rows, 1, "| entity | n-revs |\n| --- | --- |\n| a\\|b.go | 3 |\n"},
		{"header only", rows[:1], 0, "_None._\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Table(tt.rows, tt.limit); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		limit int
		want  string
	}{
		{"all items, markup escaped", []string{"__init__.py", "b.go"}, 0, "1. \\_\\_init\\_\\_.py\n2. b.go\n"},
		{"limited", []string{"a.go", "b.go"}, 1, "1. a.go\n"},
		{"empty", nil, 0, "_None._\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := List(tt.items, tt.limit); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
