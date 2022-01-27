.PHONY: build

build:
	GOOS=linux GOARCH=arm GOARM=5 go build -o pi-app-updater
