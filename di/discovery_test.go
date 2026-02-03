package di

import (
	"reflect"
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

func (s *DiscoverySuite) TestResolveAllByType() {
	c := New()
	For[*discImplA](c).ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })
	For[*discImplB](c).ProviderFunc(func(_ *Container) *discImplB { return &discImplB{} })

	// Resolve all Service interface
	typ := reflect.TypeOf((*discService)(nil)).Elem()
	results, err := c.ResolveAllByType(typ)
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

	results, err := c.ResolveAllByName(TypeName[*discImplA]())
	s.Require().NoError(err)
	s.Len(results, 2)
}

func (s *DiscoverySuite) TestResolveGroup() {
	c := New()
	For[*discImplA](c).InGroup("mygroup").ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} })
	For[*discImplB](c).InGroup("mygroup").ProviderFunc(func(_ *Container) *discImplB { return &discImplB{} })
	For[*discImplA](c).Named("other").ProviderFunc(func(_ *Container) *discImplA { return &discImplA{} }) // Not in group

	results, err := c.ResolveGroup("mygroup")
	s.Require().NoError(err)
	s.Len(results, 2)
}

func (s *DiscoverySuite) TestResolveGroup_Empty() {
	c := New()
	results, err := c.ResolveGroup("nonexistent")
	s.Require().NoError(err)
	s.Empty(results)
}
