# BOOSTER-WEB: Web interface to [BOOSTER](http://booster.c3bi.pasteur.fr)
This interface presents informations about BOOSTER program, and allows to run BOOSTER easily.

# Installing BOOSTER-WEB
## Already compiled
Download a release in the [release](https://github.com/fredericlemoine/booster-web/releases) section. You can directly run the executable for your platform.

## From source
To compile BOOSTER-WEB, you must [download](https://golang.org/dl/) and [install](https://golang.org/doc/install) Go (version >=1.9) on your system.

Then you just have to type :
```
go get github.com/fredericlemoine/booster-web/
```
This will download BOOSTER-WEB sources from github, and its dependencies.

If you cloned the repository and want to install dependencies manually:

```bash
go get github.com/jteeuwen/go-bindata/...
go get github.com/elazarl/go-bindata-assetfs/...
go get github.com/dgrijalva/jwt-go
go get github.com/fredericlemoine/golaxy
go get github.com/fredericlemoine/gotree
go get github.com/go-sql-driver/mysql
```

You can then build BOOSTER-WEB with:
```
cd $GOPATH/src/github.com/fredericlemoine/booster-web/
make
```

The `booster-web` executable should be located in the `$GOPATH/bin` folder.

# Running BOOSTER-WEB
## Default configuration
You can directly run the `booster-web` executable without any configuration. It will setup a web server with the following default properties:
* Run on localhost, port 8080
* Log to stderr
* In memory database (analyses will not persist after server shutdown)
* Local processor (booster jobs will run on the local machine)
* 1 parallel Runner: One job at a time
* Job Timeout: unlimited
* 1 thread per job

To access the web interface, just go to [http://localhost:8080](http://localhost:8080)

## Other configurations
It is possible to configure `booster-web` to run with specific options. To do so, create a configuration file `booster-web.toml` with the following sections:
* database
  * type = "[memory|mysql]"
  * user = "[mysql user]"
  * port = [mysql port]
  * host = "[mysql host]"
  * pass = "[mysql pass]"
  * dbname = "[mysql dbname]"
* runners
  * type="[galaxy|local]"
  * queuesize=[size of job queue]
  * nbrunners=[number of parallel local runners]
  * jobthreads=[number of threads per local job]
  * timeout=[job timeout in seconds: 0=ulimited]
* galaxy (Only used if runners.type="galaxy")
  * key="[galaxy api key]"
  * url="[url of the galaxy server: http(s)://ip:port]"
* galaxy.tools
  * booster="[Id of booster tool on the galaxy server]"
  * phyml="[Id of PHYML-SMS tool on the galaxy server]"
  * fasttree="[Id of FastTree tool on the galaxy server]"
* notification (for notification when jobs are finished)
  * activated=[true|false]
  * smtp="[smtp serveur for sending email]"
  * port=[smtp port]
  * user="[smtp user]"
  * pass="[smtp password]"
  * resultpage = "[url to result pages]"
  * sender="[sender of the notification]"
* logging
  * logfile= "[stderr|stdout|/path/to/logfile]"
* http
  * port=[http server listening port]
* authentication
  * user="[global username]"
  * password="[global password]"

And run booster web: `booster-web --config booster-web.toml`

## Example of configuration file
```
[database]
# Type : memory|mysql (default memory)
type = "mysql"
user = "mysql_user"
port = 3306
host = "mysql_server"
pass = "mysql_pass"
dbname = "mysql_db_name"

[runners]
# galaxy|local if galaxy: required galaxykey & galaxyurl
type="galaxy"
# Maximum number of pending jobs (default : 10): for galaxy & local
queuesize = 200
# Number of parallel running jobs (default : 1): for local only
nbrunners  = 1
# Number of cpus per bootstrap job : for local only
jobthreads  = 10
# Timout for each job in seconds (default unlimited): for local only
#timeout  = 10
# Keep old finished analyses for 10 days, default=0 (unlimited)
keepold = 10

#Only used if runners.type="galaxy"
[galaxy]
key="galaxy_api_key"
url="https://galaxy.server.com/"

[galaxy.tools]
# Id of booster tool on the galaxy server
booster="/.../booster/booster/version"
# Id of PhyML-SMS tool on the galaxy server
phyml="/.../phyml-sms/version"
# Id of FastTree tool on the galaxy server
fasttree="/.../fasttree/version"

# For notification when job is finished
[notification]
# true|false
activated=true
# smtp serveur for sending email
smtp="smtp.serveur.com"
# Port
port=587
# Smtp user 
user="smtp_user"
# Smtp password
pass="smtp_pass"
# booster-web server name:port/view page,
# used to give the right url in result email
resultpage = "http://url/view"
# sender of the notification
sender = "sender@server.com"

[logging]
# Log file : stdout|stderr|any file
logfile = "booster.log"

[http]
# HTTP server Listening port
port = 4000

# For running a private server, default: no authentication
#[authentication]
#user     = "user"
#password = "pass"
```
