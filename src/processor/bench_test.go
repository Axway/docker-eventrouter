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
	for i := 0; i < 10; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return fmt.Sprintf("%x", sum)
}

func f0(n int) {
	for i := 0; i < n; i++ {
		r := action(i)
		r = op(r)
		if r == "" {
			panic("done")
		}
	}
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
	ctx := context.Background()
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
		case <-ctx.Done():
			panic("done")
		}
	}
}

func f3(n int) {
	a := make(chan string, 1000)
	ctx := context.Background()
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
		case <-ctx.Done():
			panic("")
		}
	}
}

func f3_2(n int) {
	a := make(chan string, 1000)
	ctx := context.Background()

	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	for i := 0; i < n; i++ {
		t := time.NewTimer(1 * time.Second)
		select {
		case r := <-a:
			r = op(r)
			if r == "" {
				panic("")
			}
		case <-t.C:
			panic("")
		case <-ctx.Done():
			panic("")
		}
		t.Stop()
	}
}

func f4(n int) {
	a := make(chan string, 1000)
	b := make(chan string, 1000)
	ctx := context.Background()
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
		t := time.NewTimer(1 * time.Second)
		select {
		case r := <-b:
			if r == "" {
				panic("emty string")
			}
		case <-t.C:
			panic("timeout")
		case <-ctx.Done():
			panic("done")
		}
		t.Stop()
	}
}

func f5(n, c int) {
	a := make(chan string, 1000)
	b := make(chan string, 1000)
	ctx := context.Background()
	go func() {
		for i := 0; i < n; i++ {
			a <- action(i)
		}
	}()

	for i := 0; i < c; i++ {
		go func() {
			for {
				r := <-a
				b <- op(r)
			}
		}()
	}

	for i := 0; i < n; i++ {
		t := time.NewTimer(1 * time.Second)
		select {
		case r := <-b:
			if r == "" {
				panic("emty string")
			}
		case <-t.C:
			panic("timeout")
		case <-ctx.Done():
			panic("done")
		}
		t.Stop()
	}
}

func BenchmarkF0_noqueue(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f0(1000000)
	}
}

func BenchmarkF1_simplequeue(b *testing.B) {
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

func BenchmarkF3_1(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f3(1000000)
	}
}

func BenchmarkF3_2(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f3(1000000)
	}
}

func BenchmarkF4_1prod_1cpu(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f4(1000000)
	}
}

func BenchmarkF5_1prod_2cpu(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f5(1000000, 2)
	}
}

func BenchmarkF5_1prod_3cpu(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f5(1000000, 3)
	}
}

func BenchmarkF5_1prod_4cpu(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		f5(1000000, 4)
	}
}
