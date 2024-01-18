package balancer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"sync"
	"time"
)

var ErrNoUpstream = errors.New("no upstream")

type ClientDialer[T io.Closer] interface {
	Dial(ctx context.Context) (T, error) // connect client
	IsConnectionError(err error) bool    // return if error is connection error and client should be removed
	GetLimitRps() uint32                 // limit request per second
}

type clientRecord[T io.Closer] struct {
	dialer    ClientDialer[T]
	idx       int
	connected bool
	bucket    int64
	client    T
}

type Observations[T io.Closer] struct {
	OnConnectionStatusChange func(client T, connected bool)
}

// load balance between multiple clients
type ClientBalancer[T io.Closer] struct {
	clients      []*clientRecord[T]
	connected    []*clientRecord[T]
	mu           sync.Mutex
	lastUpdated  time.Time
	index        int
	log          *slog.Logger
	observations *Observations[T]
}

func NewBalancerWLog[T io.Closer](clients []ClientDialer[T], observations *Observations[T], log *slog.Logger) *ClientBalancer[T] {
	var clientRecords []*clientRecord[T]
	for i, client := range clients {
		clientRecords = append(clientRecords, &clientRecord[T]{dialer: client, idx: i, bucket: int64(client.GetLimitRps())})
	}
	return &ClientBalancer[T]{
		clients:      clientRecords,
		log:          log,
		observations: observations,
	}
}

func NewBalancer[T io.Closer](clients []ClientDialer[T]) *ClientBalancer[T] {
	return NewBalancerWLog[T](clients, nil, slog.Default())
}

func (c *ClientBalancer[T]) Start() *ClientBalancer[T] {
	c.lastUpdated = time.Now().Truncate(1 * time.Second)
	c.connectClients()
	go c.reconnectLoop(2 * time.Second)
	return c
}

func (c *ClientBalancer[T]) reconnectLoop(interval time.Duration) {
	t := time.NewTimer(interval)

	for {
		select {
		case <-t.C:
			c.connectClients()
			t.Reset(interval)
		}
	}
}

func (c *ClientBalancer[T]) connectClients() {
	if (len(c.clients)) == len(c.connected) {
		return
	}
	newConnected := make([]*clientRecord[T], 0)
	var err error
	for _, client := range c.clients {
		if !client.connected {
			client.client, err = client.dialer.Dial(context.Background())
			if err != nil {
				c.log.Warn("failed to connect upstream", "error", err)
				continue
			}
			newConnected = append(newConnected, client)
			client.connected = true
			if c.observations != nil && c.observations.OnConnectionStatusChange != nil {
				c.observations.OnConnectionStatusChange(client.client, true)
			}
		} else {
			newConnected = append(newConnected, client)
		}
	}
	c.mu.Lock()
	c.connected = newConnected
	c.mu.Unlock()
}

func (c *ClientBalancer[T]) markDisconnected(client *clientRecord[T]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, v := range c.connected {
		if v.idx == client.idx {
			c.connected = append(c.connected[:i], c.connected[i+1:]...)
			v.connected = false
			if c.observations != nil && c.observations.OnConnectionStatusChange != nil {
				c.observations.OnConnectionStatusChange(v.client, false)
			}
			return
		}
	}
}

/*
Find first client from last index with available limit or wait on client with lowest delay
*/
func (c *ClientBalancer[T]) selectClient(ctx context.Context) *clientRecord[T] {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := len(c.connected)

	if c.index >= l {
		c.index = 0
	}
	now := time.Now()

	// refill buckets
	if now.Sub(c.lastUpdated) >= 1*time.Second {
		for _, client := range c.connected {
			effLimit := int64(client.dialer.GetLimitRps())
			if effLimit <= 0 {
				effLimit = math.MaxInt64
			}
			client.bucket = effLimit
		}
		c.lastUpdated = now.Truncate(1 * time.Second) // pad by 1 second
	}
	for i := 0; i < l; i++ {
		client := c.connected[(c.index+i)%l]
		if client.bucket > 0 {
			c.index = (c.index + i + 1) % l

			if client.bucket > 0 {
				client.bucket--
			}
			return client
		} else {
			// find client with max bucket
			maxVal := int64(0)
			maxIdx := -1
			for j := 0; j < l; j++ {
				if c.connected[j].bucket > maxVal {
					maxVal = c.connected[j].bucket
					maxIdx = j
				}
			}
			if maxIdx >= 0 {
				client := c.connected[maxIdx]
				client.bucket--
				return client
			}
		}
	}

	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}

// find available client and call op with it
// if no client avaialble return ErrNoUpstream
func (c *ClientBalancer[T]) Call(ctx context.Context, op func(ctx context.Context, client T) error) error {
	client := c.selectClient(ctx)
	if client == nil {
		return ErrNoUpstream
	}

	err := op(ctx, client.client)
	if client.dialer.IsConnectionError(err) {
		c.markDisconnected(client)
	}
	return err
}

// same as Call but wait for available client
// use context timeout to limit wait time
func (c *ClientBalancer[T]) CallW(ctx context.Context, op func(ctx context.Context, client T) error) error {
	var client *clientRecord[T]
	for {
		client = c.selectClient(ctx)
		if client == nil {
			c.log.Debug("no upstream available")
			sleepContext(ctx, 1*time.Second)
			if ctx.Err() != nil {
				return ctx.Err()
			}
		} else {
			break
		}
	}

	err := op(ctx, client.client)
	if client.dialer.IsConnectionError(err) {
		c.markDisconnected(client)
	}
	return err
}

func (c *ClientBalancer[T]) CallEveryUpstream(ctx context.Context, op func(ctx context.Context, client T) error) error {
	var err error
	for _, client := range c.clients {
		if !client.connected {
			continue
		}
		err = op(ctx, client.client)
	}
	return err
}

func (c *ClientBalancer[T]) Close() error {
	for _, client := range c.clients {
		if client.connected {
			err := client.client.Close()
			if err != nil {
				slog.Error("failed to close client", "error", err)
			}
		}
	}
	return nil
}
