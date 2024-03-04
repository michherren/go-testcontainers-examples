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

	proxy, redisClient, err := bootstrapRedisClient(ctx, t)
	if err != nil {
		t.Fatalf("could not bootstrap redis: %v", err)
	}
	defer flushRedis(ctx, *redisClient)

	// Set data
	key := fmt.Sprintf("{user.%s}.favoritefood", uuid.NewString())
	value := "Cabbage Biscuits"
	ttl, _ := time.ParseDuration("2h")
	err = redisClient.Set(ctx, key, value, ttl).Err()
	if err != nil {
		t.Fatal(err)
	}

	_, err = proxy.AddToxic("latency_down", "latency", "downstream", 1.0, toxiproxy.Attributes{
		"latency": 1000,
		"jitter":  100,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Get data
	savedValue, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		t.Fatal(err)
	}

	// perform assertions
	if savedValue != value {
		t.Fatalf("Expected value %s. Got %s.", savedValue, value)
	}
}

func bootstrapRedisClient(ctx context.Context, t *testing.T) (*toxiproxy.Proxy, *redis.Client, error) {
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

	// Clean up the container after the test is complete
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
