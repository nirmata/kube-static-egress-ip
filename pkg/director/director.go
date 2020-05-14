package director

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/golang/glog"
	"github.com/janeczku/go-ipset/ipset"
	utils "github.com/nirmata/kube-static-egress-ip/pkg/utils"
	"k8s.io/client-go/kubernetes"
)

const (
	customStaticEgressIPRouteTableID   = "99"
	customStaticEgressIPRouteTableName = "kube-static-egress-ip"
	staticEgressIPFWMARK               = "1000"
	bypassCNIMasquradeChainName        = "STATIC-EGRESS-BYPASS-CNI"
)

// EgressDirector manages routing rules needed on a node to redirect egress traffic from the pods that need
// a static egress IP to a node acting as egress gateway based on the `staticegressip` CRD object
type EgressDirector struct {
	ipt    *iptables.IPTables
	nodeIP string
}

// NewEgressDirector is a constructor for EgressDirector
func NewEgressDirector(clientset kubernetes.Interface) (*EgressDirector, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to locate iptables: %v", err)
	}

	nodeObject, err := utils.GetNodeObject(clientset, "")
	if err != nil {
		return nil, err
	}
	nodeIP, err := utils.GetNodeIP(nodeObject)
	if err != nil {
		return nil, errors.New("Failed to get node IP due to " + err.Error())
	}
	return &EgressDirector{nodeIP: nodeIP.String(), ipt: ipt}, nil
}

// Setup sets up the node with one-time basic settings needed for director functionality
func (d *EgressDirector) Setup() error {

	// create custom routing table for directing the traffic from director nodes to the gateway node
	b, err := ioutil.ReadFile("/etc/iproute2/rt_tables")
	if err != nil {
		return errors.New("Failed to add custom routing table in /etc/iproute2/rt_tables needed for policy routing for directing traffing to egress gateway" + err.Error())
	}
	if !strings.Contains(string(b), customStaticEgressIPRouteTableName) {
		f, err := os.OpenFile("/etc/iproute2/rt_tables", os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return errors.New("Failed to open /etc/iproute2/rt_tables to verify custom routing table " + customStaticEgressIPRouteTableName + " required for static egress IP functionality due to " + err.Error())
		}
		defer f.Close()
		if _, err = f.WriteString(customStaticEgressIPRouteTableID + " " + customStaticEgressIPRouteTableName + "\n"); err != nil {
			return errors.New("Failed to add custom routing table " + customStaticEgressIPRouteTableName + " in /etc/iproute2/rt_tables needed for policy routing due to " + err.Error())
		}
	}

	// create policy based routing (ip rule) to lookup the custom routing table for FWMARK packets
	out, err := exec.Command("ip", "rule", "list").Output()
	if err != nil {
		return errors.New("Failed to verify if `ip rule` exists due to: " + err.Error())
	}
	if !strings.Contains(string(out), customStaticEgressIPRouteTableName) {
		err = exec.Command("ip", "rule", "add", "prio", "32764", "fwmark", staticEgressIPFWMARK, "table", customStaticEgressIPRouteTableName).Run()
		if err != nil {
			return errors.New("Failed to add policy rule to lookup traffic marked with fwmark " + staticEgressIPFWMARK + " to the custom " + " routing table due to " + err.Error())
		}
	}

	// setup a chain in nat table to bypass the CNI masqurade for the traffic bound to egress gateway
	err = d.ipt.NewChain("nat", bypassCNIMasquradeChainName)
	if err != nil && err.(*iptables.Error).ExitStatus() != 1 {
		return errors.New("Failed to add a " + bypassCNIMasquradeChainName + " chain in NAT table required to bypass CNI masqurading due to" + err.Error())
	}
	ruleSpec := []string{"-j", bypassCNIMasquradeChainName}
	hasRule, err := d.ipt.Exists("nat", "POSTROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify bypass CNI masqurade rule exists in POSTROUTING chain of nat table due to " + err.Error())
	}
	if !hasRule {
		err = d.ipt.Insert("nat", "POSTROUTING", 1, ruleSpec...)
		if err != nil {
			return errors.New("Failed to run iptables command to add a rule to jump to STATIC_EGRESSIP_BYPASS_CNI_MASQURADE chain due to " + err.Error())
		}
	}

	glog.Infof("Node has been setup for static egress IP director functionality successfully.")

	return nil
}

