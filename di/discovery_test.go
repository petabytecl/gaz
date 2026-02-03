package di

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DiscoverySuite struct {
	suite.Suite
}

func TestDiscoverySuite(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}

type discService interface {
	GetValue() string
}

type discImplA struct{}

func (i *discImplA) GetValue() string { return "A" }

type discImplB struct{}

func (i *discImplB) GetValue() string { return "B" }

func (s *DiscoverySuite) TestResolveAll() {
	c := New()
	For[*discImplA](c).ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })
	For[*discImplB](c).ProviderFunc(func(_ *Container) *discImplB { return &discImplB{} })

	// Resolve all Service interface using generics
	results, err := ResolveAll[discService](c)
	s.Require().NoError(err)
	s.Len(results, 2)

	// Check types
	var seenA, seenB bool
	for _, res := range results {
		if _, ok := res.(*discImplA); ok {
			seenA = true
		}
		if _, ok := res.(*discImplB); ok {
			seenB = true
		}
	}
	s.True(seenA, "Should see implA")
	s.True(seenB, "Should see implB")
}

func (s *DiscoverySuite) TestResolveAllByName_MultipleProviders() {
	c := New()
	For[*discImplA](c).ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })
	For[*discImplA](c).ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })

	// ResolveAllByName is not exposed via generics yet, so we test the container method
	results, err := c.ResolveAllByName(TypeName[*discImplA]())
	s.Require().NoError(err)
	s.Len(results, 2)
}

func (s *DiscoverySuite) TestResolveGroup() {
	c := New()
	For[*discImplA](c).InGroup("mygroup").ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })
	For[*discImplB](c).InGroup("mygroup").ProviderFunc(func(_ *Container) *discImplB { return &discImplB{} })
	For[*discImplA](c).Named("other").ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} }) // Not in group

	// Use generic ResolveGroup
	results, err := ResolveGroup[discService](c, "mygroup")
	s.Require().NoError(err)
	s.Len(results, 2)
}

func (s *DiscoverySuite) TestResolveGroup_Empty() {
	c := New()
	results, err := ResolveGroup[discService](c, "nonexistent")
	s.Require().NoError(err)
	s.Empty(results)
}
