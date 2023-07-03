package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

func throw(e error) {
	if e != nil {
		panic(e)
	}
}
func must[R any](r R, e error) R {
	throw(e)
	return r
}

func assert(cond bool) {
	if !cond {
		panic(errors.New("assertion"))
	}
}
func Close(closer io.Closer) {
	if closer != nil {
		_ = closer.Close()
	}
}
func mustParsePublicKeys(keys ...string) []ed25519.PublicKey {
	result := make([]ed25519.PublicKey, 0, len(keys))
	for _, b64 := range keys {
		pk := ed25519.PublicKey(must(base64.RawURLEncoding.DecodeString(b64)))
		if len(pk) != ed25519.PublicKeySize {
			panic(fmt.Errorf("public key size is not %d", ed25519.PublicKeySize))
		}
		result = append(result, pk)
	}
	return result
}

func removeFile(file *os.File) {
	if file == nil {
		return
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
}
