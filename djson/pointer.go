package djson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

func NewPointer(tokens ...string) Pointer {
	return Pointer(tokens)
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

func (p *Pointer) MustParse(s string) {
	if err := p.Parse(s); err != nil {
		panic(fmt.Errorf("cannot parse json pointer %q: %w", s, err))
	}
}

func (p Pointer) String() string {
	var buf bytes.Buffer

	for _, token := range p {
		buf.WriteByte('/')
		buf.WriteString(encodeToken(token))
	}

	return buf.String()
}

func (p Pointer) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *Pointer) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	return p.Parse(s)
}

func (p *Pointer) Append(token string) {
	*p = append(*p, token)
}

func (p Pointer) Parent() Pointer {
	if len(p) == 0 {
		panic("empty pointer")
	}

	return append(Pointer{}, p[:len(p)-1]...)
}

func (p Pointer) Child(tokens ...string) Pointer {
	p2 := append(Pointer{}, p...)
	return append(p2, tokens...)
}

func (p Pointer) Find(value interface{}) interface{} {
	v := value

	for _, token := range p {
		switch tv := v.(type) {
		case []interface{}:
			i, err := strconv.ParseInt(token, 10, 64)
			if err != nil {
				return nil
			}

			if i < 0 || i >= int64(len(tv)) {
				return nil
			}

			v = tv[i]

		case map[string]interface{}:
			child, found := tv[token]
			if !found {
				return nil
			}

			v = child

		default:
			return nil
		}
	}

	return v
}

func encodeToken(s string) string {
	return tokenEncoder.Replace(s)
}

func decodeToken(s string) string {
	return tokenDecoder.Replace(s)
}
