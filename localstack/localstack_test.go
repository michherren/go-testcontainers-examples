package localstack

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"log"
	"testing"
)

const (
	awsRegion       = "us-east-1"
	localstackImage = "localstack/localstack:3.2.0"
)

func TestS3Lookup(t *testing.T) {
	ctx := context.Background()

	client, localstackContainer, err := bootstrapS3Client(ctx)
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	// Clean up the container
	defer func() {
		if err := localstackContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate container: %s", err)
		}
	}()

	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		t.Fatalf("fail to get buckets: %v", err)
	}

	t.Logf("buckets: %v", buckets.Buckets)
}

func bootstrapS3Client(ctx context.Context) (*s3.Client, *localstack.LocalStackContainer, error) {
	localstackContainer, err := localstack.RunContainer(ctx,
		testcontainers.WithImage(localstackImage),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start container: %s", err)
	}

	mappedPort, err := localstackContainer.MappedPort(ctx, nat.Port("4566/tcp"))
	if err != nil {
		return nil, nil, fmt.Errorf("could not map port: %v", err)
	}

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return nil, nil, fmt.Errorf("could get provider: %v", err)
	}
	defer provider.Close()

	host, err := provider.DaemonHost(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("host: %v", err)
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           fmt.Sprintf("http://%s:%d", host, mappedPort.Int()),
				SigningRegion: region,
			}, nil
		})

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("noop", "noop", "noop")),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get aws config: %v", err)
	}

	// Create the resource client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return client, localstackContainer, nil
}
