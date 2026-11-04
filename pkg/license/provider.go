package license

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"uuid"

	"github.com/obot-platform/obot/pkg/gateway/client"
)

const (
	// LicenseMachineIDPropertyKey is the database property key used to persist the installation's machine fingerprint.
	LicenseMachineIDPropertyKey = "obot-license-machine-id"

	// EnterpriseAuthProvidersEntitlement enables enterprise auth providers.
	EnterpriseAuthProvidersEntitlement = "OBOT_ENTERPRISE_AUTH_PROVIDERS"

	// EnterpriseEntitlement enables enterprise edition.
	EnterpriseEntitlement = "OBOT_ENTERPRISE"

	// CommunityEntitlement enables community edition.
	CommunityEntitlement = "OBOT_COMMUNITY"

	// CloudEntitlement identifies Obot Cloud deployments.
	CloudEntitlement = "OBOT_CLOUD"

	// EnterpriseModelProvidersEntitlement enables enterprise model providers.
	EnterpriseModelProvidersEntitlement = "OBOT_ENTERPRISE_MODEL_PROVIDERS"
)

var (
	// grantedEntitlements are reported as held by every installation. CommunityEntitlement is left out because the
	// UI treats it as the Community edition.
	grantedEntitlements = []string{
		EnterpriseEntitlement,
		EnterpriseAuthProvidersEntitlement,
		EnterpriseModelProvidersEntitlement,
	}

	// ErrNotConfigured indicates license validation was requested without a license key.
	ErrNotConfigured = errors.New("license provider is not configured")

	// ErrLicenseKeyViaConfiguration indicates the license key is managed by startup configuration.
	ErrLicenseKeyViaConfiguration = errors.New("license key is configured at startup")

	// ErrInvalidLicense indicates the provided license key was rejected.
	ErrInvalidLicense = errors.New("license key is invalid")

	// ErrLicenseCheckFailed indicates the license key couldn't be checked.
	ErrLicenseCheckFailed = errors.New("failed to check license key")
)

// Config contains the license settings for this Obot installation.
type Config struct {
	LicenseKey string `usage:"Obot license key, only used to authenticate to Obot-hosted services such as the MCP tester model proxy; all features are enabled without one"`
}

// Provider grants every entitlement with no user or device limits. It keeps the
// configured license key only to pass to Obot-hosted services.
type Provider struct {
	machineFingerprint   string
	configuredLicenseKey string
}

// NewProvider creates a license provider that grants every entitlement.
func NewProvider(ctx context.Context, gatewayClient *client.Client, config Config) (*Provider, error) {
	machineFingerprint, err := ensureMachineFingerprint(ctx, gatewayClient)
	if err != nil {
		return nil, err
	}

	return &Provider{
		machineFingerprint:   machineFingerprint,
		configuredLicenseKey: strings.TrimSpace(config.LicenseKey),
	}, nil
}

func ensureMachineFingerprint(ctx context.Context, gatewayClient *client.Client) (string, error) {
	if gatewayClient == nil {
		return uuid.New().String(), nil
	}

	property, err := gatewayClient.GetOrCreateProperty(ctx, LicenseMachineIDPropertyKey, uuid.New().String())
	if err != nil {
		return "", fmt.Errorf("failed to ensure license machine ID: %w", err)
	}
	return property.Value, nil
}

// LicenseKey returns the configured license key, or an empty string when none is configured.
func (p *Provider) LicenseKey(context.Context) (string, error) {
	return p.configuredLicenseKey, nil
}

// MachineFingerprint returns the existing persisted installation identity.
func (p *Provider) MachineFingerprint() string {
	return p.machineFingerprint
}

// LicenseKeyViaConfiguration always reports true: the license key can only be set at startup.
func (p *Provider) LicenseKeyViaConfiguration() bool {
	return true
}

func (p *Provider) SetLicenseKey(context.Context, string) error {
	return ErrLicenseKeyViaConfiguration
}

func (p *Provider) SetCommunityLicenseKey(context.Context, string) error {
	return ErrLicenseKeyViaConfiguration
}

func (p *Provider) RemoveLicenseKey(context.Context) error {
	return ErrLicenseKeyViaConfiguration
}

func (p *Provider) Validate(context.Context) error {
	return nil
}

func (p *Provider) HasValidLicense(context.Context) (bool, error) {
	return true, nil
}

func (p *Provider) Entitlements(context.Context) ([]string, error) {
	entitlements := slices.Clone(grantedEntitlements)
	slices.Sort(entitlements)
	return entitlements, nil
}
