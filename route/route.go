package route

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func DetectDefaultInterface() (string, error) {
	lst, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", err
	}
	_, ipnet, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		return "", err
	}
	sipnet := ipnet.String()
	for _, item := range lst {
		if item.Dst.String() != sipnet {
			continue
		}
		iface, err := netlink.LinkByIndex(item.LinkIndex)
		if err != nil {
			return "", err
		}
		ifacename := iface.Attrs().Name
		return ifacename, nil
	}
	return "", fmt.Errorf("unable to found default network interface")
}
