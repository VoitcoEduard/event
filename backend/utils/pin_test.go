package utils

import "testing"

func TestNewEventPIN(t *testing.T) {
	for i := 0; i < 100; i++ {
		pin, err := NewEventPIN()
		if err != nil {
			t.Fatal(err)
		}
		if len(pin) != 6 {
			t.Fatalf("PIN %q has length %d", pin, len(pin))
		}
		for _, digit := range pin {
			if digit < '0' || digit > '9' {
				t.Fatalf("PIN %q contains non-digit", pin)
			}
		}
	}
}
