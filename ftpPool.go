package main

import (
	"context"
	ftp "github.com/jlaffaye/ftp"
	"net/url"
	"sync"
	"time"
)

type ftpPool struct {
	pool sync.Pool
	host url.URL
}

func NewFTPPool(host url.URL) *ftpPool {
	ftpPool := &ftpPool{pool: sync.Pool{}, host: host}
	ftpPool.pool.New = ftpPool.newConn
	return ftpPool
}
func (pool *ftpPool) newConn() any {
	opts := []ftp.DialOption{ftp.DialWithTimeout(3 * time.Second)}
	conn, err := ftp.Dial(pool.host.Host, opts...)
	if err != nil {
		panic(err)
	}
	pass, _ := pool.host.User.Password()
	err = conn.Login(pool.host.User.Username(), pass)
	if err != nil {
		panic(err)
	}
	err = conn.ChangeDir(pool.host.Path)
	if err != nil {
		panic(err)
	}
	return conn
}
func (pool *ftpPool) Get(ctx context.Context) (*ftp.ServerConn, error) {
	for {
		err := ctx.Err()
		if err != nil {
			return nil, err
		}
		conn := pool.pool.Get().(*ftp.ServerConn)
		err = conn.NoOp()
		if err == nil {
			return conn, nil
		}
	}
}
func (pool *ftpPool) Put(conn *ftp.ServerConn) {
	pool.pool.Put(conn)
}
