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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/jim-barber-he/go/util"
	redis "github.com/redis/go-redis/v9"
)

// Default Values.
const (
	defLockHost              string = "localhost"
	defLockPort              int    = 6379
	defLockDB                int    = 0
	defLockTLS               bool   = false
	defLockTLSSkipVerify     bool   = false
	defLockRedisTimeout      int    = 30
	defLockReconnectAttempts int    = 5
	defLockReconnectBackoff  int    = 5
	defLockGrace             int    = 40
	defLockRelease           int    = 86400
	defLockPrefix            string = "cronlock."
	defLockReset             string = "no"
	defLockTimeout           int    = 0
)

// Environment Variables.
const (
	envLockHost              = "CRONLOCK_HOST"
	envLockPort              = "CRONLOCK_PORT"
	envLockDB                = "CRONLOCK_DB"
	envLockTLS               = "CRONLOCK_TLS"
	envLockTLSSkipVerify     = "CRONLOCK_TLS_SKIP_VERIFY"
	envLockRedisTimeout      = "CRONLOCK_REDIS_TIMEOUT"
	envLockReconnectAttempts = "CRONLOCK_RECONNECT_ATTEMPTS"
	envLockReconnectBackoff  = "CRONLOCK_RECONNECT_BACKOFF"
	envLockGrace             = "CRONLOCK_GRACE"
	envLockRelease           = "CRONLOCK_RELEASE"
	envLockPrefix            = "CRONLOCK_PREFIX"
	envLockKey               = "CRONLOCK_KEY"
	envLockReset             = "CRONLOCK_RESET"
	envLockTimeout           = "CRONLOCK_TIMEOUT"
	envLockVerbose           = "CRONLOCK_VERBOSE"
)

// Exit codes.
// An exit code less than 200 means a lock was acquired and is the exit code of the command that was run.
const (
	exitSuccess int = 200 // Success. Delete succeeded OR lock not acquired, but normal execution.
	exitFailure int = 201 // Failure. Error encountered.
	exitTimeout int = 202 // Failure. Lock timed out.
)

func NewRedisPingError(response string) error {
	return &util.Error{
		Msg:   "could not ping Redis: ",
		Param: response,
	}
}

// getRedisKey returns the name of the Redis key to use for the lock.
// If not set via the environment, then one is calculated based on the MD5 hash of the command and its arguments.
func getRedisKey(lockPrefix, command string) string {
	redisKey := os.Getenv(envLockKey)
	if redisKey == "" {
		hash := md5.Sum([]byte(command))
		redisKey = hex.EncodeToString(hash[:])
	}

	return lockPrefix + redisKey
}

// getRedisOptions returns a redis.Options struct with the values set from the environment variables.
func getRedisOptions() *redis.Options {
	redisReconnectBackoff := time.Second * time.Duration(
		util.GetEnvInt(envLockReconnectBackoff, defLockReconnectBackoff),
	)
	opts := &redis.Options{
		Addr: fmt.Sprintf(
			"%s:%d", util.GetEnv(envLockHost, defLockHost), util.GetEnvInt(envLockPort, defLockPort),
		),
		DB:              util.GetEnvInt(envLockDB, defLockDB),
		DialTimeout:     time.Second * time.Duration(util.GetEnvInt(envLockRedisTimeout, defLockRedisTimeout)),
		MaxRetries:      util.GetEnvInt(envLockReconnectAttempts, defLockReconnectAttempts),
		MaxRetryBackoff: redisReconnectBackoff,
		MinRetryBackoff: redisReconnectBackoff,
	}
	if auth := os.Getenv("CRONLOCK_AUTH"); auth != "" {
		opts.Password = auth
	}
	if tlsEnabled := util.GetEnvBool(envLockTLS, defLockTLS); tlsEnabled {
		opts.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: util.GetEnvBool(envLockTLSSkipVerify, defLockTLSSkipVerify),
		}
	}

	return opts
}

// redisConnect connects to a Redis server with the supplied options and returns a client.
func redisConnect(ctx context.Context, connOpts *redis.Options) (*redis.Client, error) {
	slog.Debug("Connecting to redis at " + connOpts.Addr)

	rdb := redis.NewClient(connOpts)

	response, err := rdb.Ping(ctx).Result()
	switch {
	case err != nil:
		return nil, fmt.Errorf("could not connect to Redis: %w", err)
	case response != "PONG":
		return nil, NewRedisPingError(response)
	}

	return rdb, nil
}

