package ui

import (
	"testing"
)

func TestPropagateANSIState(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI sequences",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "color carries to next line",
			input:    "\x1b[38;2;0;0;255mhello\nworld",
			expected: "\x1b[38;2;0;0;255mhello\n\x1b[38;2;0;0;255mworld",
		},
		{
			name:     "reset stops carry",
			input:    "\x1b[38;2;0;0;255mhello\x1b[0m\nworld",
			expected: "\x1b[38;2;0;0;255mhello\x1b[0m\nworld",
		},
		{
			name:     "reset with short form stops carry",
			input:    "\x1b[38;2;0;0;255mhello\x1b[m\nworld",
			expected: "\x1b[38;2;0;0;255mhello\x1b[m\nworld",
		},
		{
			name:     "multiple colors accumulate",
			input:    "\x1b[38;2;0;0;255m\x1b[48;2;255;0;0mhello\nworld",
			expected: "\x1b[38;2;0;0;255m\x1b[48;2;255;0;0mhello\n\x1b[38;2;0;0;255m\x1b[48;2;255;0;0mworld",
		},
		{
			name:     "color on line 1, carries to lines 2 and 3",
			input:    "\x1b[38;2;0;0;255mline1\nline2\nline3",
			expected: "\x1b[38;2;0;0;255mline1\n\x1b[38;2;0;0;255mline2\n\x1b[38;2;0;0;255mline3",
		},
		{
			name:     "test.ans scenario: line1 has fg+bg, line2 has blue fg, line3 has no escape",
			input:    "\x1b[38;2;0;255;0;40m\x1b[48;2;0;0;0m   \n\x1b[38;2;0;0;255m   HELLO\n   WORLD\n\x1b[0m",
			expected: "\x1b[38;2;0;255;0;40m\x1b[48;2;0;0;0m   \n\x1b[38;2;0;255;0;40m\x1b[48;2;0;0;0m\x1b[38;2;0;0;255m   HELLO\n\x1b[38;2;0;255;0;40m\x1b[48;2;0;0;0m\x1b[38;2;0;0;255m   WORLD\n\x1b[38;2;0;255;0;40m\x1b[48;2;0;0;0m\x1b[38;2;0;0;255m\x1b[0m",
		},
		{
			name:     "color change mid-content",
			input:    "\x1b[31mred\n\x1b[32mgreen\nstill green",
			expected: "\x1b[31mred\n\x1b[31m\x1b[32mgreen\n\x1b[31m\x1b[32mstill green",
		},
		{
			name:     "empty lines preserve carry",
			input:    "\x1b[34mblue\n\nstill blue",
			expected: "\x1b[34mblue\n\x1b[34m\n\x1b[34mstill blue",
		},
		{
			name:     "CRLF normalization",
			input:    "\x1b[34mblue\r\nworld",
			expected: "\x1b[34mblue\n\x1b[34mworld",
		},
		{
			name:     "single line unchanged",
			input:    "\x1b[34mhello world",
			expected: "\x1b[34mhello world",
		},
		{
			name:     "no carry needed when empty",
			input:    "plain\ntext",
			expected: "plain\ntext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := propagateANSIState(tt.input)
			if got != tt.expected {
				t.Errorf("propagateANSIState() =\n  %q\nwant:\n  %q", got, tt.expected)
			}
		})
	}
}
