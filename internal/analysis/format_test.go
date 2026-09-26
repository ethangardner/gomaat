package analysis

import (
	"reflect"
	"testing"
)

func TestFormatRows(t *testing.T) {
	header := []string{"col1", "col2"}
	type item struct {
		a, b string
	}

	t.Run("empty items", func(t *testing.T) {
		got := formatRows(header, []item{}, func(it item) []string {
			return []string{it.a, it.b}
		})
		want := [][]string{{"col1", "col2"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("multiple items", func(t *testing.T) {
		items := []item{
			{"x", "1"},
			{"y", "2"},
		}
		got := formatRows(header, items, func(it item) []string {
			return []string{it.a, it.b}
		})
		want := [][]string{
			{"col1", "col2"},
			{"x", "1"},
			{"y", "2"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
