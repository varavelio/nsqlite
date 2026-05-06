package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKvToArgs(t *testing.T) {
	t.Run("NoArgs", func(t *testing.T) {
		result := kvToArgs()
		assert.Equal(t, []any{}, result)
	})

	t.Run("OneArg", func(t *testing.T) {
		kv := KV{"key": "value"}
		result := kvToArgs(kv)
		assert.Equal(t, []any{"key", "value"}, result)
	})

	t.Run("MultipleArgs", func(t *testing.T) {
		kv1 := KV{"key1": "value1", "key2": "value2"}
		kv2 := KV{"key3": "value3"}
		result := kvToArgs(kv1, kv2)
		assert.Equal(t, []any{"key1", "value1", "key2", "value2"}, result)
	})

	t.Run("PickOnlyFirst", func(t *testing.T) {
		kv1 := KV{"key1": "value1"}
		kv2 := KV{"key2": "value2"}
		result := kvToArgs(kv1, kv2)
		assert.Equal(t, []any{"key1", "value1"}, result)
	})

	t.Run("Order", func(t *testing.T) {
		kv := KV{"z": "value1", "a": "value2"}
		result := kvToArgs(kv)
		assert.Equal(t, []any{"a", "value2", "z", "value1"}, result)
	})
}

func TestKvToArgsNs(t *testing.T) {
	t.Run("NoArgs", func(t *testing.T) {
		result := kvToArgsNs("namespace")
		assert.Equal(t, []any{"ns", "namespace"}, result)
	})

	t.Run("OneArg", func(t *testing.T) {
		kv := KV{"key": "value"}
		result := kvToArgsNs("namespace", kv)
		assert.Equal(t, []any{"ns", "namespace", "key", "value"}, result)
	})

	t.Run("MultipleArgs", func(t *testing.T) {
		kv1 := KV{"key1": "value1", "key2": "value2"}
		kv2 := KV{"key3": "value3"}
		result := kvToArgsNs("namespace", kv1, kv2)
		assert.Equal(t, []any{"ns", "namespace", "key1", "value1", "key2", "value2"}, result)
	})

	t.Run("PickOnlyFirst", func(t *testing.T) {
		kv1 := KV{"key1": "value1"}
		kv2 := KV{"key2": "value2"}
		result := kvToArgsNs("namespace", kv1, kv2)
		assert.Equal(t, []any{"ns", "namespace", "key1", "value1"}, result)
	})

	t.Run("Order", func(t *testing.T) {
		kv := KV{"z": "value1", "a": "value2"}
		result := kvToArgsNs("namespace", kv)
		assert.Equal(t, []any{"ns", "namespace", "a", "value2", "z", "value1"}, result)
	})
}
