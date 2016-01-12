package routingmanager

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/drivers/ipvlan/routing/gobgp"
	"net"
)

var routemanager RoutingInterface

type RoutingInterface interface {
	StartMonitoring() error
	AdvertizeNewRoute(localPrefix *net.IPNet) error
	WithdrawRoute(localPrefix *net.IPNet) error
}

func InitRoutingManager(masterIface string, managermode string, serveraddr string) {
	switch managermode {
	case "gobgp":
		log.Infof("Routing manager is %s", managermode)
		routemanager = gobgp.NewBgpRouteManager(masterIface, net.ParseIP(serveraddr))
	default:
		log.Infof("Default Routing manager: Gobgp")
		routemanager = gobgp.NewBgpRouteManager(masterIface, net.ParseIP(serveraddr))
	}
}

func StartMonitoring() error {
	error := routemanager.StartMonitoring()
	if error != nil {
		return error
	}
	return nil
}
func WithdrawRoute(localPrefix *net.IPNet) error {
	error := routemanager.WithdrawRoute(localPrefix)
	if error != nil {
		return error
	}
	return nil
}
func AdvertizeNewRoute(localPrefix *net.IPNet) error {
	error := routemanager.AdvertizeNewRoute(localPrefix)
	if error != nil {
		return error
	}
	return nil
}
