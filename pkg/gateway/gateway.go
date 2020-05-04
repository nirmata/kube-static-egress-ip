package gateway

import (
	"errors"
	"fmt"

	"github.com/coreos/go-iptables/iptables"
	"github.com/golang/glog"
	ipset "github.com/janeczku/go-ipset/ipset"
)

// EgressGateway configures the gateway node (based on the `staticegressip` CRD object)
// with SNAT rules to NAT egress traffic from the pods that need a static egress IP
type EgressGateway struct {
	ipt *iptables.IPTables
}

const (
	defaultTimeOut            = 0
	defaultNATIptable         = "nat"
	egressGatewayNATChainName = "STATIC-EGRESS-NAT-CHAIN"
	defaultEgressChainName    = "STATIC-EGRESS-IP-CHAIN"
	egressGatewayFWChainName  = "STATIC-EGRESS-FORWARD-CHAIN"
	defaultPostRoutingChain   = "POSTROUTING"
	staticEgressIPFWMARK      = "1000"
)

// NewEgressGateway is a constructor for EgressGateway interface
func NewEgressGateway() (*EgressGateway, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to locate iptables: %v", err)
	}
	return &EgressGateway{ipt: ipt}, nil
}

func (gateway *EgressGateway) Setup() error {

	// setup a chain to hold rules to accept forwarding traffic from director nodes with
	// out which default policy FORWARD chain of filter table drops the packet
	err := gateway.createChainIfNotExist("filter", egressGatewayFWChainName)
	if err != nil {
		return errors.New("Failed to add a chain in filter table required to permit forwarding traffic from director nodes" + err.Error())
	}

	ruleSpec := []string{"-j", egressGatewayFWChainName}
	hasRule, err := gateway.ipt.Exists("filter", "FORWARD", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in FORWARD chain of filter table to permit forward traffic from the directors" + err.Error())
	}
	if !hasRule {
		err = gateway.ipt.Append("filter", "FORWARD", ruleSpec...)
		if err != nil {
			return errors.New("Failed to add iptables command to permit traffic from directors to be forwrded in filter chain" + err.Error())
		}
	}

	// setup a chain in nat table to bypass run through the rules to snat traffic from the pods that need static egress ip
	err = gateway.ipt.NewChain("nat", egressGatewayNATChainName)
	if err != nil && err.(*iptables.Error).ExitStatus() != 1 {
		return errors.New("Failed to add a " + egressGatewayNATChainName + " chain in NAT table" + err.Error())
	}
	ruleSpec = []string{"-j", egressGatewayNATChainName}
	hasRule, err = gateway.ipt.Exists("nat", "POSTROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify  rule exists in POSTROUTING chain of nat table due to " + err.Error())
	}
	if !hasRule {
		err = gateway.ipt.Insert("nat", "POSTROUTING", 1, ruleSpec...)
		if err != nil {
			return errors.New("Failed to run iptables command to add a rule to jump to STATIC-EGRESS-NAT-CHAIN chain due to " + err.Error())
		}
	}

	return nil
}

// AddStaticIptablesRule adds iptables rule for SNAT, creates source
// and destination IPsets. IPs can then be dynamically added to these IPsets.
func (gateway *EgressGateway) AddStaticIptablesRule(setName string, sourceIPs []string, destinationIP, egressIP string) error {

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
	glog.Infof("Added ips %v to the ipset name: %s", sourceIPs, setName)

	ruleSpec := []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err := gateway.ipt.Exists("filter", egressGatewayFWChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in " + egressGatewayFWChainName + " chain of filter table" + err.Error())
	}
	if !hasRule {
		err = gateway.ipt.Append("filter", egressGatewayFWChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to add iptables command to ACCEPT traffic from director nodes to get forrwarded" + err.Error())
		}
	}
	glog.Infof("Added rules in filter table FORWARD chain to permit traffic")

	ruleSpec = []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "SNAT", "--to-source", egressIP}
	if err := gateway.insertRule(defaultNATIptable, egressGatewayNATChainName, 1, ruleSpec...); err != nil {
		return fmt.Errorf("failed to insert rule to chain %v err %v", defaultPostRoutingChain, err)
	}

	// create iptables rule in mangle table PREROUTING chain to match to inbound return traffic to static egress IP
	// matching  destinationIP then fwmark the packets
	ruleSpec = []string{"-s", destinationIP, "-d", egressIP, "-j", "MARK", "--set-mark", staticEgressIPFWMARK}
	hasRule, err = gateway.ipt.Exists("mangle", "PREROUTING", ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in PREROUTING chain of mangle table to fwmark inbound return traffic to static egress IP" + err.Error())
	}
	if !hasRule {
		err = gateway.ipt.Insert("mangle", "PREROUTING", 1, ruleSpec...)
		if err != nil {
			return errors.New("Failed to add rule in PREROUTING chain of mangle table to fwmark inbound return traffic to static egress IP" + err.Error())
		}
		glog.Infof("added rule in PREROUTING chain of mangle table to fwmark inbound return traffic to static egress IP")
	}
	glog.Infof("iptables rule in mangle table PREROUTING chain to match src to ipset")

	return nil
}

