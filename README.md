# pi-app-updater

It is annoying to update apps running on a Raspberry Pi. The app must be built locally using ARM configuratiuon, ssh/scp the files, restart services, etc. I wanted an automated deployment to the pi on new releases, or even on pushes to main. I want a generalized tool that handles checking for updates for a given Github repo. This tool can also handle first installation of the app. I want to ssh to a pi, use a one-line command to install and configure the pi-app-updater. It should prompt me for any environment variables/configuration. This tool implements all of these features.

## Usage

Use the following arguments with the script `./pi-app-updater --full-repo-name <full-repo-name> --update-script <update-script> --install-script <install-script>`.

`<full-repo-name>`
- Name of the Github repo including the owner. For example `andrewmarklloyd/pi-app-updater`. The updater will use the full name in the API call to get the latest release and check if a newer version is available. For example given the repo name above, the API URL that the updater will check is `https://api.github.com/repos/andrewmarklloyd/pi-sensor/releases/latest`

`<update-script>`
- Absolute path of the script or executable that is responsible for running the update actions for a given app. This could be downloading attachments to the release, updating configuration, etc.
- Requirements: command line argument 1 passed to the script is the latest release version as set in Github, for example `v0.1.5`. It is expected that consumers of this app use this version to implement all of the update logic for the app. Any exit code other than 0 will be considered a failure and on the next retry the updater will attempt to run the script again.

`<install-script>`
- Absolute path of the script or executable that is responsible for running the installing actions for a given app. This could be downloading attachments to the release, updating configuration, etc. Note this script will be called on first run of the updater.
- Requirements: command line argument 1 passed to the script is the latest release version as set in Github, for example `v0.1.5`. It is expected that consumers of this app use this version to implement all of the install logic for the app. Any exit code other than 0 will be considered a failure and on the next retry the updater will attempt to run the script again.
