package license

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/obot-platform/obot/apiclient/types"
	gatewayclient "github.com/obot-platform/obot/pkg/gateway/client"
	gatewaydb "github.com/obot-platform/obot/pkg/gateway/db"
	storageservices "github.com/obot-platform/obot/pkg/storage/services"
)

func TestProviderGrantsEverything(t *testing.T) {
	provider, err := NewProvider(t.Context(), nil, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := provider.HasValidLicense(t.Context()); err != nil || !ok {
		t.Fatalf("HasValidLicense() = %v, %v, want true, nil", ok, err)
	}

	entitlements, err := provider.Entitlements(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, entitlement := range []string{
		EnterpriseEntitlement,
		EnterpriseAuthProvidersEntitlement,
		EnterpriseModelProvidersEntitlement,
	} {
		if !slices.Contains(entitlements, entitlement) {
			t.Errorf("Entitlements() = %v, missing %s", entitlements, entitlement)
		}
	}
	for _, entitlement := range []string{CommunityEntitlement, CloudEntitlement} {
		if slices.Contains(entitlements, entitlement) {
			t.Errorf("Entitlements() = %v, want no %s", entitlements, entitlement)
		}
	}

	if missing, err := provider.MissingEntitlements(t.Context(), []string{"ANY_ENTITLEMENT"}); err != nil || len(missing) != 0 {
		t.Fatalf("MissingEntitlements() = %v, %v, want none", missing, err)
	}
	if err := provider.RequireEntitlements(t.Context(), []string{"ANY_ENTITLEMENT"}); err != nil {
		t.Fatalf("RequireEntitlements() error = %v, want nil", err)
	}
	if violations, err := provider.GetLicenseViolations(t.Context(), nil); err != nil || len(violations) != 0 {
		t.Fatalf("GetLicenseViolations() = %v, %v, want none", violations, err)
	}

	if limit, err := provider.UserLimit(t.Context()); err != nil || !limit.Unlimited {
		t.Fatalf("UserLimit() = %+v, %v, want unlimited", limit, err)
	}
	if limit, err := provider.DeviceLimit(t.Context()); err != nil || !limit.Unlimited {
		t.Fatalf("DeviceLimit() = %+v, %v, want unlimited", limit, err)
	}
}

func TestProviderLicenseKeyOnlyFromConfiguration(t *testing.T) {
	provider, err := NewProvider(t.Context(), nil, Config{LicenseKey: " configured-key "})
	if err != nil {
		t.Fatal(err)
	}

	if key, err := provider.LicenseKey(t.Context()); err != nil || key != "configured-key" {
		t.Fatalf("LicenseKey() = %q, %v, want configured-key", key, err)
	}
	if !provider.LicenseKeyViaConfiguration() {
		t.Fatal("LicenseKeyViaConfiguration() = false, want true")
	}
	if err := provider.SetLicenseKey(t.Context(), "other"); !errors.Is(err, ErrLicenseKeyViaConfiguration) {
		t.Fatalf("SetLicenseKey() error = %v, want %v", err, ErrLicenseKeyViaConfiguration)
	}
	if err := provider.SetCommunityLicenseKey(t.Context(), "other"); !errors.Is(err, ErrLicenseKeyViaConfiguration) {
		t.Fatalf("SetCommunityLicenseKey() error = %v, want %v", err, ErrLicenseKeyViaConfiguration)
	}
	if err := provider.RemoveLicenseKey(t.Context()); !errors.Is(err, ErrLicenseKeyViaConfiguration) {
		t.Fatalf("RemoveLicenseKey() error = %v, want %v", err, ErrLicenseKeyViaConfiguration)
	}
}

func TestMachineFingerprintUsesPersistedInstallationIdentity(t *testing.T) {
	client := newTestLicenseGatewayClient(t)
	if _, err := client.SetProperty(t.Context(), LicenseMachineIDPropertyKey, "existing-installation"); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		provider, err := NewProvider(t.Context(), client, Config{})
		if err != nil {
			t.Fatal(err)
		}

		if provider.MachineFingerprint() != "existing-installation" {
			t.Fatal("replaced persisted machine fingerprint")
		}
	}
}

func TestUnlicensedReportsNoEntitlements(t *testing.T) {
	entitlements, err := Unlicensed{}.Entitlements(t.Context())
	if err != nil || len(entitlements) != 0 {
		t.Fatalf("Entitlements() = %v, %v, want none", entitlements, err)
	}
	if got := GetDistributionFromEntitlements(entitlements); got != types.ProductTelemetryDistributionUnregistered {
		t.Fatalf("GetDistributionFromEntitlements() = %q, want %q", got, types.ProductTelemetryDistributionUnregistered)
	}
}

func TestGetDistributionFromEntitlements(t *testing.T) {
	for _, test := range []struct {
		name         string
		entitlements []string
		want         types.ProductTelemetryDistribution
	}{
		{
			name: "unregistered",
			want: types.ProductTelemetryDistributionUnregistered,
		},
		{
			name:         "registered",
			entitlements: []string{CommunityEntitlement},
			want:         types.ProductTelemetryDistributionRegistered,
		},
		{
			name:         "enterprise",
			entitlements: []string{CommunityEntitlement, EnterpriseEntitlement},
			want:         types.ProductTelemetryDistributionEnterprise,
		},
		{
			name:         "cloud",
			entitlements: []string{CommunityEntitlement, EnterpriseEntitlement, CloudEntitlement},
			want:         types.ProductTelemetryDistributionCloud,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := GetDistributionFromEntitlements(test.entitlements); got != test.want {
				t.Fatalf("GetDistributionFromEntitlements() = %q, want %q", got, test.want)
			}
		})
	}
}

func newTestLicenseGatewayClient(t *testing.T) *gatewayclient.Client {
	t.Helper()

	storageServices, err := storageservices.New(storageservices.Config{DSN: "sqlite://:memory:"})
	if err != nil {
		t.Fatalf("failed to create storage services: %v", err)
	}
	database, err := gatewaydb.New(storageServices.DB.DB, storageServices.DB.SQLDB, true)
	if err != nil {
		t.Fatalf("failed to create gateway database: %v", err)
	}
	if err := database.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate gateway database: %v", err)
	}

	gatewayClient := gatewayclient.New(t.Context(), database, nil, nil, nil, time.Hour, 10, 0, 0, 0, false)
	t.Cleanup(func() {
		if err := gatewayClient.Close(); err != nil {
			t.Errorf("failed to close gateway client: %v", err)
		}
	})
	return gatewayClient
}
