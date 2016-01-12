package ipvlan

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/drivers/ipvlan/routing"
	"github.com/docker/libnetwork/netlabel"
	"github.com/docker/libnetwork/options"
	"github.com/docker/libnetwork/types"
)

// CreateNetwork the network for the spcified driver type
func (d *driver) CreateNetwork(id string, option map[string]interface{}, ipV4Data, ipV6Data []driverapi.IPAMData) error {

	if id == "" {
		return fmt.Errorf("invalid network id")
	}
	// Parse and validate the config. It should not conflict with existing networks' config
	config, err := parseNetworkOptions(id, option)
	if err != nil {
		return err
	}
	// UX must specify a parent interface on the host to attach the container vif to
	if config.HostIface == "" {
		return fmt.Errorf("%s requires an interface from the docker host to be specified (example: -o host_iface=eth0)", networkType)
	}
	// if the host iface doesnt exist the requested pool will be released
	if ok := validateHostIface(config.HostIface); !ok {
		return fmt.Errorf("The requested interface [ %s ] was not found on the host.", config.HostIface)
	}
	// Default to Bridge mode
	if config.IpvlanMode == "" {
		config.IpvlanMode = modeL2
	}
	n := &network{
		id:        id,
		driver:    d,
		endpoints: endpointTable{},
		subnets:   []*subnet{},
		netConfig: config,
	}

	if config.IpvlanMode == modeL3 && config.RoutingManager != "" {
		routingmanager.InitRoutingManager(config.HostIface, config.RoutingManager, "127.0.0.1")
		go routingmanager.StartMonitoring()
	}
	for _, ipd := range ipV4Data {
		s := &subnet{
			subnetIP: ipd.Pool,
			gwIP:     ipd.Gateway,
		}
		n.subnets = append(n.subnets, s)
		logrus.Debugf("Network added [Network_Type: %s, Mode: %s, Subnet: %s, Gateway %s, Host_Interface: %s] ",
			networkType, n.netConfig.IpvlanMode, s.subnetIP.String(), s.gwIP.String(), n.netConfig.HostIface)
		if config.IpvlanMode == modeL3 && config.RoutingManager != "" {
			err := routingmanager.AdvertizeNewRoute(s.subnetIP)
			if err != nil {
				fmt.Errorf("Error installing container route : %s", err)
			}
		}

	}
	d.addNetwork(n)
	return nil
}

// DeleteNetwork the network for the spcified driver type
func (d *driver) DeleteNetwork(nid string) error {
	if nid == "" {
		return fmt.Errorf("invalid network id")
	}
	n := d.network(nid)
	if n == nil {
		return fmt.Errorf("could not find network with id %s", nid)
	}
	if n.netConfig.IpvlanMode == modeL3 && n.netConfig.RoutingManager != "" {
		for _, subnet := range n.subnets {
			err := routingmanager.WithdrawRoute(subnet.subnetIP)
			if err != nil {
				fmt.Errorf("Error withdrawing container route : %s", err)
			}
		}

	}
	d.deleteNetwork(nid)
	return nil
}

// Validate performs a static validation on the network configuration parameters.
func (c *networkConfiguration) Validate() error {

	if c.Mtu < 0 {
		return ErrInvalidMtu(c.Mtu)
	}
	// If bridge v4 subnet is specified
	if c.AddressIPv4 != nil {
		// If Container restricted subnet is specified, it must be a subset of bridge subnet
		if c.FixedCIDR != nil {
			// Check Network address
			if !c.AddressIPv4.Contains(c.FixedCIDR.IP) {
				return &ErrInvalidContainerSubnet{}
			}
			// Check it is effectively a subset
			brNetLen, _ := c.AddressIPv4.Mask.Size()
			cnNetLen, _ := c.FixedCIDR.Mask.Size()
			if brNetLen > cnNetLen {
				return &ErrInvalidContainerSubnet{}
			}
		}
		// If default gw is specified, it must be part of bridge subnet
		if c.DefaultGatewayIPv4 != nil {
			if !c.AddressIPv4.Contains(c.DefaultGatewayIPv4) {
				return &ErrInvalidGateway{}
			}
		}
	}

	// If default v6 gw is specified, FixedCIDRv6 must be specified and gw must belong to FixedCIDRv6 subnet
	if c.EnableIPv6 && c.DefaultGatewayIPv6 != nil {
		if c.FixedCIDRv6 == nil || !c.FixedCIDRv6.Contains(c.DefaultGatewayIPv6) {
			return &ErrInvalidGateway{}
		}
	}
	return nil
}

// Conflicts check if two NetworkConfiguration objects overlap
func (c *networkConfiguration) Conflicts(o *networkConfiguration) bool {
	if o == nil {
		return false
	}

	// They must be in different subnets
	if (c.AddressIPv4 != nil && o.AddressIPv4 != nil) &&
		(c.AddressIPv4.Contains(o.AddressIPv4.IP) || o.AddressIPv4.Contains(c.AddressIPv4.IP)) {
		return true
	}

	return false
}

// Parse docker network options
func parseNetworkOptions(id string, option options.Generic) (*networkConfiguration, error) {
	var (
		err    error
		config = &networkConfiguration{}
	)
	// Parse generic label first, config will be re-assigned
	if genData, ok := option[netlabel.GenericData]; ok && genData != nil {
		if config, err = parseNetworkGenericOptions(genData); err != nil {
			return nil, err
		}
	}
	// Process well-known labels next
	if val, ok := option[netlabel.EnableIPv6]; ok {
		config.EnableIPv6 = val.(bool)
	}
	return config, nil
}

// Parse generic driver docker network options
func parseNetworkGenericOptions(data interface{}) (*networkConfiguration, error) {
	var (
		err    error
		config *networkConfiguration
	)
	switch opt := data.(type) {
	case *networkConfiguration:
		config = opt
	case map[string]string:
		config = &networkConfiguration{
			EnableICC:          true,
			EnableIPMasquerade: true,
		}
		err = config.fromLabels(opt)
	case options.Generic:
		var opaqueConfig interface{}
		if opaqueConfig, err = options.GenerateFromModel(opt, config); err == nil {
			config = opaqueConfig.(*networkConfiguration)
		}
	default:
		err = types.BadRequestErrorf("do not recognize network configuration format: %T", opt)
	}

	return config, err
}

// Bind the generic options
func (c *networkConfiguration) fromLabels(labels map[string]string) error {
	for label, value := range labels {
		switch label {
		case hostIfaceOpt:
			c.HostIface = value

		case driverModeOpt:
			c.IpvlanMode = value
			logrus.Debugf("Driver %s mode type is: %s", networkType, value)

		case routingManagerOpt:
			c.RoutingManager = value

		}
	}

	return nil
}

func parseErr(label, value, errString string) error {
	return types.BadRequestErrorf("failed to parse %s value: %v (%s)", label, value, errString)
}
