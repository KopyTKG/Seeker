package valkey

import (
	"context"
	"fmt"

	"github.com/valkey-io/valkey-go"
	vk "github.com/valkey-io/valkey-go"
	"thekrew.app/seeker/internal/cache"
)

type Client struct {
	c vk.Client
}

func New(address string) (*Client, error) {
	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{address}})
	if err != nil {
		return nil, fmt.Errorf("Error during valkey initialization: %s", err)
	}
	return &Client{client}, nil
}

func (v *Client) Close() {
	v.c.Close()
}

func (v *Client) Get(ctx context.Context, key string) (string, error) {
	vc, err := v.c.Do(ctx, v.c.B().Get().Key(key).Build()).ToString()
	if vk.IsValkeyNil(err) {
		return "", cache.ErrNotFound
	}
	return vc, err
}

func (v *Client) Set(ctx context.Context, key, data string) error {
	// Hard limit to 24h then expire
	return v.c.Do(ctx, v.c.B().Set().Key(key).Value(data).ExSeconds(86400).Build()).Error()
}

func (v *Client) Exists(ctx context.Context, key string) (bool, error) {
	n, err := v.c.Do(ctx, v.c.B().Exists().Key(key).Build()).AsInt64()
	return n > 0, err
}
