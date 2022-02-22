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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

type APIRequestError struct {
	Status   int
	APIError *APIError
	Message  string
}

func (err APIRequestError) Error() string {
	return err.Message
}

type APIClient struct {
	*Client
}

func NewAPIClient(c *Client) *APIClient {
	return &APIClient{
		Client: c,
	}
}

func (c *APIClient) SendRequest(method string, uri *url.URL, header map[string]string, body io.Reader) (*http.Response, error) {
	res, err := c.Client.SendRequest(method, uri, header, body)
	if err != nil {
		return nil, err
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		reqErr := &APIRequestError{
			Status:   res.StatusCode,
			APIError: nil,
			Message: fmt.Sprintf("request failed with status %d",
				res.StatusCode),
		}

		resBody, err := ioutil.ReadAll(res.Body)
		if err == nil {
			if res.Header.Get("Content-Type") == "application/json" {
				var apiErr APIError
				if err := json.Unmarshal(resBody, &apiErr); err == nil {
					reqErr.APIError = &apiErr
					reqErr.Message += ": " + apiErr.Message
				} else {
					c.Log.Error("cannot decode api error response: %v", err)
				}
			}

			if reqErr.APIError == nil && len(resBody) > 0 {
				reqErr.Message += ": " + string(resBody)
			}
		} else {
			c.Log.Error("cannot read response body: %v", err)
		}

		return res, reqErr
	}

	return res, nil
}

func (c *APIClient) SendJSONRequest(method string, uri *url.URL, header map[string]string, value interface{}) (*http.Response, error) {
	var body io.Reader

	if value != nil {
		var buf bytes.Buffer

		encoder := json.NewEncoder(&buf)
		if err := encoder.Encode(value); err != nil {
			return nil, fmt.Errorf("cannot encode request body: %w", err)
		}

		body = &buf
	}

	if header == nil {
		header = make(map[string]string)
	}

	if _, found := header["Content-Type"]; !found {
		header["Content-Type"] = "application/json"
	}

	return c.SendRequest(method, uri, header, body)
}
