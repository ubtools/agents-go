package balancer

// balancer test

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testClient struct {
	Name      string
	Connected bool
}

var connErr1 = errors.New("connection error 1")
var connErr2 = errors.New("connection error 2")

func (c *testClient) Dial(ctx context.Context) (testc, error) {
	if c.Connected {
		return testc(c.Name), nil
	} else {
		return "", connErr1
	}
}

func (c *testClient) IsConnectionError(err error) bool {
	return err == connErr1 || err == connErr2
}

func (c *testClient) GetLimitRps() uint32 {
	return 2
}

type testc string

func (t testc) Close() error {
	return nil
}

func TestSingle(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)

	assert.Nil(t, err)

	err = b.Call(ctx, testFunc)
	assert.Equal(t, ErrNoUpstream, err)
}

func TestRefill(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	assert.Equal(t, ErrNoUpstream, err)

	time.Sleep(1 * time.Second)

	err = b.Call(ctx, testFunc)
	assert.Nil(t, err)
}

func TestCallW(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	err = b.CallW(ctx, testFunc)
	assert.Nil(t, err)
}

func TestCallWTimeout(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	ctxTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	err = b.CallW(ctxTimeout, testFunc)
	assert.Equal(t, err, context.DeadlineExceeded)
}

func TestRoundRobin(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	c2 := &testClient{Name: "client2", Connected: false}
	c3 := &testClient{Name: "client3", Connected: true}
	c4 := &testClient{Name: "client4", Connected: false}
	b := NewBalancer([]ClientDialer[testc]{c1, c2, c3, c4}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)
	err = b.Call(ctx, testFunc)

	assert.Nil(t, err)
	assert.Equal(t, []testc{testc("client1"), testc("client3"), testc("client1"), testc("client3")}, vals)
}

func TestNoConnection(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: false}
	c2 := &testClient{Name: "client2", Connected: false}
	c3 := &testClient{Name: "client3", Connected: false}
	c4 := &testClient{Name: "client4", Connected: false}
	b := NewBalancer([]ClientDialer[testc]{c1, c2, c3, c4}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return nil
	}
	err = b.Call(ctx, testFunc)

	assert.Equal(t, ErrNoUpstream, err)
}

func TestDisconnected(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1}).Start()
	ctx := context.Background()
	vals := []testc{}
	var err error
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		vals = append(vals, client)
		return connErr2
	}
	err = b.Call(ctx, testFunc)

	assert.Equal(t, connErr2, err)

	err = b.Call(ctx, testFunc)
	assert.Equal(t, ErrNoUpstream, err)
}

func array_sorted_equal(a, b []testc) bool {
	if len(a) != len(b) {
		return false
	}

	a_copy := make([]string, len(a))
	b_copy := make([]string, len(b))

	for i, v := range a {
		a_copy[i] = string(v)
	}
	for i, v := range b {
		b_copy[i] = string(v)
	}

	sort.Strings(a_copy)
	sort.Strings(b_copy)

	return reflect.DeepEqual(a_copy, b_copy)
}

func TestMultipleGoroutine(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	c2 := &testClient{Name: "client2", Connected: true}
	c3 := &testClient{Name: "client3", Connected: true}
	c4 := &testClient{Name: "client4", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1, c2, c3, c4}).Start()
	ctx := context.Background()
	vals := []testc{}
	mu := &sync.Mutex{}
	var wg sync.WaitGroup
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		mu.Lock()
		defer mu.Unlock()
		wg.Done()
		vals = append(vals, client)
		return nil
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go b.Call(ctx, testFunc)
	}

	wg.Wait()

	assert.Len(t, vals, 8)
	assert.True(t, array_sorted_equal(vals, []testc{testc("client1"), testc("client1"), testc("client2"), testc("client2"), testc("client3"), testc("client3"), testc("client4"), testc("client4")}), "all clients called twice")
}

func TestMultipleGoroutineWithDisconnect(t *testing.T) {
	c1 := &testClient{Name: "client1", Connected: true}
	c2 := &testClient{Name: "client2", Connected: true}
	c3 := &testClient{Name: "client3", Connected: true}
	c4 := &testClient{Name: "client4", Connected: true}
	b := NewBalancer([]ClientDialer[testc]{c1, c2, c3, c4}).Start()
	ctx := context.Background()
	vals := []testc{}
	mu := &sync.Mutex{}
	var wg sync.WaitGroup
	testFunc := func(ctx context.Context, client testc) error {
		t.Logf("call %s", client)
		mu.Lock()
		defer mu.Unlock()
		wg.Done()
		vals = append(vals, client)
		return nil
	}
	for i := 0; i < 7; i++ {
		wg.Add(1)
		tf := testFunc
		if i == 1 {
			tf = func(ctx context.Context, client testc) error {
				t.Logf("call %s", client)
				wg.Done()
				return connErr2
			}
		}
		go b.Call(ctx, tf)
	}

	wg.Wait()

	assert.Len(t, vals, 6)
}

func BenchmarkBalancer(b *testing.B) {
	b.Log("R")
	c1 := &testClient{Name: "client1", Connected: true}
	c2 := &testClient{Name: "client2", Connected: false}
	c3 := &testClient{Name: "client3", Connected: true}
	c4 := &testClient{Name: "client4", Connected: true}
	balancer := NewBalancer([]ClientDialer[testc]{c1, c2, c3, c4}).Start()
	ctx := context.Background()
	testFunc := func(ctx context.Context, client testc) error {
		return nil
	}
	for i := 0; i < b.N; i++ {
		b.Log(i)
		balancer.CallW(ctx, testFunc)
	}
}
