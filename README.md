# Consul Service Discovery for Cloud Run

[![build](https://github.com/DazWilkin/consul-sd-cloudrun/actions/workflows/build.yaml/badge.svg)](https://github.com/DazWilkin/consul-sd-cloudrun/actions/workflows/build.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/DazWilkin/consul-sd-cloudrun.svg)](https://pkg.go.dev/github.com/DazWilkin/consul-sd-cloudrun)
[![Go Report Card](https://goreportcard.com/badge/github.com/dazwilkin/consul-sd-cloudrun)](https://goreportcard.com/report/github.com/dazwilkin/consul-sd-cloudrun)

A Consul discovery agent that enumerates Cloud Run services and registers them with Consul.

## Image

+ ghcr.io/dazwilkin/consul-sd-cloudrun:281a1a2184871fe7b44f139ffbbd4b6c51219eba

## Run

### Docker (Compose)

Run Consul, Prometheus and cAdvisor:

```bash
docker-compose up
```

### Podman

```bash
POD="consul-pod"

podman pod create \
--name=${POD} \
--publish=8500:8500/tcp \
--publish=8600:8600/udp 

podman run \
--detach --rm --tty \
--pod=${POD} \
--name=consul \
docker.io/consul:1.11.0-beta

podman run \
--detach --rm --tty \
--pod=${POD} \
--name=discoverer \
--volume=${HOME}/.config/gcloud/application_default_credentials.json:/secrets/adc.json \
--env=GOOGLE_APPLICATION_CREDENTIALS=/secrets/adc.json \
ghcr.io/dazwilkin/consul-sd-cloudrun:281a1a2184871fe7b44f139ffbbd4b6c51219eba \
--consul=localhost:8500 \
--project_ids=${PROJECT}
```

### Discoverer only

```bash
# Use user's default account
export GOOGLE_APPLICATION_CREDENTIALS="${HOME}/.config/gcloud/application_default_credentials.json"

# Convert list of Projects into comma-separated list
# Includes trailing comma
PROJECTS=$(\
  gcloud projects list \
  --format='csv[terminator=","](projectId)') \
&& echo ${PROJECTS}

go run ./cmd \
--project_ids=${PROJECTS}
```

## Consul

```bash
http://localhost:8500/ui/dc1/services
```


## Debugging

Deregister services:

```bash
ID=...

curl \
--request PUT \
localhost:8500/v1/agent/service/deregister/${ID}
```

## Relabeling

![relabeling](/images/relabeling.png)

## Notes

The `consul` service that is registered (by default) with the Consul agent expects to be scraped as `/v1/agent/metrics` but this disagrees with the metrics endpoints of the Cloud Run services.

![`consul` service](/images/consulservice.png)

Want to exclude the `consul` service.

Using `relabel_config` to drop the `consul` service:

```YAML
relabel_configs:
- source_labels:
  - __meta_consul_service_port
  regex: "8300"
  action: drop
```

Here's the Prometheus configuration that scrapes the `consul` service if you decide you want it:

```YAML
scrape_configs:
  # Consul
  - job_name: consul
    metrics_path: "/v1/agent/metrics"
    params:
      format:
        - "prometheus"
    consul_sd_configs:
      - server: consul:8500
        datacenter: dc1
```