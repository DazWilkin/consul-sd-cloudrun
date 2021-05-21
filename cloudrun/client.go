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

type Client struct {
	log    logr.Logger
	client *run.APIService
}

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

// Parent is a function that returns the correct parent value for the Cloud Run services list method
func Parent(project string) string {
	return fmt.Sprintf("namespaces/%s", project)
}
