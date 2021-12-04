# pi-app-updater

It is annoying to update apps running on a Raspberry Pi. The app must be built locally using ARM configuratiuon, ssh/scp the files, restart services, etc. I wanted an automated deployment to the pi on new releases, or even on pushes to main. I want a generalized tool that handles checking for updates for a given Github repo. This tool can also handle first installation of the app. I want to ssh to a pi, use a one-line command to install and configure the pi-app-updater. It should prompt me for any environment variables/configuration. This tool implements all of these features.

## Features
- creates systemd unit
- allows specifying environemnt variables
- supports additional commands to be run for setup before app runs

## Usage

Use the following arguments with the script `./pi-app-updater --repo-name <repo-name> --package-name <package-name> [--install] [--uninstall]`.

`--repo-name <repo-name>`
- Name of the Github repo including the owner. For example `andrewmarklloyd/pi-app-updater`. The updater will use the full name in the API call to get the latest release and check if a newer version is available. For example given the repo name above, the API URL that the updater will check is `https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest`

`--package-name <package-name>`
- Name of the release tar.gz file to install.

`[--install]`
- `Optional`: Indicates the script should run the first time installation for an app.

`[--uninstall]`
- `Optional`: Disables systemd unit and removes all files.

## Requirements
1. Add `.pi-app-updater.yaml` to the root of the repo

    ```
    name: <name-of-app>
    heroku:
      app: <heroku-app-name>
      env:
      - <list of environment vars>
    ```

1. Create Heroku account and get API key to dynamically lookup env vars in a secure location. Uses Heroku app's environment config get all env vars. Create a Heroku app and save name of app.

1. Create github action in your repo: `.github/workflows/release.yml`

    ```
    on: 
    release:
        types: [created]

    jobs:
    releases-matrix:
        name: Release Go Binary
        runs-on: ubuntu-latest
        strategy:
        matrix:
            goos: [linux]
            goarch: [arm]
        steps:
        - uses: actions/checkout@v2
        - uses: wangyoucao577/go-release-action@v1.18
        with:
            github_token: ${{ secrets.GITHUB_TOKEN }}
            goos: ${{ matrix.goos }}
            goarch: ${{ matrix.goarch }}
            project_path: "./"
            binary_name: "<insert-binary-name>"
            extra_files: .pi-app-updater.yaml

    ```

1. Create a github release


## TODO
- uninstall
- refactor
- package functions need improvement
