// Copyright (c) 2022 Exograd SAS.
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY
// SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR
// IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package dcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type AES256Key [32]byte

const (
	AES256IVSize int = aes.BlockSize
)

func (key AES256Key) Hex() string {
	return hex.EncodeToString(key[:])
}

func (key *AES256Key) FromHex(s string) error {
	data, err := hex.DecodeString(s)
	if err != nil {
		return err
	}

	if len(data) != 32 {
		return fmt.Errorf("invalid key size")
	}

	copy((*key)[:], data[:32])

	return nil
}

func EncryptAES256(inputData []byte, key AES256Key) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}

	paddedData := PadPKCS5(inputData, aes.BlockSize)

	outputData := make([]byte, AES256IVSize+len(paddedData))

	iv := outputData[:AES256IVSize]
	encryptedData := outputData[AES256IVSize:]

	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("cannot generate iv: %w", err)
	}

	encrypter := cipher.NewCBCEncrypter(blockCipher, iv)
	encrypter.CryptBlocks(encryptedData, paddedData)

	return outputData, nil
}

func DecryptAES256(inputData []byte, key AES256Key) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("cannot create cipher: %w", err)
	}

	if len(inputData) < AES256IVSize {
		return nil, fmt.Errorf("truncated data")
	}

	iv := inputData[:AES256IVSize]
	paddedData := inputData[AES256IVSize:]

	decrypter := cipher.NewCBCDecrypter(blockCipher, iv)
	decrypter.CryptBlocks(paddedData, paddedData)

	outputData, err := UnpadPKCS5(paddedData)
	if err != nil {
		return nil, fmt.Errorf("invalid padded data: %w", err)
	}

	return outputData, nil
}
