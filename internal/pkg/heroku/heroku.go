package heroku

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HerokuClient struct {
	APIKey string
}

func NewHerokuClient(apiKey string) HerokuClient {
	return HerokuClient{
		APIKey: apiKey,
	}
}

func (c *HerokuClient) GetEnvVars(herokuApp string) (map[string]string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.heroku.com/apps/%s/config-vars", herokuApp), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request to heroku: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading respone body from heroku: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response from heroku, receieved status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var envVars map[string]string
	err = json.Unmarshal(body, &envVars)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling response body from heroku: %s", err)
	}

	return envVars, nil
}
