package main

import (
	"flag"
	stdlog "log"
	"os"
	"runtime"
	"strings"
	"time"

	cloudrun "github.com/dazwilkin/consul-sd-cloudrun/cloudrun"
	consul "github.com/dazwilkin/consul-sd-cloudrun/consul"

	"github.com/go-logr/stdr"
)

var (
	// BuildTime is the time that this binary was built represented as a UNIX epoch
	BuildTime string
	// GitCommit is the git commit value and is expected to be set during build
	GitCommit string
	// GoVersion is the Golang runtime version
	GoVersion = runtime.Version()
	// OSVersion is the OS version (uname --kernel-release) and is expected to be set during build
	OSVersion string
	// StartTime is the start time of the exporter represented as a UNIX epoch
	StartTime = time.Now().Unix()
)

var (
	consulEndpoint = flag.String("consul", "", "Endpoint of Consul service")
	consulFilter   = flag.String("filter", "", "Consul Filter expression applied to refine Consul service query")
	projectIDs     = flag.String("project_ids", "", "Comma-separated list of GCP Project IDs")
	frequency      = flag.Duration("frequency", 15*time.Second, "Frequency of polling Projects for Cloud Run services")
)

func main() {
	log := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})

	flag.Parse()

	if *consulEndpoint == "" {
		log.Info("`--consul` is a required flag.")
		return
	}
	if *projectIDs == "" {
		// Nothing to do
		log.Info("No `--project_ids` provided. Nothing to do!")
		return
	}

	log.Info("Flags",
		"consulEndpoint", *consulEndpoint,
		"consulFilter", *consulFilter,
		"projectIDs", *projectIDs,
		"frequency", *frequency,
	)

	// Convert comma-separated list of Project IDs into a slice of Project IDs
	projectIDs := strings.Split(*projectIDs, ",")

	// Create Cloud Run client
	cloudrunClient, err := cloudrun.NewClient(log)
	if err != nil {
		log.Error(err, "unable to create Cloud Run client")
		return
	}

	// Create Consul client
	consulClient, err := consul.NewClient(*consulEndpoint, log)
	if err != nil {
		log.Error(err, "unable to create Consul client")
		return
	}

	// Create ticker
	ticker := time.NewTicker(*frequency)

	// Repeat forever
	for ; true; <-ticker.C {
		// Reset log
		log := log

		// Start
		start := time.Now()
		log.Info("Enumerating Projects",
			"start", start,
		)

		// List Cloud Run services across Project IDs
		wants, err := cloudrunClient.List(projectIDs)
		if err != nil {
			log.Error(err, "unable to list Cloud Run services for project")
		}

		// Determine filtered list of Consul services
		haves, err := consulClient.List(*consulFilter)
		if err != nil {
			log.Error(err, "unable to retrieve list of Consul services")
			return
		}

		// Compare the two

		// if we have the Cloud Run service but don't have it in Consul, add it to Consul
		for key, want := range wants {
			log.Info("Wants",
				"key", key,
			)

			// If the want isn't in haves, add it
			if _, ok := haves[key]; !ok {
				// Add the want
				ID, err := consulClient.Create(want)
				log = log.WithValues(
					"ID", ID,
				)
				if err != nil {
					log.Error(err, "unable to create Consul service")
				}
			}
		}

		// if we have it in Consul but don't have the Cloud Run service, delete it from Consul
		for key := range haves {
			log.Info("Haves",
				"key", key,
			)

			// if the have isn't in wants, delete it
			if _, ok := wants[key]; !ok {
				if err := consulClient.Delete(key); err != nil {
					log.Error(err, "unable to delete Consul service")
				}
			}
		}

		// Done
		end := time.Now()
		log.Info("Done Enumerating Projects",
			"start", start,
			"end", end,
			"duration", end.Sub(start),
		)
	}
}
