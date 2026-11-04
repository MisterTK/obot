package license

import (
	"context"
	"slices"

	"github.com/obot-platform/obot/apiclient/types"
	gatewayclient "github.com/obot-platform/obot/pkg/gateway/client"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Violation describes a configured provider that requires license entitlements
// that are not currently available.
type Violation struct {
	Type                 string   `json:"type"`
	Namespace            string   `json:"namespace"`
	Name                 string   `json:"name"`
	RequiredEntitlements []string `json:"requiredEntitlements"`
	MissingEntitlements  []string `json:"missingEntitlements"`
	Message              string   `json:"message"`
}

// Unlicensed reports no entitlements. Use it to describe the installation to Obot, which issued none of the
// entitlements Provider grants.
type Unlicensed struct{}

// GetDistributionFromEntitlements returns the product distribution represented by the entitlements.
func GetDistributionFromEntitlements(entitlements []string) types.ProductTelemetryDistribution {
	switch {
	case slices.Contains(entitlements, CloudEntitlement):
		return types.ProductTelemetryDistributionCloud
	case slices.Contains(entitlements, EnterpriseEntitlement):
		return types.ProductTelemetryDistributionEnterprise
	case slices.Contains(entitlements, CommunityEntitlement):
		return types.ProductTelemetryDistributionRegistered
	default:
		return types.ProductTelemetryDistributionUnregistered
	}
}

func (Unlicensed) Entitlements(context.Context) ([]string, error) {
	return nil, nil
}

// MissingEntitlements always returns no missing entitlements.
func (p *Provider) MissingEntitlements(context.Context, []string) ([]string, error) {
	return nil, nil
}

// RequireEntitlements always succeeds.
func (p *Provider) RequireEntitlements(context.Context, []string) error {
	return nil
}

// UserLimit always returns an unlimited user limit.
func (p *Provider) UserLimit(context.Context) (gatewayclient.UserLimit, error) {
	return gatewayclient.UserLimit{Unlimited: true}, nil
}

// DeviceLimit always returns an unlimited device limit.
func (p *Provider) DeviceLimit(context.Context) (gatewayclient.DeviceLimit, error) {
	return gatewayclient.DeviceLimit{Unlimited: true}, nil
}

// GetLicenseViolations always returns no violations.
func (p *Provider) GetLicenseViolations(context.Context, kclient.Client) ([]Violation, error) {
	return nil, nil
}
