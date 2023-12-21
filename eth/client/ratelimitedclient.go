package client

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/time/rate"
)

type RateLimitedClient struct {
	ethclient.Client
	Limiter rate.Limiter
}

func (c *RateLimitedClient) BatchCallContext(ctx context.Context, batch []rpc.BatchElem) error {
	if err := c.Limiter.Wait(ctx); err != nil {
		return err
	}
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		for _, elem := range batch {
			slog.Debug("BatchRequest", "method", elem.Method, "args", elem.Args)
		}
	}
	err := c.Client.Client().BatchCallContext(ctx, batch)
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		for _, elem := range batch {
			slog.Debug("BatchResponse", "method", elem.Method, "result", elem.Result, "error", elem.Error)
		}
	}
	return err
}

func (c *RateLimitedClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	if err := c.Limiter.Wait(ctx); err != nil {
		return err
	}
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.Debug("Request", "method", method, "args", args)
	}
	err := c.Client.Client().CallContext(ctx, result, method, args...)
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.Debug("Response", "method", method, "result", result, "error", err)
	}
	return err
}

func DialContext(ctx context.Context, url string, limitRps float64) (*RateLimitedClient, error) {
	actualLimitRps := rate.Limit(limitRps)
	if limitRps == 0 {
		actualLimitRps = rate.Inf
	}
	c := RateLimitedClient{Limiter: *rate.NewLimiter(actualLimitRps, int(limitRps*60))}
	client, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	c.Client = *client
	return &c, nil
}
