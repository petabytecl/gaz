package di

// NewTestContainer creates a container suitable for testing.
// It's functionally identical to New() but named for clarity in test code.
//
// Example:
//
//	func TestUserService(t *testing.T) {
//	    c := di.NewTestContainer()
//	    di.For[Database](c).Instance(&MockDatabase{})
//	    di.For[*UserService](c).Provider(NewUserService)
//	    svc := di.MustResolve[*UserService](c)
//	    // ... test assertions
//	}
func NewTestContainer() *Container {
	return New()
}
