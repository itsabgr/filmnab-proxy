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

func validateKey(key string) bool {
	if len(key) > 255 {
		return false
	}
	if len(key) <= 0 {
		return false
	}
	if key == "/" {
		return false
	}
	if strings.Contains(key, "//") {
		return false
	}
	if strings.Contains(key, "./") {
		return false
	}
	if strings.Contains(key, "/.") {
		return false
	}
	if strings.Contains(key, `\`) {
		return false
	}
	if strings.Contains(key, ":") {
		return false
	}
	if filepath.Ext(key) == "" {
		return false
	}
	return true
}

func Auth(path string, public ...ed25519.PublicKey) (filePath string, err error) {
	if len(public) == 0 {
		if false == validateKey(path) {
			return "", errors.New("invalid path")
		}
		return path, nil
	}
	path = strings.Trim(path, "/")
	for _, pk := range public {
		switch err = auth(filepath.Dir(path), pk); err {
		case errNoAuth:
			continue
		case nil:
			key := "/" + strings.SplitN(path, "/", 3)[2]
			if false == validateKey(key) {
				return "", errors.New("invalid path")
			}
			return key, nil
		default:
			break
		}
	}
	return "", errors.New("auth failed: " + err.Error())
}
func genAuth(dir string, deadline time.Time, key ed25519.PrivateKey) string {
	var token string
	if strings.HasPrefix(dir, "/") {
		token = strconv.FormatInt(deadline.UTC().Unix(), 10) + dir
	} else {
		token = strconv.FormatInt(deadline.UTC().Unix(), 10) + "/" + dir
	}
	sig := base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, []byte(token)))
	return sig + "/" + token
}

var errNoAuth = errors.New("unauthorized")

func auth(dir string, public ed25519.PublicKey) error {
	parts := strings.SplitN(dir, "/", 2)
	if len(parts) != 2 {
		return errNoAuth
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[0])
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
		return errNoAuth
	}
	return nil
}
