package auth

import (
	"context"
)

// InspectionTokenCreds implements credentials.PerRPCCredentials for the inspection token.
type InspectionTokenCreds struct {
	InspectionToken string
	UserAgentHeader string
}

// GetRequestMetadata adds the inspection token to the gRPC metadata.
func (c *InspectionTokenCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if c.InspectionToken == "" {
		return map[string]string{"user-agent": c.UserAgentHeader}, nil
	}
	return map[string]string{
		"x-goog-iam-authorization-token": c.InspectionToken,
		"user-agent":                     c.UserAgentHeader,
	}, nil
}

// RequireTransportSecurity indicates that this credential requires a secure transport.
func (c *InspectionTokenCreds) RequireTransportSecurity() bool {
	return true
}
