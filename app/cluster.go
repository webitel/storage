package app

import (
	"fmt"

	"github.com/webitel/engine/pkg/discovery"
	"github.com/webitel/storage/model"
)

type cluster struct {
	app       *App
	discovery discovery.ServiceDiscovery
}

func NewCluster(app *App) *cluster {
	return &cluster{
		app: app,
	}
}

func (c *cluster) Start() error {
	sd, err := discovery.NewServiceDiscovery(*c.app.id, c.app.Config().DiscoverySettings.Url, c.app.health.ReadyFunc())
	if err != nil {
		return err
	}
	c.discovery = sd

	addr, err := c.app.GrpcServer.AdvertiseAddr()
	if err != nil {
		return fmt.Errorf("resolve grpc advertise address: %w; set GRPC_ADDR to a specific IP", err)
	}
	err = sd.RegisterService(model.APP_SERVICE_NAME, addr.Addr().String(), int(addr.Port()), model.APP_SERVICE_TTL, model.APP_DEREGISTER_CRITICAL_TTL)
	if err != nil {
		return err
	}

	return nil
}

func (c *cluster) Stop() {
	c.discovery.Shutdown()
}
