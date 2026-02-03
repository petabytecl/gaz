package gateway

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// HeaderTestSuite tests header matching functionality.
type HeaderTestSuite struct {
	suite.Suite
}

func TestHeaderTestSuite(t *testing.T) {
	suite.Run(t, new(HeaderTestSuite))
}

func (s *HeaderTestSuite) TestAllowedHeaders() {
	expected := []string{
		"authorization",
		"x-request-id",
		"x-correlation-id",
		"x-forwarded-for",
		"x-forwarded-host",
		"accept-language",
	}

	s.Require().Equal(expected, AllowedHeaders)
}

func (s *HeaderTestSuite) TestHeaderMatcher_Authorization() {
	key, ok := HeaderMatcher("Authorization")

	s.Require().True(ok)
	s.Require().Equal("authorization", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_XRequestID() {
	key, ok := HeaderMatcher("X-Request-ID")

	s.Require().True(ok)
	s.Require().Equal("x-request-id", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_XCorrelationID() {
	key, ok := HeaderMatcher("X-Correlation-ID")

	s.Require().True(ok)
	s.Require().Equal("x-correlation-id", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_XForwardedFor() {
	key, ok := HeaderMatcher("X-Forwarded-For")

	s.Require().True(ok)
	s.Require().Equal("x-forwarded-for", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_XForwardedHost() {
	key, ok := HeaderMatcher("X-Forwarded-Host")

	s.Require().True(ok)
	s.Require().Equal("x-forwarded-host", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_AcceptLanguage() {
	key, ok := HeaderMatcher("Accept-Language")

	s.Require().True(ok)
	s.Require().Equal("accept-language", key)
}

func (s *HeaderTestSuite) TestHeaderMatcher_CaseInsensitive() {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "authorization", "authorization"},
		{"UPPERCASE", "AUTHORIZATION", "authorization"},
		{"MixedCase", "Authorization", "authorization"},
		{"weird case", "AuThOrIzAtIoN", "authorization"},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			key, ok := HeaderMatcher(tc.input)
			s.Require().True(ok, "Header should be matched")
			s.Require().Equal(tc.expected, key)
		})
	}
}

func (s *HeaderTestSuite) TestHeaderMatcher_UnknownHeader() {
	// Unknown headers fall back to default grpc-gateway behavior.
	// The default behavior returns false for non-standard headers.
	key, ok := HeaderMatcher("X-Custom-Header")

	// Default matcher returns empty string and false for custom headers.
	s.Require().False(ok, "Custom headers should not be matched by default")
	s.Require().Empty(key, "Key should be empty for unmatched headers")
}

func (s *HeaderTestSuite) TestHeaderMatcher_ContentType() {
	// Content-Type is handled by grpc-gateway default matcher.
	key, ok := HeaderMatcher("Content-Type")

	// Default matcher should handle standard headers.
	s.Require().NotEmpty(key)
	_ = ok
}

func (s *HeaderTestSuite) TestHeaderMatcher_AllAllowedHeaders() {
	// Verify all headers in AllowedHeaders list are matched.
	for _, header := range AllowedHeaders {
		s.Run(header, func() {
			key, ok := HeaderMatcher(header)
			s.Require().True(ok, "Header %s should be matched", header)
			s.Require().Equal(header, key, "Key should match lowercase header")
		})
	}
}
