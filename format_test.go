package main

import "testing"

func TestBytesHuman(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, tt := range tests {
		got := BytesHuman(tt.in)
		if got != tt.want {
			t.Fatalf("BytesHuman(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestCommaAny(t *testing.T) {
	if CommaAny(1234567) != "1,234,567" {
		t.Fatalf("CommaAny(int) unexpected: %s", CommaAny(1234567))
	}
	if CommaAny(int64(9876543210)) != "9,876,543,210" {
		t.Fatalf("CommaAny(int64) unexpected: %s", CommaAny(int64(9876543210)))
	}
	// fallback to fmt.Sprint for other types
	if CommaAny("abc") != "abc" {
		t.Fatalf("CommaAny(string) unexpected: %s", CommaAny("abc"))
	}
}
