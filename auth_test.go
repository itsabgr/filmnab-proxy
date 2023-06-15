package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestAuthCorrectness(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(time.Minute), key)
	if err := auth(token, public); err != nil {
		t.Error(err)
	}
	fp, err := Auth("/"+token+"/file.ext", public)
	if err != nil {
		t.Error(err)
	}
	if fp != "/dir/sub/dir2/file.ext" {
		t.Fail()
	}
}
func TestPrefixedDir(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	deadline := time.Now().Add(time.Minute)
	token1 := genAuth("dir/sub/dir2", deadline, key)
	token2 := genAuth("/dir/sub/dir2", deadline, key)
	if token2 != token1 {
		t.Fail()
	}
}
func TestAuthSubDir(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(time.Minute), key)
	if err := auth(token, public); err != nil {
		t.Error(err)
	}
	fp, err := Auth("/"+token+"/sub/file", public)
	if err == nil {
		t.Fail()
	}
	if fp != "" {
		t.Fail()
	}
}

func TestAuthPastTime(t *testing.T) {
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	token := genAuth("dir/sub/dir2", time.Now().Add(-time.Second), key)
	if err := auth(token, public); err == nil {
		t.Fail()
	}
}
