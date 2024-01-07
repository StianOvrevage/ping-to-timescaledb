# Network pinger to TimescaleDB

Simple program that pings multiple hosts and saves results to TimescaleDB for analysis.

By sending packets at a fixed interval regardless of outstanding requests it tries to avoid the [Coordinated Omission](http://highscalability.com/blog/2015/10/5/your-load-generator-is-probably-lying-to-you-take-the-red-pi.html) problem. *

## Usage

_See below how to run and set up TimecaleDB._

    git clone https://github.com/StianOvrevage/ping-to-timescaledb # Download files 
    cd ping-to-timescaledb
    go mod tidy # Download dependencies
    export PING_TIMESCALEDB_CONNSTR=postgres://myuser:mypassword@timescaledb:5432/mydatabase
    sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
    go run main.go

> Assumes you already have [golang installed](https://go.dev/doc/install).

Optional configuration:

    # Ping each host every interval. Regardless of whether there are outstanding requests.
    export PING_INTERVAL=200ms
    # Timeout. If requests time out their latency will be set to this number, and the `timeout` field in DB will be true.
    export PING_TIMEOUT=2s
    # Comma separated list of destination hosts to ping
    export PING_DESTINATIONS=www.vg.no,192.168.1.1,test.hkg1.servers.com

The program runs in the foreground forever. You can use `screen` to run it in the background after disconnection from a terminal session.

## TimescaleDB preparation

Run TimescaleDB as a container:

    docker run -d --name timescaledb -p 5432:5432 \
      -e POSTGRES_PASSWORD=myPassword timescale/timescaledb-ha:pg16-all

> PS: This is without persistence since I'm strugling how to get volume mounts and postgres to work in podman.

Connect to the database with `psql` or [pgAdmin](https://www.pgadmin.org/)

Create a new database (`mydatabase` above). Create a new user (`myuser` above) with a password (`mypassword` above) and permission to log in.

Create the table and hypertable:

    CREATE TABLE pings (
      time TIMESTAMPTZ NOT NULL,
      from_host TEXT NOT NULL,
      to_host TEXT NOT NULL,
      interface TEXT NOT NULL, -- Not implemented yet
      error TEXT, -- Not implemented yet
      timeout BOOL NOT NULL DEFAULT FALSE,
      latency_ms DOUBLE PRECISION NULL
    );

    SELECT create_hypertable('pings', by_range('time'));

Ensure that the new user has permissions to the database and table.

Optionally create a read-only user for the table to use with Grafana.


## Caveats

If we are writing data to TimescaleDB close to capacity and there are bursts of requests that suddenly finish, we may not be fast enough to flush the results. If that condition lasts long enough for the result collection queue (size 1000 results) to fill up we will have a coordinated omission problem. Even though I believe this is unlikely the program will print the current size of the queue every 5 seconds.

## Background

I've had some issues with my wireless Hi-Fi (KEF LS50 & System Audio Focus SA-5).

Intermittently there will be random issues with noise on one or both speakers. Speakers becoming unavailable to Spotify Connect for a little while. And so on.

The issue has persisted between changing my wireless setup three times. I'm now on Unify Amplify Mesh. And moving between three different places, first an appartment complex and now in a free standing house. And two separate Hi-Fi systems are having issues.

I always have my trusty `ping -t vg.no` running in the background and have noticed that network reliability isn't as good as it should be. What if this is related?

I'm not too keen on spending even more money on this by investing in a [Wi-Fi spectrum analyzer](https://shop.metageek.com/products/wipry-clarity-by-oscium) that costs from $1.000 and infinitely upwards.

Can we just fire away a bunch of pings (at a steady rate to avoid the coordinated omission proble) from all of my machines (Intel NUC server, wired. Desktop, wireless. Laptop, wireless) and see any correlation when the sound is having problems?

At least it could verify that I'm not going insane! If I find something I can start the process of elimination turning of and disconnecting everything and use the ping data as guidance instead of me intermittently listening ot music waiting for intermittent problems, which would take months.

> I also wanted to collect Wi-Fi information but I haven't been able to get the Intel NUC Wi-Fi adapter to cooperate.
