package syncutil

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomic(t *testing.T) {
	t.Run("LoadEmpty", func(t *testing.T) {
		atomic := &Atomic[string]{}

		// Load should return the zero value for strings (empty string)
		assert.Equal(t, "", atomic.Load(), "Expected empty string for uninitialized Atomic[string]")
	})

	t.Run("StoreAndLoad_String", func(t *testing.T) {
		atomic := &Atomic[string]{}

		// Store a string and then load it
		atomic.Store("test value")
		assert.Equal(t, "test value", atomic.Load(), "Stored value should match the loaded value")
	})

	t.Run("StoreAndLoad_Int", func(t *testing.T) {
		atomic := &Atomic[int]{}

		// Store an integer and then load it
		atomic.Store(42)
		assert.Equal(t, 42, atomic.Load(), "Stored value should match the loaded value")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		atomic := &Atomic[string]{}
		atomic.Store("initial value")

		const goroutines = 100
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(i int) {
				defer wg.Done()
				// Concurrently store new values
				atomic.Store("value " + string(rune(i)))
			}(i)
		}

		wg.Wait()

		// Load the final value (this may vary, but it should not crash)
		result := atomic.Load()
		assert.NotEmpty(t, result, "Concurrent access should not leave the value empty")
	})

	t.Run("TypeSafety_String", func(t *testing.T) {
		atomic := &Atomic[string]{}

		// Store and load a valid string
		atomic.Store("valid string")
		assert.Equal(t, "valid string", atomic.Load(), "Stored string should be retrievable")
	})

	t.Run("TypeSafety_Int", func(t *testing.T) {
		atomic := &Atomic[int]{}

		// Store and load an integer
		atomic.Store(12345)
		assert.Equal(t, 12345, atomic.Load(), "Stored int should be retrievable")
	})

	t.Run("NewAtomic_String", func(t *testing.T) {
		// Test constructor for strings
		atomic := NewAtomic("initial value")
		assert.Equal(
			t,
			"initial value",
			atomic.Load(),
			"Constructor should initialize the string correctly",
		)

		// Test updating value
		atomic.Store("updated value")
		assert.Equal(
			t,
			"updated value",
			atomic.Load(),
			"Updated value should match the loaded value",
		)
	})

	t.Run("NewAtomic_Int", func(t *testing.T) {
		// Test constructor for integers
		atomic := NewAtomic(42)
		assert.Equal(t, 42, atomic.Load(), "Constructor should initialize the integer correctly")

		// Test updating value
		atomic.Store(100)
		assert.Equal(t, 100, atomic.Load(), "Updated value should match the loaded value")
	})
}
