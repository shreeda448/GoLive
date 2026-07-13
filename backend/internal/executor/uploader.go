package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func UploadToS3(ctx context.Context, fileReader io.Reader, objectKey string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("admin", "password@212", "")))
	if err != nil {
		return "", err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
	})
	myBucket := "golive-build-artifacts"
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, fileReader)
	if err != nil {
		return "", fmt.Errorf("failed to buffer file stream: %v", err)
	}
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &myBucket,
		Key:    &objectKey,
		Body:   bytes.NewReader(buf.Bytes()),
	})
	if err != nil {
		return "", err
	}
	artifactURL := fmt.Sprintf("http://localhost:9000/%s/%s", myBucket, objectKey)
	return artifactURL, nil
}
