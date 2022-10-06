# TSM
Project IDA TriStar SNMP Monitoring Application

### Install GO
__On FBSD system as user *nrts*:__

* mkdir -p build/go
* cd build/go
* curl -L -o go1.18.1.freebsd-386.tar.gz https://go.dev/dl/go1.18.1.freebsd-386.tar.gz
_Golang v18.1 current as of this writing_

__as *root*:__

* cd ~nrts/build/go
* tar -C /usr/local -xzf go1.18.1.freebsd-386.tar.gz

__as user *nrts*:__

* Check ~nrts/.pathrc for `set MyPath = ($MyPath /usr/local/go/bin)` and if not there, add before the line `set path = ($MyPath $path)`.
* log out (`exit`)
* log back in
* Check go installation by viewing the go version: `go version`. You should see: *go version go1.18.1 freebsd/386*

### Get TSM Source from a Release on Github and Build
You must use a Personal Access Token from your GitHub account
* cd ~/build
* set TOKEN = "_your token goes here in quotes"
#### _This example would download and build release version 1.2 created on Github_
* curl -sL --header "Authorization: token $TOKEN" --header 'Accept: application/octet-stream' https://github.com/ProjectIDA/tsm/archive/refs/tags/v1.2.tar.gz -o tsm.v1.2.tar.gz
* tar xvf tsm.v1.2.tar.gz
* cd tsm-1.2
* go build
* _if ~/bin/tsm exists and you are upgrading, then_ `chmod 755 ~/bin/tsm`
* cp tsm.toml ~/etc
* cd ~/etc
* Edit ~/etc/tsm.toml and set correct station code (uppercase) at top of file

### TODO
*Convert to Cobra CLI framework
*Implement MODBUS write functionality since Morningstar does not support SNMP writes