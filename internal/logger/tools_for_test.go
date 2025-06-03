package logger_test

import (
	"testing"
)

// assertEqual checks if two values of a comparable type are equal.
func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// assertSlicesEqual checks if two slices of a comparable type are equal.
func assertSlicesEqual[T comparable](t *testing.T, expected, actual []T) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}

	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("expected %v, got %v", expected, actual)
		}
	}
}

// assertNoError fails the test if err is not nil, indicating an unexpected error occurred.
func assertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// assertErrorMessageEqual checks if the error message matches the expected string.
func assertErrorMessageEqual(t *testing.T, err error, expected string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if err.Error() != expected {
		t.Errorf("expected error message: %q, got: %q", expected, err.Error())
	}
}

// assertNotNil checks if the provided object is not nil.
func assertNotNil(t *testing.T, obj any) {
	t.Helper()

	if obj == nil {
		t.Fatal("expected non-nil object, got nil")
	}
}
