// Package gaztest provides test utilities for gaz applications.
// It enables easy testing with automatic cleanup, mock injection,
// and assertion methods that fail tests on error.
//
// Basic usage:
//
//	func TestMyService(t *testing.T) {
//	    app, err := gaztest.New(t).Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    defer app.RequireStop()
//
//	    // ... test logic
//	}
//
// With mock replacement:
//
//	func TestWithMock(t *testing.T) {
//	    // First create an app with registered services
//	    baseApp := gaz.New()
//	    gaz.For[Database](baseApp.Container()).Instance(&RealDatabase{})
//	    baseApp.Build()
//
//	    // Then create test app with mock replacement
//	    mock := &MockDatabase{}
//	    app, err := gaztest.New(t).
//	        WithApp(baseApp).
//	        Replace(mock).
//	        Build()
//	    require.NoError(t, err)
//
//	    app.RequireStart()
//	    // ... test logic using mock
//	}
//
// Custom timeout:
//
//	func TestWithTimeout(t *testing.T) {
//	    app, err := gaztest.New(t).
//	        WithTimeout(10 * time.Second).
//	        Build()
//	    require.NoError(t, err)
//	    // ...
//	}
package gaztest
