package cloudrun

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dazwilkin/consul-sd-cloudrun/generic"

	"github.com/go-logr/logr"

	"google.golang.org/api/googleapi"

	"google.golang.org/api/run/v1"
)

// Client is a type that represents a Google Cloud Run API service
type Client struct {
	log    logr.Logger
	client *run.APIService
}

// NewClient is a function that returns a new Client
func NewClient(log logr.Logger) (*Client, error) {
	ctx := context.Background()
	cloudrunService, err := run.NewService(ctx)
	if err != nil {
		log.Error(err, "unable to create NewService")
		return &Client{}, err
	}
	return &Client{
		log:    log,
		client: cloudrunService,
	}, nil
}

// List is a method that queries a list of Google Cloud Platform projects for Cloud Run services
// It returns a map comprising service ID and generic.Service
// The ID is a hash of the service's URL|Endpoint
func (c *Client) List(projectIDs []string) (map[string]*generic.Service, error) {
	log := c.log.WithName("List")

	var services = make(map[string]*generic.Service)

	for _, projectID := range projectIDs {
		log := log.WithValues(
			"projectID", projectID,
		)

		rqst := c.client.Namespaces.Services.List(Parent(projectID))

		cont := ""
		for {
			rqst.Continue(cont)
			resp, err := rqst.Do()
			if err != nil {
				if e, ok := err.(*googleapi.Error); ok {
					if e.Code == http.StatusForbidden {
						// Probably (!) Cloud Run Admin API has not been used in this project
						log.Error(err, "Forbidden. Have you enabled Cloud Run Admin API for this project?")
						return nil, err
					}
				}

				log.Error(err, "unable to execute request")
				return nil, err
			}

			// Append Cloud Run services to the results
			for _, item := range resp.Items {
				service := &generic.Service{}
				if err := service.FromRun(item); err != nil {
					log.Error(err, "unable to convert Cloud Run service")
					return nil, err
				}

				// To avoid losing it, include Project ID in service metadata
				// The Cloud Run service retains the Project number as part of the Namespace name
				service.Meta["project_id"] = projectID

				ID := service.ID()
				services[ID] = service
			}

			if resp.Metadata != nil {
				// If there's Metadata, update continue_
				log.Info("Another page")
				cont = resp.Metadata.Continue
			} else {
				// Otherwise, we're done
				log.Info("Done")
				break
			}
		}
	}

	return services, nil
}

// Parent is a function that returns the parent value for the Cloud Run services list method
// The parent value is a combination of `namespaces/` and the Google Cloud Platform Project ID
func Parent(project string) string {
	return fmt.Sprintf("namespaces/%s", project)
}
