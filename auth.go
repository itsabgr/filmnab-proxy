package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Auth(path string, public ed25519.PublicKey) (filePath string, err error) {
	path = strings.Trim(path, "/")
	if err := auth(filepath.Dir(path), public); err != nil {
		return "", err
	}
	return "/" + strings.SplitN(path, "/", 3)[2], nil
}
func genAuth(dir string, deadline time.Time, key ed25519.PrivateKey) string {
	token := strconv.FormatInt(deadline.UTC().Unix(), 10) + "/" + dir
	sig := base64.URLEncoding.EncodeToString(ed25519.Sign(key, []byte(token)))
	return sig + "/" + token
}
func auth(dir string, public ed25519.PublicKey) (err error) {
	parts := strings.SplitN(dir, "/", 2)
	if len(parts) != 2 {
		return errors.New("no auth")
	}
	sig, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}
	token := parts[1]
	parts = strings.SplitN(parts[1], "/", 2)
	if len(parts) != 2 {
		return errors.New("no timestamp")
	}
	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return err
	}
	if timestamp < 0 || timestamp < time.Now().UTC().Unix() {
		return errors.New("past timestamp")
	}
	if !ed25519.Verify(public, []byte(token), sig) {
		return errors.New("unauthorized")
	}
	return nil
}
