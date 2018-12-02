package iptables

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
	"github.com/nirmata/kube-static-egress-ip/enforcer"
)

type manager struct {
	ipt *iptables.IPTables
	// add ipset info
}

// NewSourceIPModifier is a constructor for SourceIPModifier
func NewSourceIPModifier() (enforcer.SourceIPModifier, error) {
	ipt, err := iptables.New()
	if err != nil {
		return nil, fmt.Errorf("failed to locate iptables: %v", err)
	}
	return &manager{ipt: ipt}, nil
}

func (m *manager) AddStaticIptablesRule(srcSetName, dstSetName, NatVipIP string) error {
	return nil
}

// ClearIptablesRule clears IPtables rules added by AddNatRules
func (m *manager) ClearIptablesRule(srcSetName, dstSetName, NatVipIP string) error {
	return nil
}

// ListIptablesRule lists Nat rules configured
func (m *manager) ListIptablesRule() ([]string, error) {
	return nil, nil
}

// AddSourceIP
func (m *manager) AddSourceIP(ip string) error {
	return nil
}

// DelSourceIP
func (m *manager) DelSourceIP(ip string) error {
	return nil
}

// AddDestIP
func (m *manager) AddDestIP(ip string) error {
	return nil
}

// DelDestIP
func (m *manager) DelDestIP(ip string) error {
	return nil
}
