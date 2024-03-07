package toxiproxy

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/network"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/google/uuid"
)

func TestRedisLatency(t *testing.T) {
	ctx := context.Background()

	toxiProxy, redisClient, err := setupTests(ctx, t)
	if err != nil {
		t.Fatalf("could not setup tests: %v", err)
	}
	defer flushRedis(ctx, *redisClient)

	// set data
	key := fmt.Sprintf("{user.%s}.go-meetup", uuid.NewString())
	value := "attendance"
	ttl, _ := time.ParseDuration("2h")
	err = redisClient.Set(ctx, key, value, ttl).Err()
	if err != nil {
		t.Fatal(err)
	}

	// introduce chaos
	_, err = toxiProxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 1000,
		"jitter":  100,
	})
	if err != nil {
		t.Fatal(err)
	}

	// get data
	savedValue, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		t.Fatal(err)
	}

	// assert
	if savedValue != value {
		t.Fatalf("expected: %s got: %s", savedValue, value)
	}
}

func setupTests(ctx context.Context, t *testing.T) (*toxiproxy.Proxy, *redis.Client, error) {
	newNetwork, err := network.New(ctx, network.WithCheckDuplicate())
	if err != nil {
		return nil, nil, fmt.Errorf("could not create network: %v", err)
	}
	networkName := newNetwork.Name

	toxiproxyContainer, err := startContainer(ctx, networkName, []string{"toxiproxy"})
	if err != nil {
		return nil, nil, fmt.Errorf("could not start toxiproxy: %v", err)
	}

	redisContainer, err := tcRedis.RunContainer(ctx, testcontainers.WithImage("redis:7.2"), network.WithNetwork([]string{"redis"}, newNetwork))
	if err != nil {
		return nil, nil, fmt.Errorf("could not start redis: %v", err)
	}

	t.Cleanup(func() {
		if err := toxiproxyContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
		if err := newNetwork.Remove(ctx); err != nil {
			t.Fatalf("failed to terminate networkName: %s", err)
		}
	})

	toxiproxyClient := toxiproxy.NewClient(toxiproxyContainer.URI)
	proxy, err := toxiproxyClient.CreateProxy("redis", "0.0.0.0:8666", "redis:6379")
	if err != nil {
		t.Fatal(err)
	}

	toxiproxyProxyPort, err := toxiproxyContainer.MappedPort(ctx, "8666")
	if err != nil {
		t.Fatal(err)
	}

	toxiproxyHostIP, err := toxiproxyContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	redisUri := fmt.Sprintf("redis://%s:%s?read_timeout=2s", toxiproxyHostIP, toxiproxyProxyPort.Port())
	options, err := redis.ParseURL(redisUri)
	if err != nil {
		t.Fatal(err)
	}
	redisClient := redis.NewClient(options)

	return proxy, redisClient, err
}

func flushRedis(ctx context.Context, client redis.Client) error {
	return client.FlushAll(ctx).Err()
}
