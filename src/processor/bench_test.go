package processor_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func action(i int) string {
	return fmt.Sprint(i) + "fldkhglsfdhglfdjghlsd"
}

func op(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func f1(n int) {
	a := make(chan string, 1000)

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	for i := 0; i < n; i++ {
		r := <-a
		r = op(r)
		if r == "" {
			panic("done")
		}
	}
}

func f2(n int) {
	a := make(chan string, 1000)

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	for i := 0; i < n; i++ {
		select {
		case r := <-a:
			r = op(r)
			if r == "" {
				panic("empty")
			}
		case <-context.Background().Done():
			panic("done")
		}
	}
}

func f3(n int) {
	a := make(chan string, 1000)

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	for i := 0; i < n; i++ {
		select {
		case r := <-a:
			r = op(r)
			if r == "" {
				panic("")
			}
		case <-time.After(1 * time.Second):
			panic("")
		case <-context.Background().Done():
			panic("")
		}
	}
}

func f4(n int) {
	a := make(chan string, 1000)
	b := make(chan string, 1000)

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	go func() {
		for i := 0; i < n; i++ {
			r := <-a
			b <- op(r)
		}
	}()

	for i := 0; i < n; i++ {
		select {
		case r := <-b:
			if r == "" {
				panic("emty string")
			}
		case <-time.After(1 * time.Second):
			panic("timeout")
		case <-context.Background().Done():
			panic("done")
		}
	}
}

func f5(n int) {
	a := make(chan string, 1000)
	b := make(chan string, 1000)

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	go func() {
		for {
			r := <-a
			b <- op(r)
		}
	}()
	go func() {
		for {
			r := <-a
			b <- op(r)
		}
	}()

	for i := 0; i < n; i++ {
		select {
		case r := <-b:
			if r == "" {
				panic("emty string")
			}
		case <-time.After(1 * time.Second):
			panic("timeout")
		case <-context.Background().Done():
			panic("done")
		}
	}
}

func BenchmarkF1(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f1(1000000)
	}
}

func BenchmarkF2(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f2(1000000)
	}
}

func BenchmarkF3(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f3(1000000)
	}
}

func BenchmarkF4(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f4(1000000)
	}
}

func BenchmarkF5(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f5(1000000)
	}
}
