package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorTestSuite tests RFC 7807 error handling.
type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) TestProblemDetails_JSONSerialization() {
	problem := ProblemDetails{
		Type:     "https://example.com/problem/not-found",
		Title:    "NotFound",
		Status:   404,
		Detail:   "Resource not found",
		Instance: "req-123",
		Code:     "NOT_FOUND",
	}

	data, err := json.Marshal(problem)
	s.Require().NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	s.Require().NoError(err)

	s.Require().Equal("https://example.com/problem/not-found", decoded["type"])
	s.Require().Equal("NotFound", decoded["title"])
	s.Require().Equal(float64(404), decoded["status"])
	s.Require().Equal("Resource not found", decoded["detail"])
	s.Require().Equal("req-123", decoded["instance"])
	s.Require().Equal("NOT_FOUND", decoded["code"])
}

func (s *ErrorTestSuite) TestProblemDetails_OmitEmpty() {
	problem := ProblemDetails{
		Type:   "https://example.com/problem/error",
		Title:  "Error",
		Status: 500,
		// Detail, Instance, Code are empty.
	}

	data, err := json.Marshal(problem)
	s.Require().NoError(err)

	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	s.Require().NoError(err)

	// Verify omitempty works.
	_, hasDetail := decoded["detail"]
	_, hasInstance := decoded["instance"]
	_, hasCode := decoded["code"]

	s.Require().False(hasDetail, "Empty detail should be omitted")
	s.Require().False(hasInstance, "Empty instance should be omitted")
	s.Require().False(hasCode, "Empty code should be omitted")
}

func (s *ErrorTestSuite) TestErrorHandler_DevMode() {
	handler := ErrorHandler(true) // Dev mode.

	// Create test request with X-Request-ID.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-123")

	// Create recorder.
	rec := httptest.NewRecorder()

	// Create gRPC error.
	grpcErr := status.Error(codes.NotFound, "user not found: id=42")

	// Call error handler.
	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	// Verify response.
	s.Require().Equal(http.StatusNotFound, rec.Code)
	s.Require().Equal("application/problem+json", rec.Header().Get("Content-Type"))

	// Parse response body.
	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	s.Require().NoError(err)

	// In dev mode, detail includes the actual message.
	s.Require().Equal("user not found: id=42", problem.Detail)
	s.Require().Equal("NotFound", problem.Code)
	s.Require().Equal("test-request-123", problem.Instance)
	s.Require().Equal(http.StatusNotFound, problem.Status)
}

func (s *ErrorTestSuite) TestErrorHandler_ProdMode() {
	handler := ErrorHandler(false) // Prod mode.

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Create gRPC error with sensitive details.
	grpcErr := status.Error(codes.Internal, "database connection failed: password=secret123")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	// Verify response.
	s.Require().Equal(http.StatusInternalServerError, rec.Code)

	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	s.Require().NoError(err)

	// In prod mode, detail is generic.
	s.Require().Equal("Internal Server Error", problem.Detail, "Prod mode should hide sensitive details")
	s.Require().Empty(problem.Code, "Prod mode should not include code")
}

func (s *ErrorTestSuite) TestErrorHandler_StatusMapping_NotFound() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.NotFound, "not found")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal(http.StatusNotFound, rec.Code)
}

func (s *ErrorTestSuite) TestErrorHandler_StatusMapping_InvalidArgument() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.InvalidArgument, "invalid")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal(http.StatusBadRequest, rec.Code)
}

func (s *ErrorTestSuite) TestErrorHandler_StatusMapping_Unauthenticated() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.Unauthenticated, "unauthenticated")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal(http.StatusUnauthorized, rec.Code)
}

func (s *ErrorTestSuite) TestErrorHandler_StatusMapping_PermissionDenied() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.PermissionDenied, "denied")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal(http.StatusForbidden, rec.Code)
}

func (s *ErrorTestSuite) TestErrorHandler_StatusMapping_Internal() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.Internal, "internal error")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)
}

func (s *ErrorTestSuite) TestErrorHandler_ContentType() {
	handler := ErrorHandler(false)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.NotFound, "not found")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	s.Require().Equal("application/problem+json", rec.Header().Get("Content-Type"))
}

func (s *ErrorTestSuite) TestErrorHandler_InstanceHeader() {
	handler := ErrorHandler(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "unique-request-id-abc")
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.NotFound, "not found")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	s.Require().NoError(err)

	s.Require().Equal("unique-request-id-abc", problem.Instance)
}

func (s *ErrorTestSuite) TestErrorHandler_NoRequestID() {
	handler := ErrorHandler(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No X-Request-ID header.
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.NotFound, "not found")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	s.Require().NoError(err)

	s.Require().Empty(problem.Instance, "Instance should be empty without X-Request-ID")
}

func (s *ErrorTestSuite) TestErrorHandler_TypeURL() {
	handler := ErrorHandler(true)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	grpcErr := status.Error(codes.NotFound, "not found")

	handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

	var problem ProblemDetails
	err := json.Unmarshal(rec.Body.Bytes(), &problem)
	s.Require().NoError(err)

	s.Require().Contains(problem.Type, "grpc.io/docs/guides/status-codes")
	s.Require().Contains(problem.Type, "notfound")
}

func (s *ErrorTestSuite) TestErrorHandler_AllStatusCodes() {
	testCases := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{codes.OK, http.StatusOK},
		{codes.Canceled, 499}, // Client closed request.
		{codes.Unknown, http.StatusInternalServerError},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.FailedPrecondition, http.StatusBadRequest},
		{codes.Aborted, http.StatusConflict},
		{codes.OutOfRange, http.StatusBadRequest},
		{codes.Unimplemented, http.StatusNotImplemented},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DataLoss, http.StatusInternalServerError},
		{codes.Unauthenticated, http.StatusUnauthorized},
	}

	handler := ErrorHandler(false)

	for _, tc := range testCases {
		s.Run(tc.grpcCode.String(), func() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			grpcErr := status.Error(tc.grpcCode, "test error")

			handler(req.Context(), nil, &runtime.JSONPb{}, rec, req, grpcErr)

			s.Require().Equal(tc.httpCode, rec.Code, "gRPC %s should map to HTTP %d", tc.grpcCode, tc.httpCode)
		})
	}
}
