# event-router

- rpm snapshot: https://artifactory-ptx.ecd.axway.int/artifactory/sentineleventrouter-generic-snapshot-ptx/qlt-router/qlt-router-$BRANCH.rpm [https://artifactory-ptx.ecd.axway.int/artifactory/sentineleventrouter-generic-snapshot-ptx/qlt-router/qlt-router-master.rpm](qlt-router-master.rpm)
- docker snapshot:  sentineleventrouter-docker-snapshot-ptx.artifactory-ptx.ecd.axway.int/qlt-router:$BRANCH


## Contribute

### Development Prerequisites

- go
- make
- docker / docker-compose
- gitlab-runner - for local testing

### build

```sh
make 
```

### test

unit-test (short)
```sh
go test --short --timeout 5s ./src/...
# or
gotestsum --junitfile report.xml --format testname --raw-command go test --short --timeout 5s --json ./src/...
```

all test (short + !short)
```sh
./scripts/run-integration-test-local.sh
# or
docker-compose -f docker-compose.external.yml
make test
# or
go test -v --timeout 10s ./src/...
# or
gotestsum --junitfile report.xml --format testname --raw-command go test --timeout 10s --json ./src/...
```

### local ci

- mac colima
```
export DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock"
```

```sh
./scripts/gitlab-runner.sh build
./scripts/gitlab-runner.sh rpm
./scripts/gitlab-runner.sh rpm-test
./scripts/gitlab-runner.sh unit-test
./scripts/gitlab-runner.sh integration-test

gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" build
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" rpm
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" test-rpm
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" integration-test
gitlab-runner exec docker --docker-volumes "/var/run/docker.sock:/var/run/docker.sock" --docker-volumes "$PWD/artefacts:/artefacts" --docker-volumes "$PWD/cache:/cache" build-docker
```

### add dev ui

```sh
./scripts/upkg.sh
```

### Manage dependencies

```sh
go list -u -m all # view available dependency upgrades
go get -u ./... # upgrade all dependencies at once
go get -t -u ./... # upgrade all dependencies at once (test dependencies as well)
go mod tidy
```

#### Check for vulnerabilities
```sh
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```


## Profiling

```sh
./event-router --cpuprofile event-router.prof
go tool pprof ./event-router event-router.prof

```

## Other tips

Remove colourful output
```sh
sed -e $'s/\x1b\[[0-9;]*m//g'
```

## Current coverage

[![Pipeline](https://git.ecd.axway.org/cft/qlt-router/badges/master/pipeline.svg)](https://git.ecd.axway.org/cft/qlt-router/)
[![Coverage](https://git.ecd.axway.org/cft/qlt-router/badges/master/coverage.svg)](https://git.ecd.axway.org/cft/qlt-router/)
[![Coverage](https://git.ecd.axway.org/api/v4/projects/7287/jobs/artifacts/master/raw/coverage.svg?job=integration-test)](https://git.ecd.axway.org/cft/qlt-router/)



## RPM

### Building
```sh
make
cp event-router artefacts/
./rpm/rpm-build.sh event-router 3.0.`date +%Y%m%d`
```

### Installing
Download package, to install run:
```sh
sudo rpm -i event-router-master.rpm 
```
This install 2 executables: event-router and event-routerd
When installing event-router via RPM, the binaries will be copied into /usr/bin/ and the event-router.conf will be created inside /usr/lib/event-router/.

Pour ubuntu
```sh
sudo alien -i event-router-master.rpm 
```
### Uninstalling

```sh
sudo rpm -qa | grep event-router
```
Use output in 
```sh
sudo rpm -ev event-router_3.0-<BN>-1
```
### Configuring
In order to run the Axway Event Router installed via RPM, you will need to create a runtime. To do so, use the command:
```sh
/usr/bin/event-router init <runtime-folder>
```

Modify the configuration file present in: \<runtime-folder\>/etc/event-router.conf to represent your usecase.

#### Test installation and configuration
##### Start
```sh
<runtime-folder>/usr/bin/event-router start
```
##### Stop
```sh
<runtime-folder>/usr/bin/event-router stop
```
##### Check logs
```sh
cat <runtime-folder>/var/log/event-router.log
```
#### Using systemd
To generate the systemd configuration file, run:

```sh
<runtime-folder>/usr/bin/event-router gen-systemd-unit
```
This will output to stdout the configuration template and the name of the file to use:
```sh
/etc/systemd/system/event-router.service
[Unit]
Description=Axway Event Router
After=network.target

[Service]
ExecStart=/usr/bin/event-routerd --config=<runtime-folder>/etc/event-router.conf
WorkingDirectory=<runtime-folder>
User=<user>
Type=simple

[Install]
WantedBy=default.target
```
You can add to this file as needed, then create the file /etc/systemd/system/event-router.service with the modified content.

Once you have your systemd unit file, you can test it:
```sh
sudo systemctl start event-router.service
```
Check the status of the service:
```sh
sudo systemctl status event-router.service
```
The service can be stopped or restart using standard systemd commands
```sh
sudo systemctl stop event-router.service
sudo systemctl restart event-router.service
```
Finally, when everything is working as expected, you can enable the service to make sure the service start whenever the system boots
```sh
sudo systemctl enable event-router.service
```
Then reboot the system and check the status of the Axway Event Router service:
```sh
sudo systemctl status event-router.service
```
```sh
● event-router.service - Axway Event Router
     Loaded: loaded (/etc/systemd/system/event-router.service; enabled; vendor preset: disabled)
     Active: active (running) since Wed 2024-02-14 15:35:20 CET; 23s ago
   Main PID: 904 (event-routerd)
      Tasks: 7 (limit: 23568)
     Memory: 23.9M
        CPU: 54ms
     CGroup: /system.slice/event-router.service
             └─904 /usr/bin/event-routerd --config=<runtime-folder>/etc/event-router.conf

Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792018+01:00 INF [] mem-writer-mem-writer Initializing Writer... --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792105+01:00 INF [] mem-writer-mem-writer Not Starting Writer Proxy Ack Loop ! (sync writer) --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792162+01:00 INF [] stream started -- name=qlt-sink desc='qlt-server-reader -[qlt-sink-reader]-> mem-writer'
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792212+01:00 INF [] main [HTTP] Setting up /metrics (prometheus)... --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792318+01:00 INF [] main [HTTP] Setting up /api... --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792362+01:00 INF [] main [HTTP] Setting up / (static)... --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.792394+01:00 INF [] main [HTTP] Listening on 0.0.0.0:8080 --
Feb 14 15:35:20 localhost event-routerd[904]: 2024-02-14T15:35:20.794842+01:00 INF [] mem-writer-mem-writer Running --
Feb 14 15:35:21 localhost event-routerd[904]: 2024-02-14T15:35:21.792525+01:00 INF [] main channel -- name=qlt-sink-reader size=0
Feb 14 15:35:21 localhost event-routerd[904]: 2024-02-14T15:35:21.792558+01:00 INF [] main channel -- name=mem-writer-mem-writerWriterAcks size=0
```


