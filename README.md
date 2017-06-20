# BOOSTER-WEB: Web interface to [BOOSTER](http://booster.c3bi.pasteur.fr)
This interface presents informations about BOOSTER program, and allows to run BOOSTER easily.

# Installing BOOSTER-WEB
## Already compiled
Download a release in the [release](https://github.com/fredericlemoine/booster-web/releases) section. You can directly run the executable for your platform.

## From source
To compile BOOSTER-WEB, you must [download](https://golang.org/dl/) and [install](https://golang.org/doc/install) Go on your system.

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
  * galaxykey="[galaxy api key]"
  * galaxyurl="[url of the galaxy server: http(s)://ip:port]"
  * queuesize=[size of job queue]
  * nbrunners=[number of parallel local runners]
  * jobthreads=[number of threads per local job]
  * timeout=[job timeout in seconds: 0=ulimited]
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
type = "mysql"
user = "myuser"
port = 3306
host = "localhost"
pass = "mypass"
dbname = "mydbname"

[runners]
type="galaxy"
galaxykey="dsjfkhsdfhdjfhdjfhsdkjfh"
galaxyurl="http://url:80"
queuesize = 10
nbrunners  = 2
jobthreads  = 5
timeout  = 3600

[logging]
logfile = "stderr"

[http]
port = 4000
```
