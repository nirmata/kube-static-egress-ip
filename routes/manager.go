package routes

import (
	"github.com/nirmata/kube-static-egress-ip/enforcer"
)

type manager struct {
}

// NewRouteModifier is a constructor for RouteModifier
func NewRouteModifier() (enforcer.RouteModifier, error) {
	return &manager{}, nil
}

// RouteToNode adds a route in a node to redirect traffic with destinationIP
// to a targetNode
func (m *manager) RouteToNode(destinationIP, targetNode string) error {
	return nil
}

// ClearRouteToNode removes the particular route rule
func (m *manager) ClearRouteToNode(destinationIP, targetNode string) error {
	return nil
}

// RuleExists returns true if it exists, else an error
func (m *manager) RuleExists(destinationIP, targetNode string) (bool, error) {
	return false, nil
}
