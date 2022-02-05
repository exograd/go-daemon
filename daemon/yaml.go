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

import "fmt"

func YAMLValueToJSONValue(yamlValue interface{}) (interface{}, error) {
	// For some reason, go-yaml will return objects as map[string]interface{}
	// if all keys are strings, and as map[interface{}]interface{} if not. So
	// we have to handle both.

	var jsonValue interface{}

	switch v := yamlValue.(type) {
	case []interface{}:
		array := make([]interface{}, len(v))

		for i, yamlElement := range v {
			jsonElement, err := YAMLValueToJSONValue(yamlElement)
			if err != nil {
				return nil, err
			}

			array[i] = jsonElement
		}

		jsonValue = array

	case map[interface{}]interface{}:
		object := make(map[string]interface{})

		for key, yamlEntry := range v {
			keyString, ok := key.(string)
			if !ok {
				return nil,
					fmt.Errorf("object key \"%v\" is not a string", key)
			}

			jsonEntry, err := YAMLValueToJSONValue(yamlEntry)
			if err != nil {
				return nil, err
			}

			object[keyString] = jsonEntry
		}

		jsonValue = object

	case map[string]interface{}:
		object := make(map[string]interface{})

		for key, yamlEntry := range v {
			jsonEntry, err := YAMLValueToJSONValue(yamlEntry)
			if err != nil {
				return nil, err
			}

			object[key] = jsonEntry
		}

		jsonValue = object

	default:
		jsonValue = yamlValue
	}

	return jsonValue, nil
}
