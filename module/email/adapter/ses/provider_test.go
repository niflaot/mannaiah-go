package ses

import "testing"

// TestSanitizeSESTagValue validates SES tag value sanitization behavior.
func TestSanitizeSESTagValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "replaces colons used by campaign idempotency keys",
			input: "test:9da7cd63-1fb4-4ceb-9876-8410780e06f8:a4f580fd-23d3-4193-a408-b884e40e58fe",
			want:  "test_9da7cd63-1fb4-4ceb-9876-8410780e06f8_a4f580fd-23d3-4193-a408-b884e40e58fe",
		},
		{
			name:  "keeps allowed characters",
			input: "abc_123-xyz.test@email",
			want:  "abc_123-xyz.test@email",
		},
		{
			name:  "trims outer spaces",
			input: "  abc  ",
			want:  "abc",
		},
		{
			name:  "replaces unsupported characters",
			input: "abc/def+ghi:jkl",
			want:  "abc_def_ghi_jkl",
		},
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := sanitizeSESTagValue(testCase.input)
			if got != testCase.want {
				t.Fatalf("sanitizeSESTagValue(%q) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}
