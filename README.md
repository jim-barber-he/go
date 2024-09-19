
# My Go repository

## Introduction

This repository was created for me to use as I'm learning to write Go.  
I decided that the best way for me to learn it was to write some utilities that will be useful for me in my line of work.

## Utilities

The utilities in this repository are:

- [golock](golock/) Creates locks in a [Redis](https://redis.io/) server to coordinate running a command on distributed servers.
- [kubectl-n](kubectl-plugins/kubectl-n) A [kubectl](https://kubernetes.io/docs/reference/kubectl/) plugin that is an alternate
  version of `kubectl get nodes`.
- [kubectl-p](kubectl-plugins/kubectl-p) A [kubectl](https://kubernetes.io/docs/reference/kubectl/) plugin that is an alternate
  version of `kubectl get pods`.
- [ssm](ssm/) A tool for managing AWS SSM parameters.

## Go Libraries

The following libraries in this repository were written to implement the utilities above.

- [aws](aws/) Implements functions to interact with Amazon Web Services.
- [k8s](k8s/) Implements functions to interact with Kubernetes clusters.
- [texttable](texttable/) Implements functions for handling outputting a text based table.
- [util](util/) Implements various utility functions.
