/*
golock uses a central Redis server to globally coordinate the running of a job across a distributed system.
This is useful where you only want a job to run once, but want to schedule it on multiple servers for redundancy.

Also by only allowing one job to run at a time, it can stop jobs that are scheduled to frequently run, from overlapping
themselves.

It is a cronlock replacement, based on https://github.com/kvz/cronlock
It creates locks with the same names as cronlock does, and uses many of the same environment variables.
*/
package main

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jim-barber-he/go/util"
	redis "github.com/redis/go-redis/v9"
)

const (
	// Default Values.
	defaultRedisPort              int = 6379
	defaultRedisReconnectAttempts int = 5
	defaultRedisTimeout           int = 30
	defaultRedisReconnectBackoff  int = 5
	defaultLockGrace              int = 40
	defaultLockRelease            int = 86400

	/* Exit codes:
	< 200 : Acquired lock; executed command; returned exit code of the command.
	= 200 : Success. Delete succeeded OR lock not acquired, but normal execution.
	= 201 : Failure. Error encountered.
	= 202 : Failure. Lock timed out.
	*/
	golockSuccess int = 200
	golockFailure int = 201
	golockTimeout int = 202
)

var ctx = context.Background()

func redisConnect(connOpts *redis.Options) (*redis.Client, error) {
	slog.Debug("Connecting to redis at " + connOpts.Addr)

	rdb := redis.NewClient(connOpts)

	response, err := rdb.Ping(ctx).Result()
	switch {
	case err != nil:
		return nil, fmt.Errorf("Could not connect to Redis: %v", err)
	case response != "PONG":
		return nil, fmt.Errorf("Could not ping Redis: %s", response)
	}

	return rdb, nil
}

