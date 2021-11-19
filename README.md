# pi-app-updater


It's annoying to update apps running on the pi. I have to build it locally using arm configuratiuon, ssh/scp files, restart services, etc. I want an automated deployment to the pi on new releases, or even on pushes to main. I want a generalized tool that handles checking for updates for a given Github repo. This tool can also handle first installation. I want to ssh to a pi, use a one-line command to install and configure the pi-app-updater. It should prompt me for any environment variables/configuration.


Implementation:
- Polling: store the current running version of the app in a file. Either using golang cron or systemd cron, periodically check the github releases api for newer versions, and update if one exists.
    - Most simple solution but not as interesting. 
- Event based: don't want to expose port on my router, need a separate service to send webhook events to and publish to an MQTT queue. The updater subscribes to new releases and runs accordingly.
    - Much more complex but interesting. Need to write a separate service for webhook handling. Need to ensure no messages missed from queue.

