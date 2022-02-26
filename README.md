# pi-app-updater

It is annoying to update apps running on a Raspberry Pi. The app must be built locally using ARM configuratiuon, ssh/scp the files, restart services, etc. I wanted an automated deployment to the pi on new releases, or even on pushes to main. I want a generalized tool that handles checking for updates for a given Github repo. This tool can also handle first installation of the app. I want to ssh to a pi, use a one-line command to install and configure the pi-app-updater. It should prompt me for any environment variables/configuration. This tool implements all of these features.


## Agent Install
Requires several environment variables:
- GH_API_TOKEN: A GitHub App's API token with the `Actions` read-only scope. Required to download artifacts from Github.
- HEROKU_API_KEY: Heroku API key with read access to the app that contains the application's secrets
- CloudMQTT variables: Required to receive update events
    - CLOUDMQTT_AGENT_USER: Heroku provisioned cloudmqtt's read-only user
    - CLOUDMQTT_AGENT_PASSWORD: Heroku provisioned cloudmqtt's read-only password
    - CLOUDMQTT_URL: Heroku provisioned cloudmqtt's read-only domain and port
- PI_APP_UPDATER_API_KEY: Pi App Updater server's API key. Required for posting and getting data from the server.

```
bash <(curl -s -H 'Cache-Control: no-cache' "https://raw.githubusercontent.com/andrewmarklloyd/pi-app-updater/master/install/install-agent.sh?$(date +%s)")
```

TODO
- add log forwarder functionality
