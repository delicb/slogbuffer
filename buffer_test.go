package slogbuffer

import (
	"testing"
)

func expectBufferContent[T comparable](t *testing.T, b *buffer[T], expect []T) {
	t.Helper()
	for i, el := range b.All() {
		if el != expect[i] {
			t.Fatalf("element at index '%v' unexpected, got %v, expected %v", i, el, expect[i])
		}
	}
}

func TestUnboundBuffer(t *testing.T) {
	b := newBuffer[int](0)
	b.Add(1)
	b.Add(2)

	if b.Len() != 2 {
		t.Fatalf("buffer has length %d, expected 2", b.Len())
	}
	if b.IsFull() {
		t.Fatalf("unbound buffer should never be full")
	}
	expectBufferContent(t, b, []int{1, 2})

	b.Add(3)
	expectBufferContent(t, b, []int{1, 2, 3})
	if b.Len() != 3 {
		t.Fatalf("buffer has length %d, expected 2", b.Len())
	}
	if b.IsFull() {
		t.Fatalf("unbound buffer should never be full")
	}

	b.Clear()

	if b.Len() != 0 {
		t.Fatalf("buffer has length %d, expected 0", b.Len())
	}
}

func TestBuffer_Do(t *testing.T) {
	b := newBuffer[int](0)
	b.Add(2)
	b.Add(3)
	b.Add(4)

	sum := 0
	b.Do(func(i int) {
		sum += i
	})
}

func TestBoundBuffer(t *testing.T) {
	b := newBuffer[int](3)
	b.Add(1)
	b.Add(2)

	if b.Len() != 2 {
		t.Fatalf("buffer has length %d, expected 2", b.Len())
	}
	if b.IsFull() {
		t.Fatalf("buffer full after just two elements")
	}

	b.Add(3)

	if !b.IsFull() {
		t.Fatalf("buffer shold be full")
	}

	expectBufferContent(t, b, []int{1, 2, 3})

}

func TestBoundBufferOverCapacity(t *testing.T) {
	b := newBuffer[int](3)

	for i := range 5 {
		b.Add(i)
	}

	expectBufferContent(t, b, []int{2, 3, 4})

	if !b.IsFull() {
		t.Fatalf("buffer should be full")
	}
	if b.Len() != 3 {
		t.Fatalf("buffer has length %d, expected 3", b.Len())
	}
}
