package scanner_compressfile

import "testing"

func TestIsArchiveFile(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"file.zip", true},
		{"file.7z", true},
		{"file.rar", true},
		{"file.ZIP", false},
		{"file.7Z", false},
		{"file.RAR", false},
		{"file.txt", false},
		{"file", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsArchiveFile(tt.input); got != tt.want {
				t.Errorf("IsArchiveFile(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
