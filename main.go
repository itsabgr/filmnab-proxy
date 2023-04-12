package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jlaffaye/ftp"
	_ "github.com/jlaffaye/ftp"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var flagAddr = flag.String("addr", "", "* listening address")
var flagKey = flag.String("key", "", "https key file path")
var flagCrt = flag.String("crt", "", "https certificate file path")
var flagHost = flag.String("host", "ftp(s)://[user]:[pass]@[host]:[port]/root", "* ftp host uri")
var flagCORS = flag.String("cors", "*", "'Access-Control-Allow-Origin' header value")
var flagCache = flag.String("cache", "no-store", "'Cache-Control' header value")

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()
	flag.Parse()
	var ftpURL = Must(url.Parse(*flagHost))
	ftpConnPool := NewFTPPool(*ftpURL)
	ftpConnPool.Put(Must(ftpConnPool.Get(context.Background())))
	mux := &http.ServeMux{}
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.Body != nil {
			request.Body.Close()
		}
		ctx, cancel := context.WithTimeout(request.Context(), time.Second*10)
		defer cancel()
		request = request.WithContext(ctx)
		switch request.Method {
		case http.MethodGet:
			break
		case http.MethodOptions:
			writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET")
			writer.Header().Set("Access-Control-Allow-Origin", *flagCORS)
			writer.WriteHeader(http.StatusNoContent)
			return
		default:
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var userAuth string
		var filePath string
		{
			requestPathParts := strings.SplitN(request.URL.Path, "/", 3)
			if len(requestPathParts) != 3 {
				http.Error(writer, "no authorization provided", http.StatusBadRequest)
				return
			}
			userAuth = requestPathParts[1]
			filePath = requestPathParts[2]
		}
		//TODO do authorization here **
		func(string) {}(userAuth)
		//**
		conn, err := ftpConnPool.Get(request.Context())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer ftpConnPool.Put(conn)
		writer.Header().Set("Cache-Control", *flagCache)
		if fileInfo, err := conn.GetEntry(filePath); err != nil || fileInfo.Type != ftp.EntryTypeFile {
			http.NotFound(writer, request)
			return
		}
		resp, err := conn.Retr(filePath)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Close()
		if deadline, ok := request.Context().Deadline(); ok {
			if err = resp.SetDeadline(deadline); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if _, err := io.Copy(writer, resp); err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	})
	httpServer := http.Server{
		Addr:                         *flagAddr,
		Handler:                      mux,
		DisableGeneralOptionsHandler: true,
		ReadHeaderTimeout:            time.Second * 2,
		ReadTimeout:                  time.Second * 2,
		WriteTimeout:                 time.Second * 15,
		IdleTimeout:                  time.Second * 15,
		MaxHeaderBytes:               2000,
		ErrorLog:                     log.New(io.Discard, "", 0),
	}
	go func() {
		time.Sleep(time.Second)
		fmt.Println("ready")
	}()
	if *flagKey == "" {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := httpServer.ListenAndServeTLS(*flagCrt, *flagKey); err != nil {
			log.Fatal(err)
		}
	}

}