// DeleteStaticIptablesRule clears IPtables rules added by AddStaticIptablesRule
func (gateway *EgressGateway) DeleteStaticIptablesRule(setName string, destinationIP, egressIP string) error {

	// delete rule in NAT postrouting to SNAT traffic
	ruleSpec := []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "SNAT", "--to-source", egressIP}
	if err := gateway.deleteRule(defaultNATIptable, egressGatewayNATChainName, ruleSpec...); err != nil {
		return fmt.Errorf("failed to delete rule in chain %v err %v", egressGatewayNATChainName, err)
	}

	// delete rule in FORWARD chain of filter table
	ruleSpec = []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err := gateway.ipt.Exists("filter", egressGatewayFWChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in " + egressGatewayFWChainName + " chain of filter table" + err.Error())
	}
	if hasRule {
		err = gateway.ipt.Delete("filter", egressGatewayFWChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete iptables command to ACCEPT traffic from director nodes to get forwarded" + err.Error())
		}
	}

	set, err := ipset.New(setName, "hash:ip", &ipset.Params{})
	if err != nil {
		return errors.New("Failed to get ipset with name " + setName + " due to %" + err.Error())
	}
	err = set.Destroy()
	if err != nil {
		return errors.New("Failed to delete ipset due to " + err.Error())
	}

	return nil
}

// DeleteStaticIptablesRule clears IPtables rules added by AddStaticIptablesRule
func (gateway *EgressGateway) ClearStaticIptablesRule(setName string, destinationIP, egressIP string) error {

	// delete rule in NAT postrouting to SNAT traffic
	ruleSpec := []string{"-m", "set", "--match-set", setName, "src", "-d", destinationIP, "-j", "SNAT", "--to-source", egressIP}
	if err := gateway.deleteRule(defaultNATIptable, egressGatewayNATChainName, ruleSpec...); err != nil {
		return fmt.Errorf("failed to delete rule in chain %v err %v", egressGatewayNATChainName, err)
	}

	// delete rule in FORWARD chain of filter table
	ruleSpec = []string{"-m", "set", "--set", setName, "src", "-d", destinationIP, "-j", "ACCEPT"}
	hasRule, err := gateway.ipt.Exists("filter", egressGatewayFWChainName, ruleSpec...)
	if err != nil {
		return errors.New("Failed to verify rule exists in " + egressGatewayFWChainName + " chain of filter table" + err.Error())
	}
	if hasRule {
		err = gateway.ipt.Delete("filter", egressGatewayFWChainName, ruleSpec...)
		if err != nil {
			return errors.New("Failed to delete iptables command to ACCEPT traffic from director nodes to get forwarded" + err.Error())
		}
	}

	set, err := ipset.New(setName, "hash:ip", &ipset.Params{})
	if err != nil {
		return errors.New("Failed to get ipset with name " + setName + " due to %" + err.Error())
	}
	err = set.Destroy()
	if err != nil {
		return errors.New("Failed to delete ipset due to " + err.Error())
	}

	return nil
}

/*
// AddSourceIP
func (m *EgressGateway) AddSourceIP(ip string) error {
	return m.sourceIPSet.Add(ip, defaultTimeOut)
}

// DelSourceIP
func (m *EgressGateway) DelSourceIP(ip string) error {
	return m.sourceIPSet.Del(ip)
}

// AddDestIP
func (m *EgressGateway) AddDestIP(ip string) error {
	return m.destIPSet.Add(ip, defaultTimeOut)
}

// DelDestIP
func (m *EgressGateway) DelDestIP(ip string) error {
	return m.destIPSet.Del(ip)
}
*/
// CreateChainIfNotExist will check if chain exist, if not it will create one.
func (m *EgressGateway) createChainIfNotExist(table, chain string) error {
	err := m.ipt.NewChain(table, chain)
	if err == nil {
		return nil // chain didn't exist, created now.
	}
	eerr, eok := err.(*iptables.Error)
	if eok && eerr.ExitStatus() == 1 {
		return nil // chain already exists
	}
	return err
}

func (m *EgressGateway) deleteChain(table, chain string) error {
	return m.ipt.DeleteChain(table, chain)
}

func (m *EgressGateway) insertRule(table, chain string, pos int, ruleSpec ...string) error {
	exist, err := m.ipt.Exists(table, chain, ruleSpec...)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	return m.ipt.Insert(table, chain, pos, ruleSpec...)
}

func (m EgressGateway) deleteRule(table, chain string, ruleSpec ...string) error {
	return m.ipt.Delete(table, chain, ruleSpec...)
}