// resetKey will remove the supplied key from Redis if envLockReset is set to "yes".
// Will return 0 if envLockReset is not "yes".
func resetKey(ctx context.Context, rdb *redis.Client, redisKey string) int {
	reset := util.GetEnv(envLockReset, defLockReset)
	if reset == "yes" {
		slog.Debug(fmt.Sprintf("Removing %s key", redisKey))
		removed, err := rdb.Del(ctx, redisKey).Result()
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to remove key %s: %v", redisKey, err))

			return exitFailure
		}
		if removed == 0 || removed == 1 {
			return exitSuccess
		}
		slog.Error("Unable to remove " + redisKey)

		return exitFailure
	}

	return 0
}

func run() int {
	ctx := context.Background()

	// Connect to Redis.
	rdb, err := redisConnect(ctx, getRedisOptions())
	if err != nil {
		slog.Error(err.Error())

		return exitFailure
	}
	defer rdb.Close()

	// Command to run and its arguments represented as a string.
	command := strings.Join(os.Args[1:], " ")

	// The key to use in Redis.
	redisKey := getRedisKey(util.GetEnv(envLockPrefix, defLockPrefix), command)

	// If envLockReset is true, this will remove redisKey from Redis and return a 2xx code.
	if ret := resetKey(ctx, rdb, redisKey); ret != 0 {
		return ret
	}

	// Control how long the lock is held for.
	lockGrace := util.GetEnvInt(envLockGrace, defLockGrace)
	lockRelease := util.GetEnvInt(envLockRelease, defLockRelease)

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

		return exitFailure
	}

	if acquired {
		slog.Debug(fmt.Sprintf("Lock %s acquired", redisKey))
	} else {
		// Handle edge cases.

		expiresAt, err := rdb.Get(ctx, redisKey).Result()
		if err != nil {
			slog.Error(fmt.Errorf("failed to get expiration time: %w", err).Error())

			return exitFailure
		}
		expiresIn, _ := strconv.Atoi(expiresAt)
		expiresIn -= int(time.Now().UTC().Unix())

		switch {
		case expiresIn > 0:
			slog.Debug(fmt.Sprintf(
				"Lock %s acquired by another process (expires in %ds)", redisKey, expiresIn,
			))

			return exitSuccess
		case expiresIn == 0:
			slog.Debug(fmt.Sprintf("Lock %s acquired by another process but expiring now", redisKey))

			return exitSuccess
		default:
			slog.Debug(fmt.Sprintf(
				"Lock %s acquired by another process but expired %ds ago", redisKey, -expiresIn,
			))
		}

		// Handle expired locks that were not cleaned up properly or not cleaned up yet because the golock that
		// requested it is still running.
		// Try to acquire a lock again, confirming that no other running golock beats us to it.
		reacquire, err := rdb.GetSet(ctx, redisKey, expireAtMax).Result()
		if err != nil {
			slog.Error(fmt.Errorf("failed to acquire lock: %w", err).Error())

			return exitFailure
		}
		expiresIn, _ = strconv.Atoi(reacquire)
		expiresIn -= int(time.Now().UTC().Unix())
		if expiresIn > 0 {
			slog.Debug(fmt.Sprintf(
				"Lock %s was just now acquired by a different process (expires in %ds)",
				redisKey,
				expiresIn,
			))

			return exitSuccess
		}
	}

	// Run command with an optional timeout.
	timeout := util.GetEnvInt(envLockTimeout, defLockTimeout)
	exitCode, err := util.RunWithTimeout(timeout, os.Args[1], os.Args[2:]...)
	if timeout > 0 && exitCode == util.ExitCodeProcessKilled {
		slog.Error(fmt.Sprintf("emergency: had to kill [%s] after %ds timeout", command, timeout))
		exitCode = exitTimeout
	}
	// Show any errors from trying to run the command that weren't from the command itself.
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		slog.Error(err.Error())
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
	if os.Getenv(envLockVerbose) == "yes" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	exitCode := run()
	os.Exit(exitCode)
}
