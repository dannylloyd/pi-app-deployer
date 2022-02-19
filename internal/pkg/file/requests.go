package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/andrewmarklloyd/pi-app-updater/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-updater/internal/pkg/config"
)

func DownloadExtract(url, dlDir, ghApiToken string) error {
	err := os.RemoveAll(dlDir)
	if err != nil {
		return fmt.Errorf("removing download directory: %s", err)
	}
	err = os.Mkdir(dlDir, 0755)
	if err != nil {
		return fmt.Errorf("creating download directory: %s", err)
	}

	authHeader := fmt.Sprintf("Authorization: token %s", ghApiToken)
	var tarOut bytes.Buffer
	curl := exec.Command("curl", "-sL", "-H", authHeader, url)
	tar := exec.Command("tar", "xz", "-C", dlDir)
	tar.Stdout = &tarOut
	tar.Stdin, err = curl.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating curl stdout pipe: %s", err)
	}
	err = tar.Start()
	if err != nil {
		return fmt.Errorf("starting tar command: %s", err)
	}
	err = curl.Run()
	if err != nil {
		return fmt.Errorf("running curl command: %s", err)
	}
	err = tar.Wait()
	if err != nil {
		return fmt.Errorf("waiting on tar command: %s", err)
	}
	return nil
}

func RenderTemplates(m manifest.Manifest) (config.ConfigFiles, error) {
	c := config.ConfigFiles{}
	postBody, _ := json.Marshal(m)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://pi-app-updater.herokuapp.com/templates/render", bytes.NewBuffer(postBody))
	if err != nil {
		return c, err

	}
	req.Header.Add("api-key", os.Getenv("PI_APP_UPDATER_API_KEY"))
	resp, err := client.Do(req)
	if err != nil {
		return c, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c, err
	}
	defer resp.Body.Close()

	err = json.Unmarshal(data, &c)

	fmt.Println(c)
	return c, nil
}

func DownloadDirectory(packageName string) string {
	return fmt.Sprintf("/tmp/%s", packageName)
}
