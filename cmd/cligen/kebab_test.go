package main

import "testing"

func TestKebabConversion(t *testing.T) {
	tests := []struct {
		field string
		want  string
	}{
		// Single and multi-word.
		{"Name", "name"},
		{"Verbose", "verbose"},
		{"DryRun", "dry-run"},
		{"IgnoreCase", "ignore-case"},
		{"MaxCount", "max-count"},
		{"OutputDir", "output-dir"},

		// Leading acronym.
		{"HTTPPort", "http-port"},
		{"HTTPSPort", "https-port"},
		{"TLSCert", "tls-cert"},
		{"ACLEnabled", "acl-enabled"},
		{"CPULimit", "cpu-limit"},
		{"IAMRole", "iam-role"},

		// Short and trailing acronyms.
		{"UserID", "user-id"},
		{"UID", "uid"},
		{"EnableTLS", "enable-tls"},

		// Acronym with trailing digit.
		{"HTTP2", "http2"},
		{"UseHTTP2", "use-http2"},
		{"S3", "s3"},
		{"S3Bucket", "s3-bucket"},

		// Acronym-digit runs.
		{"SHA256Sum", "sha256-sum"},

		// Digit stays glued to preceding word.
		{"UseBase64", "use-base64"},
		{"UseLog2", "use-log2"},
		{"UseAtan2", "use-atan2"},
		{"UseBzip2", "use-bzip2"},

		// Unicode letters.
		{"DéjàVu", "déjà-vu"},
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
