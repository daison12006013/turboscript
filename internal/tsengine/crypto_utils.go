/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Copyright (c) 2025 TurboScript Project Contributors
 * Author: Daison Cariño <daison12006013@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on TurboScript: https://github.com/daison12006013/turboscript
 */

// Package tsengine provides shared crypto utilities for JavaScript runtime.
package tsengine

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"
	"math/big"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	hashAlgoSHA256 = "sha256"
	hashAlgoSHA512 = "sha512"
)

// RegisterSharedCryptoModule registers the crypto module and all helpers in the goja runtime.
func RegisterSharedCryptoModule(rt *goja.Runtime, registry *require.Registry) {
	cryptoObj := rt.NewObject()
	setScryptSync(rt, cryptoObj)
	setHkdfSync(rt, cryptoObj)
	setGeneratePrimeSync(rt, cryptoObj)
	setCreateHash(rt, cryptoObj)
	setCreateHmac(rt, cryptoObj)
	setTimingSafeEqual(rt, cryptoObj)
	setRandomInt(rt, cryptoObj)
	setPbkdf2Sync(rt, cryptoObj)
	setRandomBytes(rt, cryptoObj)
	setRandomUUID(rt, cryptoObj)
	setRandomHex(rt, cryptoObj)
	registry.RegisterNativeModule("crypto", func(_ *goja.Runtime, module *goja.Object) {
		if err := module.Set("exports", cryptoObj); err != nil {
			logger.Error("Failed to set crypto exports: %v", err)
		}
	})
	registry.RegisterNativeModule("node:crypto", func(_ *goja.Runtime, module *goja.Object) {
		if err := module.Set("exports", cryptoObj); err != nil {
			logger.Error("Failed to set node:crypto exports: %v", err)
		}
	})
}

func setScryptSync(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("scryptSync", func(password, salt string, N, r, p, keylen int) string {
		dk, err := scrypt.Key([]byte(password), []byte(salt), N, r, p, keylen)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("scrypt failed: %w", err)))
		}
		return hex.EncodeToString(dk)
	}); err != nil {
		logger.Error("Failed to set scryptSync function: %v", err)
	}
}

func setHkdfSync(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("hkdfSync", func(hashName, secret, salt, info string, keylen int) string {
		var hashFunc func() hash.Hash
		switch strings.ToLower(hashName) {
		case hashAlgoSHA256:
			hashFunc = sha256.New
		case hashAlgoSHA512:
			hashFunc = sha512.New
		default:
			hashFunc = sha256.New
		}
		hkdfReader := hkdf.New(hashFunc, []byte(secret), []byte(salt), []byte(info))
		out := make([]byte, keylen)
		if _, err := hkdfReader.Read(out); err != nil {
			panic(rt.NewGoError(fmt.Errorf("hkdf failed: %w", err)))
		}
		return hex.EncodeToString(out)
	}); err != nil {
		logger.Error("Failed to set hkdfSync function: %v", err)
	}
}

func setGeneratePrimeSync(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("generatePrimeSync", func(bits int) string {
		if bits < 8 {
			bits = 8
		}
		prime, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("generatePrime failed: %w", err)))
		}
		return prime.Text(16)
	}); err != nil {
		logger.Error("Failed to set generatePrimeSync function: %v", err)
	}
}

func setCreateHash(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("createHash", func(algo string) *goja.Object {
		hashObj := rt.NewObject()
		var hasher hash.Hash
		switch strings.ToLower(algo) {
		case hashAlgoSHA256:
			hasher = sha256.New()
		case hashAlgoSHA512:
			hasher = sha512.New()
		default:
			panic(rt.NewGoError(fmt.Errorf("unsupported hash algorithm: %s", algo)))
		}
		if err := hashObj.Set("update", func(data string) *goja.Object {
			_, _ = hasher.Write([]byte(data))
			return hashObj
		}); err != nil {
			logger.Error("Failed to set hash.update: %v", err)
		}
		if err := hashObj.Set("digest", func(enc ...string) string {
			sum := hasher.Sum(nil)
			if len(enc) > 0 && enc[0] == "hex" {
				return hex.EncodeToString(sum)
			}
			return string(sum)
		}); err != nil {
			logger.Error("Failed to set hash.digest: %v", err)
		}
		return hashObj
	}); err != nil {
		logger.Error("Failed to set createHash function: %v", err)
	}
}

