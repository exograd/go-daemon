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
	"bytes"
	"fmt"
)

func PadPKCS5(data []byte, blockSize int) []byte {
	paddingSize := blockSize - len(data)%blockSize
	padding := bytes.Repeat([]byte{byte(paddingSize)}, paddingSize)

	return append(data, padding...)
}

func UnpadPKCS5(data []byte, blockSize int) ([]byte, error) {
	dataSize := len(data)

	if dataSize%blockSize != 0 {
		return nil, fmt.Errorf("truncated data")
	}

	paddingSize := int(data[dataSize-1])
	if paddingSize > dataSize || paddingSize > blockSize {
		return nil, fmt.Errorf("invalid padding size %d", paddingSize)
	}

	return data[:dataSize-paddingSize], nil
}
