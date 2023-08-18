package main

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io"
	"path/filepath"
	"time"
	"unicode/utf8"
)

type Source struct {
	Bucket string `yaml:"bucket"`
	Host   string `yaml:"host"`
	ID     string `yaml:"id"`
	Key    string `yaml:"key"`
	Root   string `yaml:"root"`
}
type clientWithBucketName struct {
	api    s3iface.S3API
	bucket string
	root   string
}
type S3Client struct {
	clients        map[string]clientWithBucketName
	defaultTimeout time.Duration
}

func Connect(configs map[string]Source, defaultTimeout time.Duration) (*S3Client, error) {
	clients := map[string]clientWithBucketName{}
	for name, config := range configs {
		ses, err := session.NewSession(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(config.ID, config.Key, ""),
			Endpoint:         aws.String(config.Host),
			Region:           aws.String("us-east-1"),
			DisableSSL:       aws.Bool(false),
			S3ForcePathStyle: aws.Bool(true),
		})
		if err != nil {
			return nil, err
		}
		clients[name] = clientWithBucketName{
			s3.New(ses),
			config.Bucket,
			config.Root,
		}
	}
	return &S3Client{clients, defaultTimeout}, nil
}
func (s *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	//TODO fetch once
	switch key {
	case "", " ", "/", ".", "./", "//":
		return nil, errors.New("invalid key")
	}
	if !utf8.ValidString(key) {
		return nil, errors.New("non-utf8 key")
	}
	return downloadAny(ctx, s.clients, key, s.defaultTimeout)
}

func download(ctx context.Context, client *clientWithBucketName, path string) ([]byte, error) {
	rootPath := path
	switch client.root {
	case "", "/":
	default:
		rootPath = filepath.Join(client.root, path)
	}
	response, err := client.api.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &client.bucket,
		Key:    aws.String(rootPath),
	})
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) && awsErr.Code() == s3.ErrCodeNoSuchKey {
			return nil, nil
		}
		return nil, err
	}
	defer Close(response.Body)
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("could not download: " + err.Error())
	}
	if int64(len(content)) != *response.ContentLength {
		return nil, errors.New("failed to read body")
	}
	return content, nil
}
func downloadTimeout(ctx context.Context, client *clientWithBucketName, path string, timeout time.Duration) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return download(ctx, client, path)
}
func downloadAny(ctx context.Context, sources map[string]clientWithBucketName, path string, timeout time.Duration) ([]byte, error) {
	var last error
	for _, source := range sources {
		var content []byte
		content, last = downloadTimeout(ctx, &source, path, timeout)
		if len(content) > 0 {
			return content, nil
		}
	}
	return nil, last
}
