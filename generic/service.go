package generic

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/hashicorp/consul/api"

	"google.golang.org/api/run/v1"
)

const (
	labelLocation = "cloud.googleapis.com/location"
)

type Service struct {
	Name    string
	Meta    map[string]string
	Tags    []string
	Address string
	Port    int
}

func (s *Service) ID() string {
	x := sha256.Sum224([]byte(s.Address))
	return fmt.Sprintf("%x", x[:])
}
func (s *Service) FromConsul(consulService *api.AgentService) error {
	s.Name = consulService.ID
	s.Meta = consulService.Meta
	s.Tags = consulService.Tags
	s.Address = consulService.Address
	s.Port = consulService.Port

	return nil
}
func (s *Service) FromRun(runService *run.Service) error {
	region, err := getRegion(runService.Metadata.Labels)
	if err != nil {
		return err
	}

	var meta = make(map[string]string)
	// Represents the GCP Project number (not ID)
	meta["project_number"] = runService.Metadata.Namespace
	meta["region"] = region

	// Populate Service fields
	s.Name = runService.Metadata.Name
	s.Meta = meta
	s.Tags = []string{}
	s.Address = strings.TrimPrefix(runService.Status.Url, "https://")
	s.Port = 443

	return nil
}
func (s *Service) String() string {
	return fmt.Sprintf("Name: %s, Meta: %v, Tags: %v, Address: %s, Port: %d", s.Name, s.Meta, s.Tags, s.Address, s.Port)
}
func getRegion(m map[string]string) (string, error) {
	// labels:
	//   cloud.googleapis.com/location: us-west1
	region := m[labelLocation]
	if region == "" {
		return "", fmt.Errorf("unable to extract GCP region for service labels")
	}
	return region, nil
}
