package client

import (
	"context"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/ubtr/ubt-go/commons"
	"github.com/ubtr/ubt-go/commons/balancer"
	"github.com/ubtr/ubt-go/commons/jsonrpc"
)

type ClientConfig struct {
	Url string
	//options  []rpc.ClientOption
	LimitRps uint
	Labels   []any
}

type Upstream struct {
	Client  jsonrpc.IRpcClient
	Metrics Metrics
}

func (u Upstream) Close() error {
	return u.Client.Close()
}

func (c *ClientConfig) Dial(ctx context.Context) (Upstream, error) {
	client, err := DialContext(ctx, c.Url)
	if err != nil {
		return Upstream{}, err
	}
	return Upstream{Client: client, Metrics: c.defineMetrics(c.Labels)}, nil
}

func (c *ClientConfig) Close() {
	// if c.Client != nil {
	// 	c.Client.Close()
	// }
}

func (c *ClientConfig) defineMetrics(labels []any) Metrics {
	histogramOpts := prometheus.HistogramOpts{
		Subsystem:   "clientrpc",
		Name:        "req_sec",
		Help:        "RPC request time",
		ConstLabels: commons.LabelsToMap(labels),
		Buckets:     []float64{0.1, 1, 5, 10},
	}
	requests := promauto.NewHistogram(histogramOpts)

	up := promauto.NewGauge(prometheus.GaugeOpts{
		Subsystem:   "clientrpc",
		Name:        "up",
		Help:        "Upstream connection status",
		ConstLabels: commons.LabelsToMap(labels),
	})
	return Metrics{Requests: requests, Upstreams: up}
}

func (c *ClientConfig) IsConnectionError(err error) bool {
	return rpc.ErrClientQuit == err
}

func (c *ClientConfig) GetLimitRps() uint32 {
	return uint32(c.LimitRps)
}

type Metrics struct {
	// requests number and duration
	Requests prometheus.Histogram
	// number of upstreams
	Upstreams prometheus.Gauge
}

type BalancedClient struct {
	Clients  []*ClientConfig
	Balancer *balancer.ClientBalancer[Upstream]
	Log      *slog.Logger
}

func NewBalancedClient(clients []*ClientConfig, labels []any) *BalancedClient {
	var dialers []balancer.ClientDialer[Upstream]
	for _, client := range clients {
		dialers = append(dialers, client)
	}

	logger := slog.With(labels...)
	c := &BalancedClient{Clients: clients, Balancer: balancer.NewBalancerWLog[Upstream](dialers, &balancer.Observations[Upstream]{
		OnConnectionStatusChange: func(client Upstream, connected bool) {
			if connected {
				client.Metrics.Upstreams.Set(1)
			} else {
				client.Metrics.Upstreams.Set(0)
			}
		},
	}, logger), Log: logger}

	return c
}

func (c *BalancedClient) Start() *BalancedClient {
	c.Balancer.Start()
	return c
}

func (c *BalancedClient) BatchCallContext(ctx context.Context, batch *jsonrpc.RpcBatch) (err error) {
	if c.Log.Enabled(ctx, slog.LevelDebug) {
		for _, elem := range batch.Calls {
			c.Log.DebugContext(ctx, "BatchRequest", "method", elem.Method, "args", elem.Params)
		}
	}
	err = c.Balancer.CallW(ctx, func(ctx context.Context, us Upstream) error {
		start := time.Now()
		res := us.Client.BatchCallContext(ctx, batch)
		us.Metrics.Requests.Observe(float64(time.Since(start).Seconds()))
		return res
	})
	if c.Log.Enabled(ctx, slog.LevelDebug) {
		for _, elem := range batch.Calls {
			c.Log.DebugContext(ctx, "BatchResponse", "method", elem.Method, "result", elem.Result, "error", elem.Error)
		}
	}
	return err
}

func (c *BalancedClient) CallContext(ctx context.Context, raw *jsonrpc.RawCall) (err error) {
	if c.Log.Enabled(ctx, slog.LevelDebug) {
		c.Log.DebugContext(ctx, "Request", "method", raw.Method, "args", raw.Params)
	}
	err = c.Balancer.CallW(ctx, func(ctx context.Context, us Upstream) error {
		start := time.Now()
		res := us.Client.CallContext(ctx, raw)
		us.Metrics.Requests.Observe(float64(time.Since(start).Seconds()))
		return res
	})
	if c.Log.Enabled(ctx, slog.LevelDebug) {
		c.Log.DebugContext(ctx, "Response", "method", raw.Method, "result", raw.Result, "error", raw.Error)
	}
	return err
}

func (c *BalancedClient) Call(raw *jsonrpc.RawCall) (err error) {
	return c.CallContext(context.Background(), raw)
}

func (c *BalancedClient) Close() error {
	return c.Balancer.Close()
}

func (c *BalancedClient) CallEveryUpstream(ctx context.Context, raw *jsonrpc.RawCall) (err error) {
	return c.Balancer.CallEveryUpstream(ctx, func(ctx context.Context, us Upstream) error {
		err := us.Client.CallContext(ctx, raw)
		if err != nil {
			c.Log.Error("CallEveryUpstream", "error", err)
		}
		return err
	})
}
