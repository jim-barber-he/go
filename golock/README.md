
# golock

## Introduction

golock uses a central Redis server to globally coordinate the running of a job across a distributed system.  
This is useful where you only want a job to run once, but want to schedule it on multiple servers for redundancy.

Also by only allowing one job to run at a time, it can stop jobs that are scheduled to frequently run, from overlapping
themselves.

It is a cronlock replacement, based on https://github.com/kvz/cronlock  
It creates locks with the same names as cronlock does, and uses many of the same environment variables.

The purpose of writing golock is because cronlock does not support Redis servers where TLS is enforced.
And also it is an exercise in me learning Golang.

## Options.

- `CRONLOCK_HOST` the Redis hostname. default: `localhost`
- `CRONLOCK_PORT` the Redis port. default: `6379`
- `CRONLOCK_AUTH` the Redis auth password. default: Not present
- `CRONLOCK_DB` the Redis database. default: `0`
- `CRONLOCK_REDIS_TIMEOUT` the length of time to wait for a response from Redis before considering it in an errored state.
  This ensures that if the Redis connection goes away that we don't wait forever waiting for a response. default: `30`
- `CRONLOCK_GRACE` determines how many seconds a lock should at least persist.
  Prevents fast running jobs scheduled on multiple servers with some clock drift, from executing multiple times.
- `CRONLOCK_RELEASE` determines how long a lock can persist at most.
  Acts as a failsafe so there can be no locks that persist forever in case of failure. default is a day: `86400`
- `CRONLOCK_RECONNECT_ATTEMPTS` the number of times to try to reconnect before erroring.
  If the Redis connection is closed, attempt to reconnect upto this amount of times. default: `5`
- `CRONLOCK_RECONNECT_BACKOFF` the length of time to increase the wait between reconnects.
  Acts as a failsafe to allow Redis to be started before trying to reconnect.
  Set to 0 to retry the connection immediately. default: `5`
- `CRONLOCK_KEY` a unique key for this command in the global Redis server. default: an md5 hash of golock's arguments.
- `CRONLOCK_PREFIX` Redis key prefix used by all keys. default: `cronlock`
- `CRONLOCK_VERBOSE` set to `yes` to print debug messages. default: `no`
- `CRONLOCK_TIMEOUT` how long the command can run before it gets issued a `kill -9`. default: `0`; no timeout
- `CRONLOCK_TLS` use TLS to connect to Redis. default `false`
- `CRONLOCK_TLS_SKIP_VERIFY` donot verify TLS certificates when using TLS connections. default `false`;
  certificates are verified.
- `CRONLOCK_RESET` removes the lock and exits immediately. Needs to golock arguments passed in order to remove the right lock.

## Exit Codes

- = `200` Success (delete succeeded or lock not acquired, but normal execution)
- = `201` Failure (cronlock error)
- = `202` Failure (cronlock timeout)
- < `200` Success (acquired lock, executed your command), passes the exit code of your command

## Examples

### Single server

```
* * * * *  golock command.sh
```
The above crontab entry launches `command.sh` every minute.
If the previous `command.sh` has not finished yet, another is not started.
This works on one server since the default `CRONLOCK_HOST` of `localhost` is used.

### Distributed

```
00 08 * * * CRONLOCK_HOST=redis.example.com golock command.sh
```
In this configuration, a central Redis server is used to track the locking for `command.sh`.
If this crontab entry was on many servers, just one instance of `command.sh` would be run each morning.

### Lock commands that have different arguments

By default golock uses your command and its arguments to make a unique identifier by which the global lock is acquired.
However if you want to run `ls -al` or `ls -a`, but just one instance of either, you'll need to provide your own key:
```
# One of two will be executed because they share the same KEY
* * * * * CRONLOCK_KEY="ls" golock ls -al
* * * * * CRONLOCK_KEY="ls" golock ls -a
```

### Per application

If you use the same command and Redis server for multiple applications and you need them to run without impacting each other,
use the `CRONLOCK_PREFIX`:
```
# Crontab for app1
* * * * * CRONLOCK_PREFIX="lock.app1." cronlock command.sh
```
```
# Crontab for app2
* * * * * CRONLOCK_PREFIX="lock.app2." cronlock command.sh
```
Now `command.sh` will be able to run for each application at the same time, because even though `command.sh` will produce the same
md5 hash for both, the key's prefix makes it unique.

## Local Redis for testing

A quick way of getting a Redis server locally if you have Docker available is to run:
```shell
docker run --rm -it -p 6379:6379 redis:latest
```
