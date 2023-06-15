package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"golang.org/x/crypto/acme/autocert"
	yaml "gopkg.in/yaml.v3"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var config struct {
	S3Proxy string `yaml:"s3proxy"`
	Server  struct {
		Addr string `yaml:"addr"`
		TLS  struct {
			ACME struct {
				Domains  []string `yaml:"domains"`
				CacheDir string   `yaml:"cache-dir"`
				Email    string   `yaml:"email"`
			} `yaml:"acme"`
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
var flagConfig = flag.String("c", "./s3proxy.yaml", "yaml config file path")
var flagDebug = flag.Bool("debug", false, "debug mode")
var tempDir = os.TempDir()

func main() {
	defer func() {
		if !*flagDebug {
			if err := recover(); err != nil {
				log.Fatal(err)
			}
		}
	}()
	flag.Parse()
	*flagConfig = must(filepath.Abs(*flagConfig))
	fmt.Println("config", *flagConfig)
	throw(yaml.Unmarshal(must(os.ReadFile(*flagConfig)), &config))
	if config.S3Proxy != "2" {
		panic(errors.New(`config file does not contains "s3proxy: 2""`))
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
	throw(serve(httpServer))
}

func serve(httpServer *http.Server) error {
	if config.Server.TLS.Key != "" {
		return httpServer.ListenAndServeTLS(config.Server.TLS.Cert, config.Server.TLS.Key)
	}
	if len(config.Server.TLS.ACME.Domains) > 0 {
		if config.Server.TLS.ACME.CacheDir == "" {
			panic(errors.New("config.Server.TLS.ACME.CacheDir is empty"))
		}
		throw(os.MkdirAll(config.Server.TLS.ACME.CacheDir, 0700))
		acme := autocert.Manager{
			Cache:      autocert.DirCache(config.Server.TLS.ACME.CacheDir),
			HostPolicy: autocert.HostWhitelist(config.Server.TLS.ACME.Domains...),
			Email:      config.Server.TLS.ACME.Email,
		}
		ln := must(tls.Listen("tcp", httpServer.Addr, acme.TLSConfig()))
		defer func() { _ = ln.Close() }()
		return httpServer.Serve(ln)
	}
	fmt.Println("NO TLS")
	return httpServer.ListenAndServe()
}
