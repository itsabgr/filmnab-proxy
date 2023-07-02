package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io"
	"strings"
)

type Source struct {
	Bucket string `yaml:"bucket"`
	Host   string `yaml:"host"`
	ID     string `yaml:"id"`
	Key    string `yaml:"key"`
}
type clientWithBucketName struct {
	api    s3iface.S3API
	bucket string
}
type S3Client struct {
	clients map[string]clientWithBucketName
}

/*
	&aws.Config{
			Credentials:      credentials.NewStaticCredentials(config.Source.ID, config.Source.Key, ""),
			Endpoint:         aws.String(config.Source.Host),
			Region:           aws.String("us-east-1"),
			DisableSSL:       aws.Bool(false),
			S3ForcePathStyle: aws.Bool(true),
		}, config.Source.Bucket, config.Source.Test)
*/
func Connect(configs map[string]Source) (*S3Client, error) {
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
		}
	}
	return &S3Client{clients}, nil
}
func (s *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	//TODO fetch once
	switch key {
	case "", " ", "/", ".", "./", "//":
		return nil, errors.New("invalid key")
	}
	parts := strings.SplitN(strings.TrimLeft(key, "/"), "/", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid source+path format")
	}
	source := parts[0]
	client, has := s.clients[source]
	if !has {
		fmt.Println(key)
		return nil, errors.New("source not found")
	}
	path := parts[1]
	response, err := client.api.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &client.bucket,
		Key:    &path,
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
	if int64(len(content)) != *response.ContentLength {
		return nil, errors.New("failed to read body")
	}
	return content, nil
}
