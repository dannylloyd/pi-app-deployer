#!/bin/bash

set -euo pipefail

workDir="/home/runner/work/pi-app-deployer/pi-app-deployer"
deployerDir="/usr/local/src/pi-app-deployer"
mkdir -p ${deployerDir}

if [[ $(whoami) != "root" ]]; then
  echo "Script must be run as root"
  exit 1
fi

if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

if ! command -v jq &> /dev/null; then
  apt-get update
  apt-get install jq -y
fi

if ! command -v heroku &> /dev/null; then
  curl https://cli-assets.heroku.com/install-ubuntu.sh | sh
fi

if ! command -v redis-cli &> /dev/null; then
  curl -fsSL https://packages.redis.io/gpg | gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg

  echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/redis.list

  apt-get install redis -y >/dev/null
fi

export GH_API_TOKEN=$(heroku config:get GH_API_TOKEN -a ${DEPLOYER_APP})
export REDIS_URL=$(heroku config:get REDIS_URL -a ${DEPLOYER_APP})
redis-cli -u ${REDIS_URL} --scan --pattern "*andrewmarklloyd/pi-test*" | xargs --no-run-if-empty redis-cli -u ${REDIS_URL} del

export INVENTORY_TRANSIENT=true
mv ${workDir}/pi-app-deployer-agent ${deployerDir}
${deployerDir}/pi-app-deployer-agent install \
    --appUser runneradmin \
    --repoName andrewmarklloyd/pi-test \
    --manifestName pi-test-amd64 \
    --envVar MY_CONFIG=testing \
    --logForwarding \
    --herokuApp ${DEPLOYER_APP}

sed "s/{{.HerokuApp}}/${DEPLOYER_APP}/g" test/test-int-appconfigs.yaml > /tmp/test.yaml
grep "MY_CONFIG\=testing" ${deployerDir}/.pi-test-amd64.env >/dev/null
diff /tmp/test.yaml ${deployerDir}/.pi-app-deployer.config.yaml

sleep 10
journalctl -u pi-app-deployer-agent.service
journalctl -u pi-test-amd64.service
systemctl is-active pi-app-deployer-agent.service
systemctl is-active pi-test-amd64.service

# trigger an update
git config --global user.name "GitHub Actions Bot"
git config --global user.email "<>"
git clone https://github.com/andrewmarklloyd/pi-test.git
cd pi-test
git remote set-url origin https://andrewmarklloyd:${GH_API_TOKEN}@github.com/andrewmarklloyd/pi-test.git
deployerSHA=$(git rev-parse HEAD)
echo "Test run: ${deployerSHA};${DEPLOYER_APP}"
echo "${deployerSHA};${DEPLOYER_APP}" >> test/integration-trigger.txt
git add .
git commit -m "Pi App Deployer Test Run ${deployerSHA}"
sha=$(git rev-parse HEAD)
git push origin main

echo "Waiting for successful update of service"
i=0
found="false"
while [[ ${found} == "false" ]]; do
  if [[ ${i} -gt 20 ]]; then
    echo "Exceeded max attempts, test failed"
    echo "Logs from service: ${out}"
    cat /etc/systemd/system/pi-test-amd64.service
    exit 1
  fi
  out=$(journalctl -u pi-test-amd64.service -n 100)
  if [[ ${out} == *"${sha}"* ]]; then
    found="true"
  fi
  i=$((i+1))
  sleep 10
done

echo "Verifying deploy github action for pi-test was successful"
sleep 5 # give time for action to complete
runs=$(curl -s \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/andrewmarklloyd/pi-test/actions/runs)

conclusion=$(echo ${runs} | jq -r ".workflow_runs[] | select((.head_sha == \"${sha}\") and .name == \"Main Deploy\").conclusion")

if [[ ${conclusion} != 'success' ]]; then
    echo "Expected pi-test Main Deploy workflow run to be success, but got: ${conclusion}"
    exit 1
fi

${deployerDir}/pi-app-deployer-agent uninstall \
    --repoName andrewmarklloyd/pi-test \
    --manifestName pi-test-amd64 \
    --herokuApp ${DEPLOYER_APP}

sleep 5
journalctl -u pi-app-deployer-agent.service
systemctl is-active pi-app-deployer-agent.service
out=$(systemctl list-units -all | grep pi-test-amd64.service >/dev/null)
if [[ ! -z ${out} ]]; then
  echo "Expected pi-test-amd64 systemd unit to NOT exist but was found: ${out}"
  exit 1
fi

# test updating the agent itself is successful?

echo "Successfully ran integration tests! Now update this to use Go testing :)"
