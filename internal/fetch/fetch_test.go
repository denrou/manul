package fetch

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "bare domain", input: "example.com", want: "https://example.com/llms.txt"},
		{name: "domain with slash", input: "example.com/", want: "https://example.com/llms.txt"},
		{name: "full URL kept", input: "https://example.com/docs/page.md", want: "https://example.com/docs/page.md"},
		{name: "http kept", input: "http://example.com/llms.txt", want: "http://example.com/llms.txt"},
		{name: "surrounding spaces", input: "  example.com  ", want: "https://example.com/llms.txt"},
		{name: "empty", input: "", wantErr: true},
		{name: "unsupported scheme", input: "ftp://example.com", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Normalize(%q) = %q, want error", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Normalize(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
