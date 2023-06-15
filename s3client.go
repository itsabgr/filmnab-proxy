package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io"
	"time"
)

type S3Client struct {
	client s3iface.S3API
	bucket string
}

func Connect(config *aws.Config, bucket string, test bool) (*S3Client, error) {
	cli := s3.New(must(session.NewSession(config)))
	if test {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_, err := cli.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket),
		})
		if ctx.Err() != nil {
			return nil, fmt.Errorf("test: could not connect to s3 %q", *config.Endpoint)
		}
		if err != nil {
			return nil, err
		}
	}
	return &S3Client{cli, bucket}, nil
}
func (s *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	//TODO fetch once
	switch key {
	case "", " ", "/", ".", "./", "//":
		return nil, errors.New("invalid key")
	}
	response, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		if err, ok := err.(awserr.Error); ok && err.Code() == s3.ErrCodeNoSuchKey {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("could not download: " + err.Error())
	}
	return content, nil
}