func setCreateHmac(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("createHmac", func(algo, key string) *goja.Object {
		hmacObj := rt.NewObject()
		var mac hash.Hash
		switch strings.ToLower(algo) {
		case hashAlgoSHA256:
			mac = hmac.New(sha256.New, []byte(key))
		case hashAlgoSHA512:
			mac = hmac.New(sha512.New, []byte(key))
		default:
			panic(rt.NewGoError(fmt.Errorf("unsupported hmac algorithm: %s", algo)))
		}
		if err := hmacObj.Set("update", func(data string) *goja.Object {
			_, _ = mac.Write([]byte(data))
			return hmacObj
		}); err != nil {
			logger.Error("Failed to set hmac.update: %v", err)
		}
		if err := hmacObj.Set("digest", func(enc ...string) string {
			sum := mac.Sum(nil)
			if len(enc) > 0 && enc[0] == "hex" {
				return hex.EncodeToString(sum)
			}
			return string(sum)
		}); err != nil {
			logger.Error("Failed to set hmac.digest: %v", err)
		}
		return hmacObj
	}); err != nil {
		logger.Error("Failed to set createHmac function: %v", err)
	}
}

func setTimingSafeEqual(_ *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("timingSafeEqual", func(a, b goja.ArrayBuffer) bool {
		ab := a.Bytes()
		bb := b.Bytes()
		if len(ab) != len(bb) {
			return false
		}
		return subtle.ConstantTimeCompare(ab, bb) == 1
	}); err != nil {
		logger.Error("Failed to set timingSafeEqual function: %v", err)
	}
}

func setRandomInt(_ *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("randomInt", func(minVal, maxVal int) int {
		if minVal >= maxVal {
			return minVal
		}
		nBig, err := rand.Int(rand.Reader, big.NewInt(int64(maxVal-minVal)))
		if err != nil {
			return minVal
		}
		return int(nBig.Int64()) + minVal
	}); err != nil {
		logger.Error("Failed to set randomInt function: %v", err)
	}
}

func setPbkdf2Sync(_ *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("pbkdf2Sync", func(password, salt string, iter, keylen int, algo string) string {
		var hashFunc func() hash.Hash
		switch strings.ToLower(algo) {
		case hashAlgoSHA256:
			hashFunc = sha256.New
		case hashAlgoSHA512:
			hashFunc = sha512.New
		default:
			hashFunc = sha256.New
		}
		dk := pbkdf2.Key([]byte(password), []byte(salt), iter, keylen, hashFunc)
		return hex.EncodeToString(dk)
	}); err != nil {
		logger.Error("Failed to set pbkdf2Sync function: %v", err)
	}
}

func setRandomBytes(rt *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("randomBytes", func(size int) goja.Value {
		if size <= 0 || size > 65536 {
			size = 32
		}
		buf := make([]byte, size)
		_, err := rand.Read(buf)
		if err != nil {
			for i := range buf {
				buf[i] = 0
			}
		}
		return rt.ToValue(rt.NewArrayBuffer(buf))
	}); err != nil {
		logger.Error("Failed to set randomBytes function: %v", err)
	}
}

func setRandomUUID(_ *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("randomUUID", func() string {
		uuid := make([]byte, 16)
		_, err := rand.Read(uuid)
		if err != nil {
			for i := range uuid {
				uuid[i] = 0
			}
		}
		uuid[6] = (uuid[6] & 0x0f) | 0x40
		uuid[8] = (uuid[8] & 0x3f) | 0x80
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16],
		)
	}); err != nil {
		logger.Error("Failed to set randomUUID function: %v", err)
	}
}

func setRandomHex(_ *goja.Runtime, cryptoObj *goja.Object) {
	if err := cryptoObj.Set("randomHex", func(size int) string {
		if size <= 0 || size > 65536 {
			size = 32
		}
		buf := make([]byte, size)
		_, err := rand.Read(buf)
		if err != nil {
			for i := range buf {
				buf[i] = 0
			}
		}
		return hex.EncodeToString(buf)
	}); err != nil {
		logger.Error("Failed to set randomHex function: %v", err)
	}
}
