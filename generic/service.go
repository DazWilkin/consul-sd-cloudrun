package generic

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"

	"google.golang.org/api/run/v1"
)

// Service is a type that represents a generic service
// It is the canonical form for Cloud Run and Consul services
type Service struct {
	Name    string
	Meta    map[string]string
	Tags    []string
	Address string
	Port    int
}

// ID is a method that calculates a unique ID for a service
// It uses the service's Address:Port
// The service's Address:Port is assumed to be unique across all services
func (s *Service) ID() string {
	endpoint := fmt.Sprintf("%s:%d", s.Address, s.Port)
	x := sha256.Sum224([]byte(endpoint))
	return fmt.Sprintf("%x", x[:])
}

// FromConsul is a method that converts a Consul service into a Service
func (s *Service) FromConsul(consulService *api.AgentService) error {
	s.Name = consulService.ID
	s.Meta = consulService.Meta
	s.Tags = consulService.Tags
	s.Address = consulService.Address
	s.Port = consulService.Port

	return nil
}

// FromRun is a method that converts a Cloud Run service into a Service
func (s *Service) FromRun(runService *run.Service) error {
	var meta = make(map[string]string)
	// Represents the GCP Project number (not ID)
	meta["project_number"] = runService.Metadata.Namespace

	// Extract Cloud Run labels
	// Replacing the system-defined label cloud.googleapis.com/location as region
	// Ignoring any other DNS-like labels
	for key, val := range runService.Metadata.Labels {
		if key == "cloud.googleapis.com/location" {
			meta["region"] = val
		}
		// Avoid `.` and `/` as these are permitted by Cloud Run but aren't permitted by Consul as Tag keys
		if !strings.ContainsAny(key, "./") {
			meta[key] = val
		}
	}

	// Remove the prefixing `https://` and the suffixing `/` from the Cloud Run address
	// The protocol is for convenience when setting this as a target in Prometheus
	// The trailing `/` is to permit a valid endpoint to be created from Address:Port
	address := strings.TrimPrefix(runService.Status.Url, "https://")
	address = strings.TrimSuffix(address, "/")

	// Populate Service fields
	s.Name = runService.Metadata.Name
	s.Meta = meta
	s.Tags = []string{}
	s.Address = address
	s.Port = 443

	return nil
}

// String is a method that converts a Service into a string
func (s *Service) String() string {
	return fmt.Sprintf("Name: %s, Meta: %v, Tags: %v, Address: %s, Port: %d", s.Name, s.Meta, s.Tags, s.Address, s.Port)
}
