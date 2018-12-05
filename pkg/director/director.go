package director

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/golang/glog"
	"github.com/janeczku/go-ipset/ipset"
)

const (
	customStaticEgressIPRouteTableID   = "99"
	customStaticEgressIPRouteTableName = "kube-static-egress-ip"
	staticEgressIPFWMARK               = "1000"
	bypassCNIMasquradeChainName        = "BYPASS_CNI_MASQURADE"
)

// EgressDirector configures routing on a node to redirect egress traffic
// from the pods that need a static egress IP to a node acting as egress gateway
type EgressDirector struct {
	ipt *iptables.IPTables
}

// NewEgressDirector is a constructor for SourceIPModifier
func NewEgressDirector() (*EgressDirector, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to locate iptables: %v", err)
	}

	return &EgressDirector{ipt: ipt}, nil
}

func (d *EgressDirector) Setup() error {

	// create custom routing table for directing the traffic to gateway
	b, err := ioutil.ReadFile("/etc/iproute2/rt_tables")
	if err != nil {
		return errors.New("Failed to setup policy routing required for directing traffing to egress gateway" + err.Error())
	}

	if !strings.Contains(string(b), customStaticEgressIPRouteTableName) {
		f, err := os.OpenFile("/etc/iproute2/rt_tables", os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return errors.New("Failed to setup policy routing required for static egress IP functionality due to " + err.Error())
		}
		defer f.Close()
		if _, err = f.WriteString(customStaticEgressIPRouteTableID + " " + customStaticEgressIPRouteTableName + "\n"); err != nil {
			return errors.New("Failed to setup policy routing required for statis egress IP functionality due to " + err.Error())
		}
	}

	// create policy based routing (ip rule) to lookup the custom routing table for FWMARK packets
	out, err := exec.Command("ip", "rule", "list").Output()
	if err != nil {
		return errors.New("Failed to verify if `ip rule` exists due to: " + err.Error())
	}
	if !strings.Contains(string(out), staticEgressIPFWMARK) {
		err = exec.Command("ip", "rule", "add", "prio", "32764", "fwmark", staticEgressIPFWMARK, "table", customStaticEgressIPRouteTableID).Run()
		if err != nil {
			return errors.New("Failed to add policy rule to lookup traffic marked with fwmard to the custom " +
				" routing table due to " + err.Error())
		}
	}

	// setup a chain in nat table to bypass the CNI masqurade for the traffic bound to egress gateway
	err = d.ipt.NewChain("nat", bypassCNIMasquradeChainName)
	if err != nil && err.(*iptables.Error).ExitStatus() != 1 {
		return errors.New("Failed to add a chain in NAT table required to bypass CNI masqurading" + err.Error())
	}
	args := []string{"-j", bypassCNIMasquradeChainName}
	hasRule, err := d.ipt.Exists("nat", "POSTROUTING", args...)
	if err != nil {
		return errors.New("Failed to verify rule exists in POSTROUTING chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if !hasRule {
		err = d.ipt.Insert("nat", "POSTROUTING", 1, args...)
		if err != nil {
			return errors.New("Failed to run iptables command to add a rule to jump to STATIC_EGRESSIP_BYPASS_CNI_MASQURADE chain" + err.Error())
		}
	}

	return nil
}

// RouteToGateway adds a route in on the node to redirect traffic from a set of pod IP's
// to a specific destination CIDR to be directed to egress gateway node
// example sources = [10.1.1.0, 10.1.1.10], destinationIP=192.168.56.22/32, egressGateway=10.150.11.112
func (d *EgressDirector) AddRouteToGateway(setName string, sourceIPs []string, destinationIP, egressGateway string) error {

	// create IPset from sourceIP's
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
	glog.Infof("Added ips %x to the ipset name: %s", sourceIPs, setName)

	// create iptables rule in mangle table PREROUTING chain to match src to ipset created and destination
	// matching  destinationIP then fwmark the packets
	args := []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err := d.ipt.Exists("mangle", "PREROUTING", args...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
	}
	if !hasRule {
		err = d.ipt.Insert("mangle", "PREROUTING", 1, args...)
		if err != nil {
			return errors.New("Failed to add rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
		}
		glog.Infof("added rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP")
	}
	glog.Infof("iptables rule in mangle table PREROUTING chain to match src to ipset")

	args = []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "RETURN"}
	hasRule, err = d.ipt.Exists("nat", bypassCNIMasquradeChainName, args...)
	if err != nil {
		return errors.New("Failed to verify rule exists in BYPASS_CNI_MASQURADE chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if !hasRule {
		err = d.ipt.Append("nat", bypassCNIMasquradeChainName, args...)
		if err != nil {
			return errors.New("Failed to run iptables command to add a rule to ACCEPT traffic in BYPASS_CNI_MASQURADE chain" + err.Error())
		}
	}

	// add routing entry in custom routing table to forward destinationIP to egressGateway
	out, err := exec.Command("ip", "route", "list", "table", customStaticEgressIPRouteTableName).Output()
	if err != nil {
		return errors.New("Failed to verify required default route to gatewat exists. " + err.Error())
	}
	if !strings.Contains(string(out), destinationIP) {
		if err = exec.Command("ip", "route", "add", destinationIP, "via", egressGateway, "table", customStaticEgressIPRouteTableName).Run(); err != nil {
			return errors.New("Failed to add route in custom route table due to: " + err.Error())
		}
		glog.Infof("added route")
	}
	glog.Infof("added routing entry in custom routing table to forward destinationIP to egressGateway")

	return nil
}

// DeleteRouteToGateway removes the particular route rule
func (d *EgressDirector) DeleteRouteToGateway(setName string, destinationIP, egressGateway string) error {

	set, err := ipset.New(setName, "hash:ip", &ipset.Params{})
	if err != nil {
		return errors.New("Failed to get ipset with name " + setName + " due to %" + err.Error())
	}

	// create iptables rule in mangle table PREROUTING chain to match src to ipset created and destination
	// matching  destinationIP then fwmark the packets
	args := []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err := d.ipt.Exists("mangle", "PREROUTING", args...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("mangle", "PREROUTING", args...)
		if err != nil {
			return errors.New("Failed to delete rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP" + err.Error())
		}
		glog.Infof("deleted rule in PREROUTING chain of mangle table to fwmark egress traffic that needs static egress IP")
	}

	args = []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "RETURN"}
	hasRule, err = d.ipt.Exists("nat", bypassCNIMasquradeChainName, args...)
	if err != nil {
		return errors.New("Failed to verify rule exists in BYPASS_CNI_MASQURADE chain of nat table to bypass the CNI masqurade" + err.Error())
	}
	if hasRule {
		err = d.ipt.Delete("nat", bypassCNIMasquradeChainName, args...)
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

// ListNatRules lists Nat rules configured
func (d *EgressDirector) ListNatRules() ([]string, error) {
	return nil, nil
}
