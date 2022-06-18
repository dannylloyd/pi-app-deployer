package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
)

func SendLogs(c LogForwardConfig, log config.Log) error {
	j, err := json.Marshal(log)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(j))

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

	fmt.Println(string(body))
	// err = json.Unmarshal(body, &apiRes)
	// if err != nil {
	// 	return err
	// }

	return nil
}
