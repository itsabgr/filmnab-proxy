package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"os"
)

type S3Client struct {
	downloader *s3manager.Downloader
	bucket     string
	tmp        string
}

func Connect(config *aws.Config, bucket, tmp string) (*S3Client, error) {
	ses := must(session.NewSession(config))
	_, err := s3.New(must(session.NewSession(config))).HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, err
	}
	return &S3Client{s3manager.NewDownloader(ses), bucket, tmp}, nil
}
func (s *S3Client) Download(ctx context.Context, key string) ([]byte, error) {
	file := must(os.CreateTemp(tempDir, "*.s3proxy"))
	defer func() {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
	}()
	_, err := s.downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err == nil {
		return io.ReadAll(file)
	}
	if err, ok := err.(awserr.Error); ok && err.Code() == s3.ErrCodeNoSuchKey {
		return nil, nil
	}
	return nil, err

}
