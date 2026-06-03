package main

import "testing"

func TestKebabConversion(t *testing.T) {
	tests := []struct {
		field string
		want  string
	}{
		{"Name", "name"},
		{"IgnoreCase", "ignore-case"},
		{"MaxCount", "max-count"},

		// Lowercase to digit boundary.
		{"Port8080", "port-8080"},
		{"Retry3xx", "retry-3-xx"},

		// Digit to uppercase boundary.
		{"Base64Encoded", "base-64-encoded"},
		{"Port8080HTTP", "port-8080-http"},

		// Acronym followed by word.
		{"HTTPServer", "http-server"},
		{"HTTPSPort", "https-port"},
		{"ACLEnabled", "acl-enabled"},

		// Trailing acronym.
		{"ID", "id"},
		{"UserID", "user-id"},

		// Uppercase-digit runs stay together.
		{"SHA256Sum", "sha256-sum"},
		{"TLS13Only", "tls13-only"},
		{"H2C", "h2-c"},
	}

	for _, test := range tests {
		t.Run(test.field, func(t *testing.T) {
			cmd := &Command{
				Args: []*Arg{{Name: test.field, Type: "string"}},
			}

			if err := cmd.Process(); err != nil {
				t.Fatal(err)
			}

			if got := cmd.Args[0].Flag; got != test.want {
				t.Errorf("kebab(%s) = %q, want %q", test.field, got, test.want)
			}
		})
	}
}
