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

func NewClient(appName string, apiKey string) (HerokuClient, error) {
	return HerokuClient{
		AppName: appName,
		APIKey:  apiKey,
	}, nil
}

func (c *HerokuClient) GetEnv() error {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.heroku.com/apps/%s/config-vars", c.AppName), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var res HerokuRes
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	fmt.Println(res)
	return nil
}
