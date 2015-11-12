package ipvlan

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libnetwork/osl"
	"github.com/vishvananda/netlink"
)

const (
	ipv4ForwardConf     = "/proc/sys/net/ipv4/ip_forward"
	ipv4ForwardConfPerm = 0644
)

// Create the netlink interface specifying the source name
func createIpVlan(containerIfName, hostIface, ipvlanMode string) (string, error) {
	defer osl.InitOSContext()()

	// Set the ipvlan mode. Default is bridge
	mode, err := setIPVlanMode(ipvlanMode)
	if err != nil {
		logrus.Errorf("error parsing vlan mode [ %v ]: %s", mode, err)
		return "", fmt.Errorf("unsupported ipvlan mode: %s", err)
	}
	if ok := validateHostIface(hostIface); !ok {
		return "", fmt.Errorf("The requested interface [ %s ] was not found on the host.", hostIface)
	}
	// verify the Docker host interface acting as the ipvlan parent iface exists
	if ok := validateHostIface(hostIface); !ok {
		return "", fmt.Errorf("The requested interface [ %s ] was not found on the host.", hostIface)
	}
	// Get the link for the master index (Example: the docker host eth iface)
	hostEth, err := netlink.LinkByName(hostIface)
	if err != nil {
		logrus.Errorf("Error looking up the parent iface [ %s ] mode: [ %s ] error: [ %s ]", hostIface, mode, err)
	}
	// Create a ipvlan link
	ipvlan := &netlink.IPVlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        containerIfName,
			ParentIndex: hostEth.Attrs().Index,
		},
		Mode: mode,
	}
	if err := netlink.LinkAdd(ipvlan); err != nil {
		// verbose but will be an issue if you create a ipvlan and ipvlan using the same parent iface
		// Leaving temporarily for dev testing
		logrus.Errorf("failed to create Ipvlan: [ %v ] with the error: %s", ipvlan, err)
		logrus.Error("Ensure there are no existing [ ipvlan ] type links and remove with 'ip link del <link_name>'," +
			" also check `/var/run/docker/netns/` for orphaned links to unmount and delete, then restart the plugin")

		return "", fmt.Errorf("error creating veth pair: %v", err)
	}
	return ipvlan.Attrs().Name, nil
}

// Set one of the two ipvlan port type
func setIPVlanMode(mode string) (netlink.IPVlanMode, error) {
	switch mode {
	case "l2":
		return netlink.IPVLAN_MODE_L2, nil
	case "l3":
		return netlink.IPVLAN_MODE_L3, nil
	default:
		return 0, fmt.Errorf("unknown ipvlan mode: %s", mode)
	}
}

// enable linux ip forwarding
func setupIPForwarding() error {
	// Get the current IPv4 forwarding setup
	ipv4ForwardData, err := ioutil.ReadFile(ipv4ForwardConf)
	if err != nil {
		return fmt.Errorf("Cannot read IP forwarding setup: %v", err)
	}

	// Enable IPv4 forwarding only if it is not already enabled
	if ipv4ForwardData[0] != '1' {
		// Enable IPv4 forwarding
		if err := ioutil.WriteFile(ipv4ForwardConf, []byte{'1', '\n'}, ipv4ForwardConfPerm); err != nil {
			return fmt.Errorf("Setup IP forwarding failed: %v", err)
		}
	}
	return nil
}

// Check if a netlink interface exists in the default namespace
func validateHostIface(ifaceStr string) bool {
	_, err := net.InterfaceByName(ifaceStr)
	if err != nil {
		return false
	}
	return true
}

// TODO add to onceInit()
// modprobe for the nessecary ko mod for the driver type
func kernelSupport(networkTpe string) error {
	// attempt to load the module,silent if successful or already loaded
	exec.Command("modprobe", networkType).Run()
	f, err := os.Open("/proc/modules")
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if strings.Contains(s.Text(), networkType) {
			return nil
		}
	}
	return fmt.Errorf("%s was not found in /proc/modules", networkType)
}
