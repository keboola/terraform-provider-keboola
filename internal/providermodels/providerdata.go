// Package providermodels contains shared data models used across the provider
package providermodels

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

// the provider and resources without creating circular dependencies.
type ProviderData struct {
	Client     *keboola.AuthorizedAPI
	Token      *keboola.Token
	Components []*keboola.Component
}

// GetClient returns the keboola API client.
func (p *ProviderData) GetClient() *keboola.AuthorizedAPI {
	return p.Client
}

// GetToken returns the token details.
func (p *ProviderData) GetToken() *keboola.Token {
	return p.Token
}

// IsValid checks if the provider data is valid and logs info about it.
func (p *ProviderData) IsValid(ctx context.Context) bool {
	if p == nil {
		tflog.Error(ctx, "ProviderData is nil")

		return false
	}

	if p.Client == nil {
		tflog.Error(ctx, "ProviderData.Client is nil")

		return false
	}

	if p.Token == nil {
		tflog.Error(ctx, "ProviderData.Token is nil")

		return false
	}

	tflog.Info(ctx, "ProviderData is valid", map[string]interface{}{
		"projectId": p.Token.ProjectID(),
	})

	return true
}
