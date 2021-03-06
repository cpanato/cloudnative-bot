# CloudNative Twitter Bot

This is the [@CloudNativeBot](https://twitter.com/CloudNativeBot) Twitter bot code :-)

It uses [OpenFaas](https://github.com/openfaas/faas) to run the several slash commands and also uses the [OpenFaas Connector-SDK](https://github.com/openfaas/connector-sdk) and [Cron Connector](https://github.com/openfaas/cron-connector) to call the functions
based on the trigger received from the Twitter stream.

## Available commands at Twitter

Currently we have three commands available for any twitter user:

- `/cloudnativetv` - returns a random logo for the CNCF CloudNative TV shows
- `/k8s-patch-schedule` - returns the patch schedule for the active release branches fro Kubernetes
- `/kubecon-random-video`- returns a random KubeCon video

## Cron Twitter Jobs

- Next Kubecon - tweet how many days you need to wait for the next KubeCon
- CFP closes - tweet how long the CFP will be open for Proposal submission (TBD)

## Twitter Stream (OpenFaas Connector-SDK)

You can deploy the Twitter stream using [ko](https://github.com/google/ko) and then deploy it in your K8s cluster

To build/push and have the manifests ready to deploy you can run

```
$ export KO_DOCKER_REPO=ctadeu

$ ko resolve -f  kubernetes/stream/ > release.yaml

$ kubectl apply -f release.yaml

```

## OpenFaas CloudNative Bot Functions

To deploy each function you can use the `faas-cli` command

For example to deploy the `kubecon-randon-video`

```
$ cd functions/

$ faas-cli up -f functions-stack.yml --gateway https://YOUR-OPENFAAS

```


