package gateway

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// ConfigureStaticIP adds a secondary IP to an interface,
// Example: staticVIP=192.168.56.199/24
// 		Get the route for this staticVIP
//		Find Interface with that route,
// 		Verify interface has IP in same network as staticVIP
// 		If found configure staticVIP on that interface.
func ConfigureStaticIP(staticVIP string) error {

	interfaceName, err := getInterfaceForNetwork(staticVIP)
	if err != nil {
		return err
	}
	return addSecondaryIPToInterface(staticVIP, interfaceName)
}

func RemoveStaticIP(staticVIP string) error {

	interfaceName, err := getInterfaceForNetwork(staticVIP)
	if err != nil {
		return err
	}
	return removeSecondaryIPToInterface(staticVIP, interfaceName)
}

// getInterfaceForNetwork returns interface name within same network as staticIP
func getInterfaceForNetwork(staticIPAddr string) (string, error) {

	staticIP, _, err := net.ParseCIDR(staticIPAddr)
	if err != nil {
		return "", err
	}
	route, err := netlink.RouteGet(staticIP)
	if err != nil {
		return "", err
	}
	if len(route) < 1 {
		return "", fmt.Errorf("failed to get route err %v", err)
	}
	return getInterfaceName(staticIP, route[0].LinkIndex)
}

// getInterfaceName returns interface if interface and staicIP are in same subnet
func getInterfaceName(staticIP net.IP, routeIndex int) (string, error) {

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Index == routeIndex {
			addresses, err := iface.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addresses {
				if existsInSameNetwork(addr, staticIP) {
					return iface.Name, nil
				}
			}
		}
	}
	return "", fmt.Errorf("failed to find interface")
}

func existsInSameNetwork(addr net.Addr, ip net.IP) bool {

	_, inet, err := net.ParseCIDR(addr.String())
	if err != nil {
		return false
	}
	return inet.Contains(ip)
}

func addSecondaryIPToInterface(staticIP string, name string) error {

	addr, err := netlink.ParseAddr(staticIP)
	if err != nil {
		return fmt.Errorf("failed to parse static IP err %v", err)
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to get link by name %v", err)
	}
	return netlink.AddrAdd(link, addr)
}

func removeSecondaryIPToInterface(staticIP string, name string) error {

	addr, err := netlink.ParseAddr(staticIP)
	if err != nil {
		return fmt.Errorf("failed to parse static IP err %v", err)
	}
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("failed to get link by name %v", err)
	}
	return netlink.AddrDel(link, addr)
}
