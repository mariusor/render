package render

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

var ctx = context.Background()

func TestLockConfig(t *testing.T) {
	mutex := reflect.TypeOf(&sync.RWMutex{}).Kind()
	empty := reflect.TypeOf(&emptyLock{}).Kind()

	r1 := New(Options{
		IsDevelopment: true,
		UseMutexLock:  false,
	})
	expect(t, reflect.TypeOf(r1.lock).Kind(), mutex)

	r2 := New(Options{
		IsDevelopment: true,
		UseMutexLock:  true,
	})
	expect(t, reflect.TypeOf(r2.lock).Kind(), mutex)

	r3 := New(Options{
		IsDevelopment: false,
		UseMutexLock:  true,
	})
	expect(t, reflect.TypeOf(r3.lock).Kind(), mutex)

	r4 := New(Options{
		IsDevelopment: false,
		UseMutexLock:  false,
	})
	expect(t, reflect.TypeOf(r4.lock).Kind(), empty)
}

func BenchmarkHTML(b *testing.B) {
	render := New(Options{
		Directory: "testdata/basic",
	})

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = render.HTML(w, http.StatusOK, "hello", "gophers")
	})
	req, _ := http.NewRequestWithContext(ctx, "GET", "/foo", nil)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}

/* Test Helper */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected ||%#v|| (type %v) - Got ||%#v|| (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func expectNil(t *testing.T, a interface{}) {
	if a != nil {
		t.Errorf("Expected ||nil|| - Got ||%#v|| (type %v)", a, reflect.TypeOf(a))
	}
}

func expectNotNil(t *testing.T, a interface{}) {
	if a == nil {
		t.Errorf("Expected ||not nil|| - Got ||nil|| (type %v)", reflect.TypeOf(a))
	}
}