func run() int {
	// Command to run and its arguments represented as a string.
	command := strings.Join(os.Args[1:], " ")

	// Redis host and port.
	redisHost := util.GetEnv("CRONLOCK_HOST", "localhost")
	redisPort := util.GetEnvInt("CRONLOCK_PORT", defaultRedisPort)

	// Redis database to connect to.
	redisDB := util.GetEnvInt("CRONLOCK_DB", 0)

	// Redis TLS options.
	redisTLS := util.GetEnvBool("CRONLOCK_TLS", false)
	redisTLSSkipVerify := util.GetEnvBool("CRONLOCK_TLS_SKIP_VERIFY", false)

	// Length of time to wait for a response from Redis before considering it in an errored state.
	// Prevents waiting forever for a response from Redis.
	redisTimeout := time.Duration(util.GetEnvInt("CRONLOCK_REDIS_TIMEOUT", defaultRedisTimeout)) * time.Second

	// Number of times to try to reconnect to Redis before erroring.
	redisReconnectAttempts := util.GetEnvInt("CRONLOCK_RECONNECT_ATTEMPTS", defaultRedisReconnectAttempts)

	// Length of time to increase the wait between Redis reconnects.
	// Acts as a failsafe to allow Redis to be started before trying to reconnect.
	// Set to 0 to retry the connection immediately.
	redisReconnectBackoff := time.Second * time.Duration(
		util.GetEnvInt("CRONLOCK_RECONNECT_BACKOFF", defaultRedisReconnectBackoff),
	)

	// How many seconds a lock should at least persist.
	// Makes sure that fast running jobs will not execute many times if run from multiple servers with clock drift.
	// Recommend using a grace of at least 30s.
	lockGrace := util.GetEnvInt("CRONLOCK_GRACE", defaultLockGrace)

	// Determines how long a lock can persist at most.
	// Acts as a failsafe so there can be no locks that persist forever in case of failure.
	// Shouldn't be less than CRONLOCK_GRACE.
	lockRelease := util.GetEnvInt("CRONLOCK_RELEASE", defaultLockRelease)

	// Prefix used by all Redis keys set by this program.
	lockPrefix := util.GetEnv("CRONLOCK_PREFIX", "cronlock.")

	// Unique key for this command in the global Redis server.
	// If not set, then one is calculated based on the MD5 hash of the command and its arguments.
	redisKey := os.Getenv("CRONLOCK_KEY")
	if redisKey == "" {
		hash := md5.Sum([]byte(command))
		redisKey = hex.EncodeToString(hash[:])
	}
	redisKey = lockPrefix + redisKey

	// Remove existing lock in Redis and then exit.
	reset := util.GetEnv("CRONLOCK_RESET", "no")

	// How long the command can run in seconds before it is killed.
	timeout := util.GetEnvInt("CRONLOCK_TIMEOUT", 0)

	// Configure connection to Redis.
	connOpts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", redisHost, redisPort),
		DB:              redisDB,
		DialTimeout:     redisTimeout,
		MaxRetries:      redisReconnectAttempts,
		MaxRetryBackoff: redisReconnectBackoff,
		MinRetryBackoff: redisReconnectBackoff,
	}
	if auth := os.Getenv("CRONLOCK_AUTH"); auth != "" {
		connOpts.Password = auth
	}
	if redisTLS {
		tlsOpts := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if redisTLSSkipVerify {
			tlsOpts.InsecureSkipVerify = redisTLSSkipVerify
		}
		connOpts.TLSConfig = tlsOpts
	}

	// Connect to Redis.
	rdb, err := redisConnect(connOpts)
	if err != nil {
		slog.Error(err.Error())
		return golockFailure
	}
	defer rdb.Close()

	// Reset mode. Remove the lock and exit.
	if reset == "yes" {
		slog.Debug(fmt.Sprintf("Removing %s key", redisKey))
		removed, err := rdb.Del(ctx, redisKey).Result()
		if err != nil {
			slog.Error(err.Error())
			return golockFailure
		}
		if removed == 0 || removed == 1 {
			return golockSuccess
		}
		slog.Error("Unable to remove " + redisKey)
		return golockFailure
	}

	// Times that the lock will be completed.
	// expireAtMax is used when the lock is acquired to set the longest time we want to keep it for.
	// expireAtMin is used after the command has completed to expire the lock, but it will only expire after it has
	// persisted long enough for the minimum grace period to have passed.
	expireAtMax := time.Now().UTC().Unix() + int64(lockRelease) + 1
	expireAtMin := time.Now().UTC().Unix() + int64(lockGrace) + 1

	// Acquire lock.
	slog.Debug(fmt.Sprintf("Acquiring lock on %s key", redisKey))
	acquired, err := rdb.SetNX(ctx, redisKey, expireAtMax, time.Duration(lockRelease)*time.Second).Result()
	if err != nil {
		slog.Error(err.Error())
		return golockFailure
	}

	if acquired {
		slog.Debug(fmt.Sprintf("Lock %s acquired", redisKey))
	} else {
		// Handle edge cases.

		expiresAt, err := rdb.Get(ctx, redisKey).Result()
		if err != nil {
			slog.Error(err.Error())
			return golockFailure
		}
		expiresIn, _ := strconv.Atoi(expiresAt)
		expiresIn -= int(time.Now().UTC().Unix())

		switch {
		case expiresIn > 0:
			slog.Debug(
				fmt.Sprintf(
					"Lock %s acquired by another process (expires in %ds)",
					redisKey,
					expiresIn,
				),
			)
			return golockSuccess
		case expiresIn == 0:
			slog.Debug(fmt.Sprintf("Lock %s acquired by another process but expiring now", redisKey))
			return golockSuccess
		}
		slog.Debug(
			fmt.Sprintf("Lock %s acquired by another process but expired %ds ago", redisKey, -expiresIn),
		)

		// Handle expired locks that were not cleaned up properly or not cleaned up yet because the golock that
		// requested it is still running.
		// Try to acquire a lock again, confirming that no other running golock beats us to it.
		reacquire, err := rdb.GetSet(ctx, redisKey, expireAtMax).Result()
		if err != nil {
			slog.Error(err.Error())
			return golockFailure
		}
		expiresIn, _ = strconv.Atoi(reacquire)
		expiresIn -= int(time.Now().UTC().Unix())
		if expiresIn > 0 {
			slog.Debug(
				fmt.Sprintf(
					"Lock %s was just now acquired by a different process (expires in %ds)",
					redisKey,
					expiresIn,
				),
			)
			return golockSuccess
		}
	}

	// Run command with an optional timeout.
	exitCode, _ := util.Run(timeout, os.Args[1], os.Args[2:]...)
	if timeout > 0 {
		if exitCode == util.ExitCodeProcessKilled {
			slog.Error(fmt.Sprintf("emergency: had to kill [%s] after %ds timeout", command, timeout))
			exitCode = golockTimeout
		}
	}

	// Command is complete. We can set the key to expire once the minimum grace period has passed.

	// Set the value of the key to the timestamp defined by the minimum grace period.
	// This is for the benefit of other instances of golock trying to acquire a lock and being able to say when the
	// current one is expiring.
	slog.Debug(fmt.Sprintf("Lock %s set minimum grace period to: %d", redisKey, expireAtMin))
	_, _ = rdb.GetSet(ctx, redisKey, expireAtMin).Result()

	// Set the key to expire after the minimum grace period has passed.
	slog.Debug(fmt.Sprintf("Lock %s set to expire at: %d", redisKey, expireAtMin))
	_ = rdb.ExpireAt(ctx, redisKey, time.Unix(expireAtMin, 0))

	return exitCode
}

func main() {
	if os.Getenv("CRONLOCK_VERBOSE") == "yes" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	exitCode := run()
	os.Exit(exitCode)
}
