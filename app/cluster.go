package app

import (
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
	sd, err := discovery.NewServiceDiscovery(*c.app.id, c.app.Config().DiscoverySettings.Url, func() (b bool, appError error) {
		return true, nil
	})
	if err != nil {
		return err
	}
	c.discovery = sd

	host, port := c.app.GrpcServer.GetPublicInterface()
	err = sd.RegisterService(model.APP_SERVICE_NAME, host, port, model.APP_SERVICE_TTL, model.APP_DEREGESTER_CRITICAL_TTL)
	if err != nil {
		return err
	}

	return nil
}

func (c *cluster) Stop() {
	c.discovery.Shutdown()
}
