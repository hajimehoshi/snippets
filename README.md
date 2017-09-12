# How to test this app on your local machine

## Install Cloud SDK

See https://cloud.google.com/appengine/docs/flexible/go/quickstart

## Prepare app.yaml

```yaml
runtime: go
env: flex

automatic_scaling:
  min_num_instances: 1

#[START env_variables]
env_variables:
  GCLOUD_DATASET_ID: your-project-name
#[END env_variables]
```

## Run Datastore Emulator

See https://cloud.google.com/datastore/docs/tools/datastore-emulator

```shell
gcloud beta emulators datastore start --host-port=:8000
```

## Run this app with `bo run`

```shell
DATASTORE_EMULATOR_HOST=localhost:8000 GCLOUD_DATASET_ID=natto-umeboshi-20170912 go run datastore.go
```
