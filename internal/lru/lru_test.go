package lru

import "testing"

func TestLRUBasicEviction(t *testing.T) {
	c := New(2)

	c.Add("a", 1)
	c.Add("b", 2)

	if v, ok := c.Get("a"); !ok || v.(int) != 1 {
		t.Fatalf("expected Get(a)=1, got %v (ok=%v)", v, ok)
	}

	c.Add("c", 3)

	if _, ok := c.Get("b"); ok {
		t.Fatalf("expected b to be evicted")
	}
	if v, ok := c.Get("a"); !ok || v.(int) != 1 {
		t.Fatalf("expected a to stay in cache, got %v (ok=%v)", v, ok)
	}
	if v, ok := c.Get("c"); !ok || v.(int) != 3 {
		t.Fatalf("expected c=3, got %v (ok=%v)", v, ok)
	}
}

func TestLRUClearAndAll(t *testing.T) {
	c := New(3)
	c.Add("a", 1)
	c.Add("b", 2)

	if got := len(c.All()); got != 2 {
		t.Fatalf("expected All() len=2, got %d", got)
	}

	c.Clear()

	if _, ok := c.Get("a"); ok {
		t.Fatalf("expected cache to be empty after Clear")
	}
	if got := len(c.All()); got != 0 {
		t.Fatalf("expected All() len=0 after Clear, got %d", got)
	}
}
