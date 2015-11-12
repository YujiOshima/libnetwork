package ipvlan

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/netutils"
	"github.com/docker/libnetwork/osl"
)

// Join method is invoked when a Sandbox is attached to an endpoint.
func (d *driver) Join(nid, eid string, sboxKey string, jinfo driverapi.JoinInfo, options map[string]interface{}) error {
	defer osl.InitOSContext()()
	// Get the existing network from the network ID
	n := d.network(nid)
	if n == nil {
		return fmt.Errorf("could not find network with id %s", nid)
	}
	endpoint := n.endpoint(eid)
	if endpoint == nil {
		return fmt.Errorf("could not find endpoint with id %s", eid)
	}
	if endpoint == nil {
		return EndpointNotFoundError(eid)
	}
	// Generate a name for the iface that will be renamed to eth0 in the sbox
	containerIfName, err := netutils.GenerateIfaceName(vethPrefix, vethLen)
	if err != nil {
		return fmt.Errorf("error generating an interface name: %s", err)
	}

	// Create the netlink ipvlan interface
	vethName, err := createIpVlan(containerIfName, n.netConfig.HostIface, n.netConfig.IpvlanMode)
	if err != nil {
		logrus.Errorf("Error creating ipvlan link: %s", err)
		return err
	}
	// bind the generated iface name to the endpoint
	endpoint.srcName = vethName

	if len(n.subnets) > 0 {
		if n.subnets[0].gwIP != nil {
			err = jinfo.SetGateway(n.subnets[0].gwIP.IP)
			if err != nil {
				logrus.Infof("Error setting the default gateway %v error: %s", n.subnets[0].gwIP.IP, err)
				return err
			}
			logrus.Debugf("Endpoint joined Subnet: %s using the Gateway: %s",
				n.subnets[0].subnetIP.String(), n.subnets[0].gwIP.String())
		}
	}
	iNames := jinfo.InterfaceName()
	err = iNames.SetNames(vethName, containerVethPrefix)
	if err != nil {
		return err
	}
	return nil
}

// Leave method is invoked when a Sandbox detaches from an endpoint.
func (d *driver) Leave(nid, eid string) error {
	network, err := d.getNetwork(nid)
	if err != nil {
		return err
	}
	endpoint, err := network.getEndpoint(eid)
	if err != nil {
		return err
	}
	if endpoint == nil {
		return EndpointNotFoundError(eid)
	}
	return nil
}
