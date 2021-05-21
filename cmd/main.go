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

	if *projectIDs == "" {
		// Nothing to do
		log.Info("No `--project_ids` provided. Nothing to do!")
		return
	}
	log.Info("Projects", "projectIDs", *projectIDs)
	log.Info("Frequency", "frequency", *frequency)

	// Convert comma-separated list into slice
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
	}

	// Repeat forever
	for {
		// Reset log
		log := log

		// Determine start time in order to calculate remaining time at end of loop
		start := time.Now()
		log.Info("Enumerating Projects",
			"start", start,
		)

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
		// if we want it but don't have it, add it
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

		// if we have it but don't want it, delete it
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

		// Sleep remainder of time until next frequency
		remaining := *frequency - time.Since(start)
		if remaining < 0 {
			remaining = 0
		}
		log.Info("Sleeping",
			"remaining", remaining,
		)
		time.Sleep(remaining)
	}
}
