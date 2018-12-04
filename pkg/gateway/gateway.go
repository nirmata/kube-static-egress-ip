package gateway

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
)

// EgressGateway configures the node which does SNAT for egress traffic
// from the pods that need a static egress IP
type EgressGateway interface {

	// AddStaticIptablesRule adds rules for traffic from srcSet > dstSet to be SNAT
	// with NatVipIP.
	AddStaticIptablesRule(srcSetName, dstSetName, NatVipIP string) error

	// ClearIptablesRule clears IPtables rules added by AddNatRules
	ClearIptablesRule(srcSetName, dstSetName, NatVipIP string) error

	// AddSourceIP
	AddSourceIP(ip string) error

	// DelSourceIP
	DelSourceIP(ip string) error

	// AddDestIP
	AddDestIP(ip string) error

	// DelDestIP
	DelDestIP(ip string) error
}

type manager struct {
	ipt         *iptables.IPTables
	sourceIPSet *IPSet
	destIPSet   *IPSet
}

const (
	sourceIPSetName         = "Static-egress-sourceIPSet"
	destinationIPSetName    = "Static-egress-DestinationIPSet"
	defaultTimeOut          = 0
	defaultNATIptable       = "nat"
	defaultEgressChainName  = "STATIC-EGRESS-IP-CHAIN"
	defaultPostRoutingChain = "POSTROUTING"
)

// NewEgressGateway is a constructor for EgressGateway interface
// if the setnames are empty it will use the default values
func NewEgressGateway(srcSetName, dstSetName string) (EgressGateway, error) {

	if srcSetName == "" {
		srcSetName = sourceIPSetName
	}
	if dstSetName == "" {
		dstSetName = destinationIPSetName
	}
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to locate iptables: %v", err)
	}
	srcSet, err := New(sourceIPSetName, "hash:ip", &Params{})
	if err != nil {
		return nil, fmt.Errorf("failed to create source Set err %v", err)
	}
	dstSet, err := New(destinationIPSetName, "hash:ip", &Params{})
	if err != nil {
		return nil, fmt.Errorf("failed to create source Set err %v", err)
	}
	return &manager{ipt: ipt, sourceIPSet: srcSet, destIPSet: dstSet}, nil
}

// AddStaticIptablesRule adds iptables rule for SNAT, creates source
// and destination IPsets. IPs can then be dynamically added to these IPsets.
func (m *manager) AddStaticIptablesRule(srcSetName, dstSetName, NatVipIP string) error {

	if srcSetName == "" {
		srcSetName = sourceIPSetName
	}
	if dstSetName == "" {
		dstSetName = destinationIPSetName
	}

	if err := m.createChainIfNotExist(defaultNATIptable, defaultEgressChainName); err != nil {
		return fmt.Errorf("failed to create chain err %v", err)
	}

	ruleSpec := []string{"-j", defaultEgressChainName, "-m", "SNAT egress-static-IP rule"}
	if err := m.insertRule(defaultNATIptable, defaultPostRoutingChain, 1, ruleSpec...); err != nil {
		return fmt.Errorf("failed to insert rule to chain %v err %v", defaultPostRoutingChain, err)
	}

	ruleSpec = []string{"-m", "set", "--match-set", srcSetName, "src", "-m", "set", "--match-set", dstSetName, "dst", "-j", "SNAT", "--to-source", NatVipIP}
	if err := m.insertRule(defaultNATIptable, defaultEgressChainName, 1, ruleSpec...); err != nil {
		return fmt.Errorf("failed to insert rule to chain %v err %v", defaultPostRoutingChain, err)
	}
	return nil
}

// ClearIptablesRule clears IPtables rules added by AddNatRules
func (m *manager) ClearIptablesRule(srcSetName, dstSetName, NatVipIP string) error {

	// Delete chain reference
	ruleSpec := []string{"-j", defaultEgressChainName, "-m", "SNAT egress-static-IP rule"}
	if err := m.deleteRule(defaultNATIptable, defaultPostRoutingChain, ruleSpec...); err != nil {
		return err
	}

	return m.deleteChain(defaultNATIptable, defaultEgressChainName)
}

// AddSourceIP
func (m *manager) AddSourceIP(ip string) error {
	return m.sourceIPSet.Add(ip, defaultTimeOut)
}

// DelSourceIP
func (m *manager) DelSourceIP(ip string) error {
	return m.sourceIPSet.Del(ip)
}

// AddDestIP
func (m *manager) AddDestIP(ip string) error {
	return m.destIPSet.Add(ip, defaultTimeOut)
}

// DelDestIP
func (m *manager) DelDestIP(ip string) error {
	return m.destIPSet.Del(ip)
}

// CreateChainIfNotExist will check if chain exist, if not it will create one.
func (m *manager) createChainIfNotExist(table, chain string) error {
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

func (m *manager) deleteChain(table, chain string) error {
	return m.ipt.DeleteChain(table, chain)
}

func (m *manager) insertRule(table, chain string, pos int, ruleSpec ...string) error {
	exist, err := m.ipt.Exists(table, chain, ruleSpec...)
	if err != nil {
		return err
	}
	if exist {
		return nil
	}
	return m.ipt.Insert(table, chain, pos, ruleSpec...)
}

func (m manager) deleteRule(table, chain string, ruleSpec ...string) error {
	return m.ipt.Delete(table, chain, ruleSpec...)
}
