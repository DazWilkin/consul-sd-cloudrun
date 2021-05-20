package consul

import (
	"github.com/dazwilkin/consul-sd-cloudrun/generic"

	"github.com/hashicorp/consul/api"

	"github.com/go-logr/logr"
)

type Client struct {
	log    logr.Logger
	client *api.Client
}

func NewClient(address string, log logr.Logger) (*Client, error) {
	client, err := api.NewClient(&api.Config{
		Address: address,
		Scheme:  "http",
	})
	if err != nil {
		return &Client{}, err
	}

	return &Client{
		log:    log,
		client: client,
	}, nil
}
func (c *Client) List() (map[string]*generic.Service, error) {
	log := c.log.WithName("List")

	items, err := c.client.Agent().Services()
	if err != nil {
		log.Error(err, "unable to retrieve list of Consul services")
		return nil, err
	}

	services := make(map[string]*generic.Service)
	for k, item := range items {
		log.Info("map",
			"key", k,
			"id", item.ID,
			"address", item.Address,
			"port", item.Port,
			"tags", item.Tags,
		)
		service := &generic.Service{}
		if err := service.FromConsul(item); err != nil {
			log.Error(err, "unable to convert Cloud Run service")
			return nil, err
		}
		services[k] = service
	}

	return services, nil
}
func (c *Client) Create(service *generic.Service) (string, error) {
	log := c.log.WithName("Create")
	log = log.WithValues(
		"serviceName", service.Name,
	)

	ID := service.ID()
	log = log.WithValues(
		"serviceID", ID,
	)

	log.Info("Service registered")
	return ID, c.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
		ID:      ID,
		Name:    service.Name,
		Meta:    service.Meta,
		Tags:    service.Tags,
		Address: service.Address,
		Port:    service.Port,
	})
}
func (c *Client) Get(ID string) (*generic.Service, error) {
	log := c.log.WithName("Get")
	log = log.WithValues(
		"serviceID", ID,
	)

	service, _, err := c.client.Agent().Service(ID, &api.QueryOptions{})
	if err != nil {
		log.Error(err, "unable to query service")
		return &generic.Service{}, err
	}

	return &generic.Service{
		Name:    service.ID,
		Meta:    service.Meta,
		Tags:    service.Tags,
		Address: service.Address,
		Port:    service.Port,
	}, nil
}
func (c *Client) Delete(ID string) error {
	log := c.log.WithName("Delete")
	log = log.WithValues(
		"serviceID", ID,
	)

	log.Info("Service deleted")
	return c.client.Agent().ServiceDeregister(ID)
}
