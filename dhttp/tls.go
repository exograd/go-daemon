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

package dhttp

import (
	"crypto/x509"
	"fmt"
	"os"
)

func LoadCertificates(certificates []string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()

	for _, certificate := range certificates {
		data, err := os.ReadFile(certificate)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", certificate, err)
		}

		if pool.AppendCertsFromPEM(data) == false {
			return nil, fmt.Errorf("cannot load certificates from %s",
				certificate)
		}
	}

	return pool, nil
}
