package generator

import (
	"fmt"
	"net/url"

	"github.com/docker/oscalkit/types/oscal/catalog"
)

// Validator validates profile attributes
type Validator interface {
	ValidateHref(*catalog.Href) error
}

// NewValidator ...
func NewValidator() Validator {
	return &validator{}
}

type validator struct{}

func (v *validator) ValidateHref(href *catalog.Href) error {
	if href == nil {
		return fmt.Errorf("Href cannot be empty")
	}
	_, err := url.Parse(href.String())
	if err != nil {
		return err
	}
	return nil
}
