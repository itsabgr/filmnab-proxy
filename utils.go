package main

import "io"

func Must[R any](r R, e error) R {
	if e != nil {
		panic(e)
	}
	return r
}
func Close(closer io.Closer) {
	if closer != nil {
		_ = closer.Close()
	}
}
