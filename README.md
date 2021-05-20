# Consul Service Discovery for Cloud Run

A Consul discovery agent that enumerates Cloud Run services and registers them with Consul.

## Run

Run Consul, Prometheus and cAdvisor:

```bash
docker-compose up
```

Then:

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