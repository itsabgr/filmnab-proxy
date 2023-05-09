package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
)

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
func LoadPK(url string) (ed25519.PublicKey, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d %s", res.StatusCode, res.Status)
	}

	res.Close = true
	defer Close(res.Body)
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	pub, err := base64.RawURLEncoding.DecodeString(string(b))
	if err != nil {
		return nil, err
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}
	return pub, nil
}
