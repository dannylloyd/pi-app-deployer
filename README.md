# pi-app-updater

It is annoying to update apps running on a Raspberry Pi. The app must be built locally using ARM configuratiuon, ssh/scp the files, restart services, etc. I wanted an automated deployment to the pi on new releases, or even on pushes to main. I want a generalized tool that handles checking for updates for a given Github repo. This tool can also handle first installation of the app. I want to ssh to a pi, use a one-line command to install and configure the pi-app-updater. It should prompt me for any environment variables/configuration. This tool implements all of these features.

TODO
- add log forwarder functionality
