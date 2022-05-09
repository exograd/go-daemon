package jsonpointer

import (
	"bytes"
	"errors"
	"strings"
)

type Pointer []string

var ErrInvalidFormat = errors.New("invalid format")

var (
	tokenEncoder *strings.Replacer
	tokenDecoder *strings.Replacer
)

func init() {
	tokenEncoder = strings.NewReplacer("~", "~0", "/", "~1")
	tokenDecoder = strings.NewReplacer("~1", "/", "~0", "~")
}

func (p *Pointer) Parse(s string) error {
	if len(s) == 0 {
		*p = Pointer{}
		return nil
	}

	if s[0] != '/' {
		return ErrInvalidFormat
	}

	parts := strings.Split(s[1:], "/")

	tokens := make([]string, len(parts))
	for i, part := range parts {
		tokens[i] = decodeToken(part)
	}

	*p = Pointer(tokens)

	return nil
}

func (p Pointer) String() string {
	var buf bytes.Buffer

	for _, token := range p {
		buf.WriteByte('/')
		buf.WriteString(encodeToken(token))
	}

	return buf.String()
}

func (p *Pointer) Append(token string) {
	*p = append(*p, token)
}

func encodeToken(s string) string {
	return tokenEncoder.Replace(s)
}

func decodeToken(s string) string {
	return tokenDecoder.Replace(s)
}
