package tsengine

import (
	"testing"
	"time"
)

func TestTurboCacheMemoryDriver_AllOps(t *testing.T) {
	d := newTurboCacheMemoryDriver()

	// Test set/get
	d.set("foo", "bar", 2)
	v, ok := d.get("foo")
	if !ok || v != "bar" {
		t.Errorf("expected 'bar', got %v", v)
	}

	// Test has
	if !d.has("foo") {
		t.Error("expected has(foo) to be true")
	}

	// Test del
	d.del("foo")
	_, ok = d.get("foo")
	if ok {
		t.Error("expected get(foo) after del to be false")
	}

	// Test flush
	d.set("a", "1", 0)
	d.set("b", "2", 0)
	d.flush()
	if d.has("a") || d.has("b") {
		t.Error("expected cache to be empty after flush")
	}

	// Test TTL expiry
	d.set("exp", "gone", 1)
	time.Sleep(2 * time.Second)
	_, ok = d.get("exp")
	if ok {
		t.Error("expected expired key to be gone")
	}

	// Test complex object storage
	d.set("obj", map[string]any{"time": 123, "data": "baz"}, 0)
	v, ok = d.get("obj")
	if !ok {
		t.Error("expected object to be stored and retrieved")
	}
	if obj, ok := v.(map[string]any); ok {
		if obj["data"] != "baz" || obj["time"] != 123 {
			t.Errorf("expected object with correct properties, got %v", obj)
		}
	} else {
		t.Errorf("expected map[string]any, got %T", v)
	}
}
