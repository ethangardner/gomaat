package cli

import "testing"

func TestWhoKnowsCmdRunE(t *testing.T) {
	const header = "author,added,ownership\n"
	tests := []struct {
		name    string
		path    string
		after   string
		want    string
		wantErr string
	}{
		{name: "file inside window", path: "foo.go", after: "2024-01-01", want: header + "Jane Doe,1,100.00\n"},
		{name: "file before window", path: "foo.go", after: "2024-01-02", want: header},
		{name: "dot-slash file", path: "./foo.go", after: "2024-01-01", want: header + "Jane Doe,1,100.00\n"},
		{name: "repository root", path: ".", after: "2024-01-01", want: header + "Jane Doe,1,100.00\n"},
		{name: "bad --after", path: "foo.go", after: "yesterday", wantErr: "--after"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runOnFixture(t, newWhoKnowsCmd(), map[string]string{"after": tt.after}, tt.path)
			if tt.wantErr != "" {
				assertErrContains(t, err, tt.wantErr)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
