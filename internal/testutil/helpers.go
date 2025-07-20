package testutil

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// CreateTestDB creates an in-memory SQLite database for testing
func CreateTestDB(t *testing.T) (*sql.DB, func()) {
	// Use shared cache in-memory database for concurrent access
	db, err := sql.Open("sqlite", ":memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Set connection pool to use single connection to avoid race conditions
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Create schema
	schema := `
		CREATE TABLE artists (
			mbid TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			verified_json TEXT DEFAULT '{}',
			album_count INTEGER DEFAULT 0,
			years_active TEXT DEFAULT '',
			description TEXT DEFAULT '',
			genres_json TEXT DEFAULT '[]',
			country TEXT DEFAULT '',
			image_url TEXT DEFAULT '',
			external_urls_json TEXT DEFAULT '{}',
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			cache_expiry DATETIME NOT NULL
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db, func() { db.Close() }
}

// AssertNoError is a test helper to check that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a test helper to check that an error occurred
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}

// AssertEqual is a generic equality assertion helper
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEqual is a generic inequality assertion helper
func AssertNotEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected == actual {
		t.Errorf("Expected %v to not equal %v", expected, actual)
	}
}

// AssertTrue checks that a condition is true
func AssertTrue(t *testing.T, condition bool) {
	t.Helper()
	if !condition {
		t.Error("Expected condition to be true")
	}
}

// AssertFalse checks that a condition is false
func AssertFalse(t *testing.T, condition bool) {
	t.Helper()
	if condition {
		t.Error("Expected condition to be false")
	}
}

// AssertNotNil checks that a value is not nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Error("Expected value to not be nil")
	}
}

// AssertNil checks that a value is nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Errorf("Expected value to be nil, got: %v", value)
	}
}

// AssertGreater checks that actual > expected
func AssertGreater[T interface{ ~int | ~int64 | ~float64 }](t *testing.T, actual, expected T) {
	t.Helper()
	if actual <= expected {
		t.Errorf("Expected %v > %v", actual, expected)
	}
}

// AssertGreaterOrEqual checks that actual >= expected
func AssertGreaterOrEqual[T interface{ ~int | ~int64 | ~float64 }](t *testing.T, actual, expected T) {
	t.Helper()
	if actual < expected {
		t.Errorf("Expected %v >= %v", actual, expected)
	}
}

// AssertLess checks that actual < expected
func AssertLess[T interface{ ~int | ~int64 | ~float64 }](t *testing.T, actual, expected T) {
	t.Helper()
	if actual >= expected {
		t.Errorf("Expected %v < %v", actual, expected)
	}
}

// AssertSliceEqual checks that two slices are equal
func AssertSliceEqual[T comparable](t *testing.T, expected, actual []T) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Slice length mismatch: expected %d, got %d", len(expected), len(actual))
		return
	}
	for i, exp := range expected {
		if actual[i] != exp {
			t.Errorf("Slice element mismatch at index %d: expected %v, got %v", i, exp, actual[i])
		}
	}
}

// AssertSliceContains checks that a slice contains a specific element
func AssertSliceContains[T comparable](t *testing.T, slice []T, element T) {
	t.Helper()
	for _, item := range slice {
		if item == element {
			return
		}
	}
	t.Errorf("Expected slice to contain %v", element)
}

// AssertMapEqual checks that two maps are equal
func AssertMapEqual[K, V comparable](t *testing.T, expected, actual map[K]V) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Map length mismatch: expected %d, got %d", len(expected), len(actual))
		return
	}
	for key, expectedValue := range expected {
		if actualValue, exists := actual[key]; !exists {
			t.Errorf("Missing key %v in actual map", key)
		} else if actualValue != expectedValue {
			t.Errorf("Value mismatch for key %v: expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}
