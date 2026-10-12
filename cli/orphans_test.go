package cli

import "testing"

func TestOrphansCmdRunE(t *testing.T) {
	const header = "entity,added-lines,date,author\n"
	tests := []struct {
		name  string
		flags map[string]string
		want  string
	}{
		{"at threshold", map[string]string{"min-lines": "1"}, header + "foo.go,1,2024-01-01,Jane Doe\n"},
		{"default threshold", nil, header},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runOnFixture(t, newOrphansCmd(), tt.flags)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
