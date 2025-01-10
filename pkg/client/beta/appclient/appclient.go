// Package client provides client for the Ongoku API.
package appclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/teejays/gokutil/env/envutil"
	"github.com/teejays/gokutil/errutil"
	jsonhelper "github.com/teejays/gokutil/gopi/json"
	"github.com/teejays/gokutil/log"
)

type Client struct {
	Token      string
	httpClient *http.Client
	baseURL    string
}

type Creds struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

const _defaultBaseURL = "http://localhost:8080"

func NewClient(ctx context.Context, creds Creds) (Client, error) {
	var ret Client

	if creds.Email == "" {
		return ret, fmt.Errorf("Email is empty")
	}
	if creds.Password == "" {
		return ret, fmt.Errorf("Password is empty")
	}

	baseURL := envutil.GetEnvVarStr("ONGOKU_CLI_SERVER_BASE_URL")
	if baseURL == "" {
		log.Warn(ctx, "Env variable ONGOKU_CLI_SERVER_BASE_URL is not set. Using default value", "default", _defaultBaseURL)
		baseURL = _defaultBaseURL
	}
	httpClient := &http.Client{}

	// Make a login request
	var tokenResp TokenResponse
	url, err := url.JoinPath(baseURL, "auth/login")
	if err != nil {
		return ret, errutil.Wrap(err, "Joining URL path")
	}

	err = MakeRequest(ctx, http.MethodPost, url, httpClient, creds, &tokenResp)
	if err != nil {
		return ret, errutil.Wrap(err, "Making login request")
	}

	token := tokenResp.Token
	if token == "" {
		return ret, fmt.Errorf("Returned token is empty. Check the server response to debug.")
	}

	ret = Client{
		Token:      token,
		httpClient: httpClient,
		baseURL:    baseURL,
	}

	return ret, nil
}

func (c Client) makeRequest(ctx context.Context, method string, path string, req interface{}, resp interface{}) error {

	// Path
	if path == "" {
		return fmt.Errorf("Path cannot be empty")
	}
	if c.baseURL == "" {
		return fmt.Errorf("Base URL is not set up")
	}
	url := filepath.Join(c.baseURL, path)

	// Body
	var httpReqBody io.ReadWriter
	if method == http.MethodPost || method == http.MethodPut {
		// Marshal the request body
		httpReqBody = bytes.NewBuffer(nil)
		err := json.NewEncoder(httpReqBody).Encode(req)
		if err != nil {
			return err
		}
	}
	httpReq, err := http.NewRequest(method, url, httpReqBody)
	if err != nil {
		return err
	}

	// Headers
	if c.Token == "" {
		return fmt.Errorf("Authorizarion token is not setup")
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	httpReq.Header.Set("Content-Type", "application/json")

	// Make the request
	log.Debug(ctx, "[Ongoku Client] HTTP request being made", "method", method, "url", url, "body", jsonhelper.MustPrettyPrint(req))

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return errutil.Wrap(err, "Making HTTP request")
	}

	// Decode the response
	defer httpResp.Body.Close()

	if resp != nil {
		err = json.NewDecoder(httpResp.Body).Decode(resp)
		if err != nil {
			return errutil.Wrap(err, "Decoding HTTP response body")
		}
	}

	log.Debug(ctx, "[Ongoku Client] HTTP response received", "method", method, "url", url, "response", resp)

	return nil
}

func MakeRequest(ctx context.Context, method string, url string, httpClient *http.Client, req interface{}, resp interface{}) error {

	// Path
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Body
	var httpReqBody io.ReadWriter
	if method == http.MethodPost || method == http.MethodPut {
		// Marshal the request body
		httpReqBody = bytes.NewBuffer(nil)
		err := json.NewEncoder(httpReqBody).Encode(req)
		if err != nil {
			return err
		}
	}
	httpReq, err := http.NewRequest(method, url, httpReqBody)
	if err != nil {
		return err
	}

	// Headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Make the request
	log.Debug(ctx, "[Ongoku Client] HTTP request being made", "method", method, "url", url, "body", jsonhelper.MustPrettyPrint(req))

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return errutil.Wrap(err, "Making HTTP request")
	}

	// Decode the response
	defer httpResp.Body.Close()

	if resp != nil {
		err = json.NewDecoder(httpResp.Body).Decode(resp)
		if err != nil {
			return errutil.Wrap(err, "Decoding HTTP response body")
		}
	}

	log.Debug(ctx, "[Ongoku Client] HTTP response received", "method", method, "url", url, "response", resp)

	return nil
}
