package lru

import (
	"container/list"
	"sync"
)

// Simple thread-safe LRU cache for string keys and interface{} values.
// Minimal, no generics to keep Go1.x compatibility.
type entry struct {
	key   string
	value any
}

type LRU struct {
	mu    sync.Mutex
	ll    *list.List
	cache map[string]*list.Element
	cap   int
}

func New(capacity int) *LRU {
	if capacity <= 0 {
		capacity = 1024
	}
	return &LRU{
		ll:    list.New(),
		cache: make(map[string]*list.Element, capacity),
		cap:   capacity,
	}
}

func (l *LRU) Get(key string) (any, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.cache[key]; ok {
		l.ll.MoveToFront(e)
		return e.Value.(*entry).value, true
	}
	return nil, false
}

func (l *LRU) Add(key string, val interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.cache[key]; ok {
		l.ll.MoveToFront(e)
		e.Value.(*entry).value = val
		return
	}
	e := &entry{key: key, value: val}
	elem := l.ll.PushFront(e)
	l.cache[key] = elem
	if l.ll.Len() > l.cap {
		back := l.ll.Back()
		if back != nil {
			l.ll.Remove(back)
			en := back.Value.(*entry)
			delete(l.cache, en.key)
		}
	}
}

func (l *LRU) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e, ok := l.cache[key]; ok {
		l.ll.Remove(e)
		delete(l.cache, key)
	}
}

func (l *LRU) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ll = list.New()
	l.cache = make(map[string]*list.Element, l.cap)
}

func (l *LRU) All() []any {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []any
	for e := l.ll.Front(); e != nil; e = e.Next() {
		out = append(out, e.Value.(*entry).value)
	}
	return out
}
