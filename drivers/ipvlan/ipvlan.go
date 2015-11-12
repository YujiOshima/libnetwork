package ipvlan

import (
	"net"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/datastore"
	"github.com/docker/libnetwork/driverapi"
	"github.com/docker/libnetwork/osl"
)

const (
	containerVethPrefix     = "eth"
	maxAllocatePortAttempts = 10
	networkType             = "ipvlan"
	vethPrefix              = "veth"
	vethLen                 = 7
)

type endpointTable map[string]*endpoint

type networkTable map[string]*network

type driver struct {
	ifaceName string
	networks  networkTable
	sync.Once
	sync.Mutex
}

type endpoint struct {
	id      string
	mac     net.HardwareAddr
	addr    *net.IPNet
	srcName string
}

type network struct {
	id        string
	sbox      osl.Sandbox
	endpoints endpointTable
	subnets   []*subnet
	driver    *driver
	netConfig *networkConfiguration
	sync.Mutex
}

type networkConfiguration struct {
	AddressIPv4        *net.IPNet
	AddressIPv6        *net.IPNet
	FixedCIDR          *net.IPNet
	FixedCIDRv6        *net.IPNet
	EnableIPv6         bool
	EnableIPMasquerade bool
	EnableICC          bool
	Mtu                int
	DefaultGatewayIPv4 net.IP
	DefaultGatewayIPv6 net.IP
	HostIface          string
	IpvlanMode         string
}

type subnet struct {
	initErr  error
	subnetIP *net.IPNet
	gwIP     *net.IPNet
}

func onceInit() {
	// TODO add some one time default inits
}

// Init is called from libnetwork to initialize and register the driver
func Init(dc driverapi.DriverCallback, config map[string]interface{}) error {
	if err := kernelSupport(networkType); err != nil {
		logrus.Warnf("Failed to initialize the required %s kernel module: %v", networkType, err)
		return err
	}
	c := driverapi.Capability{
		DataScope: datastore.LocalScope,
	}
	d := &driver{
		networks: networkTable{},
	}

	return dc.RegisterDriver(networkType, d, c)
}

// EndpointOperInfo
func (d *driver) EndpointOperInfo(nid, eid string) (map[string]interface{}, error) {
	return make(map[string]interface{}, 0), nil
}

// Type
func (d *driver) Type() string {
	return networkType
}

// DiscoverNew is a notification for a new discovery event, such as a new node joining a cluster
func (d *driver) DiscoverNew(dType driverapi.DiscoveryType, data interface{}) error {
	return nil
}

// DiscoverDelete is a notification for a discovery delete event, such as a node leaving a cluster
func (d *driver) DiscoverDelete(dType driverapi.DiscoveryType, data interface{}) error {
	return nil
}