// AddRouteToGateway adds a routes on the director node to redirect traffic from a set of pod IP's
// (selected by service name in the rule of staticegressip CRD object) to a specific
// destination CIDR to be directed to egress gateway node
func (d *EgressDirector) AddRouteToGateway(setName string, sourceIPs []string, destinationIP, egressGateway string) error {

	// create IPset for the set of sourceIP's
	set, err := ipset.New(setName, "hash:ip", &ipset.Params{})
	if err != nil {
		return errors.New("Failed to create ipset with name " + setName + " due to %" + err.Error())
	}
	glog.Infof("Created ipset name: %s", setName)

	// add IP's that need to be part of the ipset
	for _, ip := range sourceIPs {
		err = set.Add(ip, 0)
		if err != nil {
			return errors.New("Failed to add an ip " + ip + " into ipset with name " + setName + " due to %" + err.Error())
		}
	}
	glog.Infof("Added ips %v to the ipset name: %s", sourceIPs, setName)

	// create iptables rule in mangle table PREROUTING chain to match src to ipset created and destination
	// matching  destinationIP then fwmark the packets
	ruleSpec := []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err := d.ipt.Exists("mangle", "PREROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
	}
	if !hasRule {
		err = d.ipt.Insert("mangle", "PREROUTING", 1, ruleSpec...)
		if err != nil {
			return errors.New("Failed to add rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
		}
		glog.Infof("added rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP")
	}
	glog.Infof("iptables rule in mangle table PREROUTING chain to match src to ipset")

	ruleSpec = []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err = d.ipt.Exists("nat", bypassCNIMasquradeChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in BYPASS_CNI_MASQURADE chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if !hasRule {
		err = d.ipt.Append("nat", bypassCNIMasquradeChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to run iptables command to add a rule to ACCEPT traffic in BYPASS_CNI_MASQURADE chain" + err.Error())
		}
	}

	// create a tunnel interface to gateway node if does not exist
	tunnelName := "tun" + strings.Replace(egressGateway, ".", "", -1)
	out, err := exec.Command("ip", "link", "list").Output()
	if err != nil {
		return errors.New("Failed to verify required tunnel to gatewat exists. " + err.Error())
	}
	if !strings.Contains(string(out), tunnelName) {
		// ip tunnel add seip mode gre remote 192.168.1.102 local 192.168.1.101
		if err = exec.Command("ip", "tunnel", "add", tunnelName, "mode", "gre", "remote", egressGateway, "local", d.nodeIP).Run(); err != nil {
			return errors.New("Failed to tunnel interface to gateway node due to: " + err.Error())
		}
	}
	if err = exec.Command("ip", "link", "set", "up", tunnelName).Run(); err != nil {
		return errors.New("Failed to set tunnel interface to up due to: " + err.Error())
	}

	// add routing entry in custom routing table to forward destinationIP to egressGateway
	out, err = exec.Command("ip", "route", "list", "table", customStaticEgressIPRouteTableName).Output()
	if err != nil {
		return errors.New("Failed to verify required default route to gatewat exists. " + err.Error())
	}

	destAddr, _, err := net.ParseCIDR(destinationIP)
	if err != nil {
		if net.ParseIP(destinationIP) == nil {

		}
	}

	if !strings.Contains(string(out), destinationIP) && !strings.Contains(string(out), destAddr.String()) {
		if err = exec.Command("ip", "route", "add", destinationIP, "dev", tunnelName, "table", customStaticEgressIPRouteTableName).Run(); err != nil {
			return errors.New("Failed to add route in custom route table due to: " + err.Error())
		}
	}

	glog.Infof("added routing entry in custom routing table to forward destinationIP to egressGateway")

	return nil
}

// DeleteRouteToGateway removes the route routes on the director node to redirect traffic to gateway node
func (d *EgressDirector) DeleteRouteToGateway(setName string, destinationIP, egressGateway string) error {

	set, err := ipset.New(setName, "hash:ip", &ipset.Params{})
	if err != nil {
		return errors.New("Failed to get ipset with name " + setName + " due to %" + err.Error())
	}

	// create iptables rule in mangle table PREROUTING chain to match src to ipset created and destination
	// matching  destinationIP then fwmark the packets
	ruleSpec := []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err := d.ipt.Exists("mangle", "PREROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("mangle", "PREROUTING", ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
		}
		glog.Infof("deleted rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP")
	}

	ruleSpec = []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err = d.ipt.Exists("nat", bypassCNIMasquradeChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in BYPASS_CNI_MASQURADE chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("nat", bypassCNIMasquradeChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete iptables command to add a rule to ACCEPT traffic in BYPASS_CNI_MASQURADE chain" + err.Error())
		}
	}

	// add routing entry in custom routing table to forward destinationIP to egressGateway
	out, err := exec.Command("ip", "route", "list", "table", customStaticEgressIPRouteTableName).Output()
	if err != nil {
		return errors.New("Failed to verify required default route to gatewat exists. " + err.Error())
	}
	if !strings.Contains(string(out), destinationIP) {
		if err = exec.Command("ip", "route", "delete", destinationIP, "via", egressGateway, "table", customStaticEgressIPRouteTableName).Run(); err != nil {
			return errors.New("Failed to delete route in custom route table due to: " + err.Error())
		}
		glog.Infof("deleted route")
	}

	err = set.Destroy()
	if err != nil {
		return errors.New("Failed to delete ipset due to " + err.Error())
	}

	return nil
}

func (d *EgressDirector) ClearStaleRouteToGateway(setName string, destinationIP, egressGateway string) error {

	// create iptables rule in mangle table PREROUTING chain to match src to ipset created and destination
	// matching  destinationIP then fwmark the packets
	ruleSpec := []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err := d.ipt.Exists("mangle", "PREROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("mangle", "PREROUTING", ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
		}
		glog.Infof("deleted rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP")
	}

	ruleSpec = []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err = d.ipt.Exists("nat", bypassCNIMasquradeChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in BYPASS_CNI_MASQURADE chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("nat", bypassCNIMasquradeChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete iptables command to add a rule to ACCEPT traffic in BYPASS_CNI_MASQURADE chain" + err.Error())
		}
	}

	// add routing entry in custom routing table to forward destinationIP to egressGateway
	out, err := exec.Command("ip", "route", "list", "table", customStaticEgressIPRouteTableName).Output()
	if err != nil {
		return errors.New("Failed to verify required default route to gatewat exists. " + err.Error())
	}
	if !strings.Contains(string(out), destinationIP) {
		if err = exec.Command("ip", "route", "delete", destinationIP, "via", egressGateway, "table", customStaticEgressIPRouteTableName).Run(); err != nil {
			return errors.New("Failed to delete route in custom route table due to: " + err.Error())
		}
		glog.Infof("deleted route")
	}

	return nil
}
