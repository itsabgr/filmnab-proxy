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
type client struct {
	api    s3iface.S3API
	bucket string
	root   string
}
type S3Client struct {
	clients        []client
	defaultTimeout time.Duration
}

func Connect(defaultTimeout time.Duration, list ...Source) (*S3Client, error) {
	var clients []client
	for _, source := range list {
		ses, err := session.NewSession(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(source.ID, source.Key, ""),
			Endpoint:         aws.String(source.Host),
			Region:           aws.String("us-east-1"),
			DisableSSL:       aws.Bool(false),
			S3ForcePathStyle: aws.Bool(true),
		})
		if err != nil {
			return nil, err
		}
		clients = append(clients, client{
			s3.New(ses),
			source.Bucket,
			source.Root,
		})
	}
	if len(clients) == 0 {
		panic(errors.New("no source"))
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
	return s.downloadAny(ctx, key)
}

func (s *S3Client) downloadAny(ctx context.Context, path string) ([]byte, error) {
	var last error
	for _, client := range s.clients {
		var content []byte
		content, last = client.DownloadTimeout(ctx, path, s.defaultTimeout)
		if len(content) > 0 {
			return content, nil
		}
	}
	return nil, last
}

func (client *client) download(ctx context.Context, path string) ([]byte, error) {
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
func (client *client) DownloadTimeout(ctx context.Context, path string, timeout time.Duration) ([]byte, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return client.download(ctx, path)
}
