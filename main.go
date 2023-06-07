package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var config struct {
	S3Proxy string `yaml:"s3proxy"`
	Server  struct {
		Addr string `yaml:"addr"`
		TLS  struct {
			Key  string `yaml:"key"`
			Cert string `yaml:"cert"`
		} `yaml:"tls"`
		Timeouts struct {
			Read  time.Duration `yaml:"read"`
			Write time.Duration `yaml:"write"`
			Idle  time.Duration `yaml:"idle"`
		} `yaml:"timeouts"`
		Headers struct {
			CORS  string `yaml:"cors"`
			Cache string `yaml:"cache"`
		} `yaml:"headers"`
	} `yaml:"server"`
	Source struct {
		Bucket string `yaml:"bucket"`
		Host   string `yaml:"host"`
		ID     string `yaml:"id"`
		Key    string `yaml:"key"`
	} `yaml:"source"`
	PublicKeys []string `yaml:"public-keys"`
	Cache      struct {
		SizeGB uint16 `yaml:"size"`
		Dir    string `yaml:"dir"`
	} `yaml:"cache"`
}
var flagConfig = flag.String("c", "./config.yaml", "yaml config file path")

var tempDir = os.TempDir()

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	flag.Parse()
	throw(yaml.Unmarshal(must(os.ReadFile(*flagConfig)), &config))
	if config.S3Proxy == "" {
		panic(errors.New(`config file does not contains "s3proxy""`))
	} else {
		fmt.Println("config: ", config.S3Proxy)
	}
	if len(config.PublicKeys) == 0 {
		fmt.Println("NO AUTH")
	}
	if config.Cache.SizeGB == 0 {
		fmt.Println("NO CACHE")
	}
	publicKeys := mustParsePublicKeys(config.PublicKeys...)
	client := must(Connect(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.Source.ID, config.Source.Key, ""),
		Endpoint:         aws.String(config.Source.Host),
		Region:           aws.String("us-east-1"),
		DisableSSL:       aws.Bool(false),
		S3ForcePathStyle: aws.Bool(true),
	}, config.Source.Bucket, tempDir))
	cache := Open(config.Cache.Dir, int64(config.Cache.SizeGB)*1e+9, client.Download)
	defer func() { _ = cache.Close() }()
	server := &Server{
		publicKeys:  publicKeys,
		cache:       cache,
		corsHeader:  config.Server.Headers.CORS,
		cacheHeader: config.Server.Headers.Cache,
	}
	httpServer := &http.Server{
		Addr:                         config.Server.Addr,
		ReadHeaderTimeout:            config.Server.Timeouts.Read,
		ReadTimeout:                  config.Server.Timeouts.Read,
		WriteTimeout:                 config.Server.Timeouts.Write,
		IdleTimeout:                  config.Server.Timeouts.Idle,
		Handler:                      server,
		MaxHeaderBytes:               5000, //5KB
		DisableGeneralOptionsHandler: true,
		ErrorLog:                     log.New(io.Discard, "", 0),
	}
	log.Fatal(serve(httpServer, config.Server.TLS.Cert, config.Server.TLS.Key))
}
