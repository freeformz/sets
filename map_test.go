package sets

import "testing"

// create similar tests for the other Map types AI!
func TestMapScan(t *testing.T) {
	t.Parallel()

	t.Run("scan nil", func(t *testing.T) {
		s := New[int]()
		s.Add(1)
		s.Add(2)

		err := s.Scan(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Cardinality() != 0 {
			t.Fatalf("expected empty set, got %d elements", s.Cardinality())
		}
	})

	t.Run("scan []byte JSON", func(t *testing.T) {
		s := New[int]()
		jsonData := []byte(`[1,2,3]`)

		err := s.Scan(jsonData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Cardinality() != 3 {
			t.Fatalf("expected 3 elements, got %d", s.Cardinality())
		}

		for _, expected := range []int{1, 2, 3} {
			if !s.Contains(expected) {
				t.Fatalf("expected set to contain %d", expected)
			}
		}
	})

	t.Run("scan string JSON", func(t *testing.T) {
		s := New[string]()
		jsonData := `["a","b","c"]`

		err := s.Scan(jsonData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Cardinality() != 3 {
			t.Fatalf("expected 3 elements, got %d", s.Cardinality())
		}

		for _, expected := range []string{"a", "b", "c"} {
			if !s.Contains(expected) {
				t.Fatalf("expected set to contain %s", expected)
			}
		}
	})

	t.Run("scan empty JSON array", func(t *testing.T) {
		s := New[int]()
		s.Add(1) // add something first

		err := s.Scan([]byte(`[]`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Cardinality() != 0 {
			t.Fatalf("expected empty set, got %d elements", s.Cardinality())
		}
	})

	t.Run("scan invalid JSON", func(t *testing.T) {
		s := New[int]()

		err := s.Scan([]byte(`invalid json`))
		if err == nil {
			t.Fatalf("expected error for invalid JSON")
		}
	})

	t.Run("scan unsupported type", func(t *testing.T) {
		s := New[int]()

		err := s.Scan(123) // int is not supported
		if err == nil {
			t.Fatalf("expected error for unsupported type")
		}

		expectedMsg := "cannot scan set of type int - not []byte or string"
		if err.Error() != expectedMsg {
			t.Fatalf("expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("scan overwrites existing data", func(t *testing.T) {
		s := New[int]()
		s.Add(99)
		s.Add(100)

		err := s.Scan([]byte(`[1,2]`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Cardinality() != 2 {
			t.Fatalf("expected 2 elements, got %d", s.Cardinality())
		}

		if s.Contains(99) || s.Contains(100) {
			t.Fatalf("expected old elements to be cleared")
		}

		if !s.Contains(1) || !s.Contains(2) {
			t.Fatalf("expected new elements to be present")
		}
	})
}
