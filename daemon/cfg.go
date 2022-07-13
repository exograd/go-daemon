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

package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

var TemplateFuncMap = map[string]interface{}{
	"env": func(name string) string {
		return os.Getenv(name)
	},
}

func LoadCfg(filePath string, dest interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	data2, err := RenderCfg(data)
	if err != nil {
		return fmt.Errorf("cannot render %s: %w", filePath, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data2))

	var yamlValue interface{}
	if err := decoder.Decode(&yamlValue); err != nil && err != io.EOF {
		return fmt.Errorf("cannot decode yaml data: %w", err)
	}

	jsonValue, err := YAMLValueToJSONValue(yamlValue)
	if err != nil {
		return fmt.Errorf("invalid yaml data: %w", err)
	}

	jsonData, err := json.Marshal(jsonValue)
	if err != nil {
		return fmt.Errorf("cannot generate json data: %w", err)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("cannot decode json data: %w", err)
	}

	return nil
}

func RenderCfg(data []byte) ([]byte, error) {
	tpl := template.New("")
	tpl = tpl.Option("missingkey=error")
	tpl = tpl.Funcs(TemplateFuncMap)

	if _, err := tpl.Parse(string(data)); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	templateData := struct{}{}

	if err := tpl.Execute(&buf, templateData); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
