package slogbuffer

import (
	"iter"
	"log/slog"
	"sync"
)

type record struct {
	slog.Record
	attrs  []slog.Attr
	groups []string
}

// buffer is a structure that stores provided values and allows iteration and cleaning entire buffer.
// If it is bound by maximum number of elements, oldest elements are overwritten when new ones
// are added. Otherwise, it grows without limit.
type buffer[T any] struct {
	// store is actual storage of elements
	store []T
	// flag indicating if storage should be bound to max number of elements or unlimited in size
	bound bool
	// for bound use case, this is start index
	startIndex int

	lock sync.Mutex
}

// Add adds new element to the buffer.
// If buffer is bound and capacity is reached, this will cause the oldest element to be
// removed to make space for new one.
func (b *buffer[T]) Add(element T) {
	b.lock.Lock()
	defer b.lock.Unlock()

	// if not bound of there is still capacity, just append element
	if !b.bound || cap(b.store) > len(b.store) {
		b.store = append(b.store, element)
		return
	}

	// we are at capacity, so overwrite the oldest entry by storing new entry
	// at current start and move current start to next element, being
	// careful to wrap if we exceed slice size
	b.store[b.startIndex] = element

	newStart := (b.startIndex + 1) % cap(b.store)
	b.startIndex = newStart
}

// iterators implementation

// All is two-value iterator (index and value) over the buffer.
// It iterates over all values in buffer (which might not be all values
// that were added to buffer, since oldest values are dropped in case capacity
// is reached).
func (b *buffer[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if b == nil {
			return
		}
		b.lock.Lock()
		defer b.lock.Unlock()

		// it does not matter if storage is bound or not, this implementation of iteration
		// works the same
		maxCap := cap(b.store)
		for i := range len(b.store) {
			ix := (b.startIndex + i) % maxCap
			if !yield(i, b.store[ix]) {
				return
			}
		}
	}
}

// Values is single value iterator (just over values).
// It iterates over all values in buffer (which might not be all values
// that were added to buffer, since oldest values are dropped in case capacity
// is reached)
func (b *buffer[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, el := range b.All() {
			if !yield(el) {
				return
			}
		}
	}
}

// Do runs provided function once for each element in buffer in same order element were added.
func (b *buffer[T]) Do(f func(el T)) {
	for el := range b.Values() {
		f(el)
	}
}

// Clear removes all elements from the buffer.
func (b *buffer[T]) Clear() {
	if b == nil {
		return
	}
	b.lock.Lock()
	defer b.lock.Unlock()
	b.store = make([]T, 0, cap(b.store))

}

// Len returns current number of elements in buffer.
func (b *buffer[T]) Len() int { return len(b.store) }

// IsFull returns flag indicating if buffer is full. Unbound buffer is never full.
func (b *buffer[T]) IsFull() bool {
	if !b.bound {
		return false
	}
	return len(b.store) == cap(b.store)
}

// newBuffer returns instance of a buffer.
// If maxElements is zero or lower, buffer is unbound, meaning there is not upper limit
// in number of elements (or memory) it can hold. If it is bound, maximum number of
// elements is enforced and if capacity is reached, oldest added elements are removed first
// to make space for new elements.
func newBuffer[T any](maxElements int) *buffer[T] {
	if maxElements > 0 {
		return &buffer[T]{
			store: make([]T, 0, maxElements),
			bound: true,
		}
	}

	// unbound buffer case
	return &buffer[T]{
		// cap 16 is arbitrary, just to avoid allocating and copying elements
		// for small buffers
		store: make([]T, 0, 16),
	}
}
