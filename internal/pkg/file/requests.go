package file

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/manifest"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
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

	zip := fmt.Sprintf("%s/artifact.zip", dlDir)
	out, err := os.Create(zip)
	defer out.Close()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", ghApiToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	_, err = io.Copy(out, resp.Body)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return unzip(zip, dlDir)
}

func RenderTemplates(m manifest.Manifest, cfg config.Config, apiKey string) (config.ConfigFiles, error) {
	p := config.RenderTemplatesPayload{
		Config:   cfg,
		Manifest: m,
	}
	c := config.ConfigFiles{}
	postBody, _ := json.Marshal(p)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://pi-app-deployer.herokuapp.com/templates/render", bytes.NewBuffer(postBody))
	if err != nil {
		return c, err

	}
	req.Header.Add("api-key", apiKey)
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

	return c, nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
