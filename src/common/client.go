package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/opslevel/opslevel-go"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Client represents a rest http client and is used to send requests to OpsLevel integrations
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// ClientOption modifies fields on a Client
type ClientOption func(c *Client)

func NewGraphClient(version string) *opslevel.Client {
	client := opslevel.NewClient(viper.GetString("api-token"), opslevel.SetURL(viper.GetString("api-url")), opslevel.SetUserAgentExtra(fmt.Sprintf("cli-%s", version)))

	clientErr := client.Validate()
	if clientErr != nil {
		if strings.Contains(clientErr.Error(), "Please provide a valid OpsLevel API token") {
			cobra.CheckErr(fmt.Errorf("%s via 'export OPSLEVEL_API_TOKEN=XXX' or '--api-token=XXX'", clientErr.Error()))
		} else {
			cobra.CheckErr(clientErr)
		}
	}
	cobra.CheckErr(clientErr)

	return client
}

// NewClient returns a Client pointer
func NewRestClient(opts ...ClientOption) *Client {
	baseURL, _ := url.Parse("https://app.opslevel.com")
	client := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
	for _, o := range opts {
		o(client)
	}
	return client
}

// WithBaseURL modifies the Client baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		bu, _ := url.Parse(baseURL)
		c.baseURL = bu
	}
}

// WithHTTPClient modifies the Client http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func (c *Client) Do(method string, contentType string, url string, body []byte, recv interface{}) error {
	var err error

	log.Debug().Msgf("%s\n%s", url, string(body))
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", contentType)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Debug().Msgf("Failed to send request to OpsLevel: %s", err.Error())
		return err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	log.Debug().Msgf("Received status code %d", resp.StatusCode)
	if resp.StatusCode != http.StatusAccepted {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		s := buf.String()
		return fmt.Errorf("status %d; %s", resp.StatusCode, s)
	}

	err = decoder.Decode(&recv)
	if err != nil {
		log.Debug().Msgf("Failed to decode response from OpsLevel: %s", err.Error())
		return err
	}
	return nil
}
