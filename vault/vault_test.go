package vault

import (
	"context"
	vaultClient "github.com/hashicorp/vault-client-go"
	"github.com/testcontainers/testcontainers-go/modules/vault"
	"log/slog"
	"testing"
	"time"
)

const (
	vaultToken      = "MyToKeN"
	vaultTestSecret = "password1234"
)

func TestGetKey(t *testing.T) {
	ctx := context.Background()
	vaultContainer, err := vault.RunContainer(ctx, vault.WithToken(vaultToken), vault.WithInitCommand("kv put -mount=secret testing value="+vaultTestSecret))
	if err != nil {
		panic(err)
	}

	// Clean up after completion
	t.Cleanup(func() {
		if err := vaultContainer.Terminate(ctx); err != nil {
			panic(err)
		}
	})

	host, _ := vaultContainer.HttpHostAddress(ctx)
	client, err := vaultClient.New(
		vaultClient.WithAddress(host),
		vaultClient.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("could not init vault: %v", err)
	}

	err = client.SetToken(vaultToken)
	if err != nil {
		t.Fatalf("could not set token: %v", err)
	}

	s, err := client.Secrets.KvV2Read(ctx, "testing", vaultClient.WithMountPath("secret"))
	slog.Info("data", "secret", s)

	if err != nil {
		slog.Error("err", err)
		t.Fail()
	}

	if s.Data.Data["value"] != vaultTestSecret {
		t.Fail()
	}

}
