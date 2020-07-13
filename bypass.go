package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sync"
)

// These are for bypassing a strange check introduced by Codeforces.
// See https://github.com/meooow25/cfspy/issues/4

type cfJar struct {
	rcpc *http.Cookie
	mu   sync.Mutex
}

func (j *cfJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie.Name == "RCPC" {
			j.mu.Lock()
			j.rcpc = cookie
			j.mu.Unlock()
		}
	}
}

func (j *cfJar) Cookies(u *url.URL) []*http.Cookie {
	if u.Host == "codeforces.com" {
		j.mu.Lock()
		defer j.mu.Unlock()
		if j.rcpc != nil {
			return []*http.Cookie{j.rcpc}
		}
	}
	return nil
}

var (
	aRe = regexp.MustCompile(`a=toNumbers\("([0-9a-f]+)"\)`)
	bRe = regexp.MustCompile(`b=toNumbers\("([0-9a-f]+)"\)`)
	cRe = regexp.MustCompile(`c=toNumbers\("([0-9a-f]+)"\)`)
)

func setStrangeCookieOnClient(script string, client http.Client) error {
	var a, b, c string
	if match := aRe.FindStringSubmatch(script); match != nil {
		a = match[1]
	} else {
		return errors.New("a not found")
	}
	if match := bRe.FindStringSubmatch(script); match != nil {
		b = match[1]
	} else {
		return errors.New("b not found")
	}
	if match := cRe.FindStringSubmatch(script); match != nil {
		c = match[1]
	} else {
		return errors.New("c not found")
	}

	// Adapted from example at https://golang.org/pkg/crypto/cipher/#NewCBCDecrypter
	key, err := hex.DecodeString(a)
	if err != nil {
		return err
	}
	ciphertext, err := hex.DecodeString(c)
	if err != nil {
		return err
	}
	iv, err := hex.DecodeString(b)
	if err != nil {
		return err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return errors.New("ciphertext is not a multiple of the block size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	rcpc := &http.Cookie{
		Name:  "RCPC",
		Value: hex.EncodeToString(ciphertext),
		Path:  "/",
	}
	cfURL := &url.URL{Scheme: "https", Host: "codeforces.com"}
	client.Jar.SetCookies(cfURL, []*http.Cookie{rcpc})
	return nil
}
