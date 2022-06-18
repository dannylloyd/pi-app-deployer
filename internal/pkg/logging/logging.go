package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
)

type response struct {
	Error string `json:"error"`
}

func SendLogs(c LogForwardConfig, log config.Log) error {
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer([]byte(log.Message)))

	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("api-key", c.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf(res.Error)
	}

	return nil
}
