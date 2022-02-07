package heroku

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HerokuClient struct {
	AppName string
	APIKey  string
}

type HerokuRes struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

func NewClient(appName string, apiKey string) HerokuClient {
	return HerokuClient{
		AppName: appName,
		APIKey:  apiKey,
	}
}

func (c *HerokuClient) GetEnv() (map[string]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.heroku.com/apps/%s/config-vars", c.AppName), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res HerokuRes
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}
	if res.Id == "unauthorized" || res.Id == "forbidden" {
		return nil, fmt.Errorf("error from heroku: %s", res.Message)
	}

	m := make(map[string]string)
	err = json.Unmarshal(body, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
