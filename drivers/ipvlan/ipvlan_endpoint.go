package ipvlan

import (
	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/netutils"
	"github.com/vishvananda/netlink"
)

// CreateEndpoint assigns the mac, ip and endpoint id for the new container
func (d *driver) CreateEndpoint(nid, eid string, ifInfo driverapi.InterfaceInfo,
	epOptions map[string]interface{}) error {

	if err := validateID(nid, eid); err != nil {
		return err
	}
	n := d.network(nid)
	if n == nil {
		return fmt.Errorf("network id %q not found", nid)
	}
	ep := &endpoint{
		id:   eid,
		addr: ifInfo.Address(),
		mac:  ifInfo.MacAddress(),
	}

	if ep.addr == nil {
		return fmt.Errorf("create endpoint was not passed interface IP address")
	}

	if s := n.getSubnetforIP(ep.addr); s == nil {
		return fmt.Errorf("no matching subnet for IP %q in network %q\n", ep.addr, nid)
	}

	if ep.mac == nil {
		ep.mac = netutils.GenerateMACFromIP(ep.addr.IP)
		if err := ifInfo.SetMacAddress(ep.mac); err != nil {
			return err
		}
	}

	if ep.addr == nil {
		return fmt.Errorf("create endpoint was not passed interface IP address")
	}

	if s := n.getSubnetforIP(ep.addr); s == nil {
		return fmt.Errorf("no matching subnet for IP %q in network %q\n", ep.addr, nid)
	}

	if ep.mac == nil {
		ep.mac = netutils.GenerateMACFromIP(ep.addr.IP)
		if err := ifInfo.SetMacAddress(ep.mac); err != nil {
			return err
		}
	}
	n.addEndpoint(ep)
	logrus.Debugf("Allocated Endpoint [IP_Addr: %s MAC_Addr: %s Endpoint_ID: %s Network_ID: %s]",
		ep.addr.String(), ep.mac.String(), ep.id, nid)
	return nil
}

// DeleteEndpoint
func (d *driver) DeleteEndpoint(nid, eid string) error {
	if err := validateID(nid, eid); err != nil {
		return err
	}
	n := d.network(nid)
	if n == nil {
		return fmt.Errorf("network id %q not found", nid)
	}
	ep := n.endpoint(eid)
	if ep == nil {
		return fmt.Errorf("endpoint id %q not found", eid)
	}
	if link, err := netlink.LinkByName(ep.srcName); err == nil {
		netlink.LinkDel(link)
	}
	return nil
}

// getSubnetforIP returns the subnet to which the given IP belongs
func (n *network) getSubnetforIP(ip *net.IPNet) *subnet {
	for _, s := range n.subnets {
		// first check if the mask lengths are the same
		i, _ := s.subnetIP.Mask.Size()
		j, _ := ip.Mask.Size()
		if i != j {
			continue
		}
		if s.subnetIP.Contains(ip.IP) {
			return s
		}
	}
	return nil
}
