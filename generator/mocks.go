package generator

import (
	"github.com/docker/oscalkit/impl"
	"github.com/docker/oscalkit/types/oscal/catalog"
	"github.com/docker/oscalkit/types/oscal/profile"
)

type mockValidator struct{}
type mockAltHandler struct {
	pro profile.Profile
	v   Validator
}
type mockProcessor struct {
	catalog       *catalog.Catalog
	catalogHelper impl.Catalog
}

// NewMockProcessor ...
func NewMockProcessor(c *catalog.Catalog, helper impl.Catalog) Processor {
	return &mockProcessor{catalog: c, catalogHelper: helper}
}
func (m *mockProcessor) ProcessAddition(profile.Alter, []catalog.Control) []catalog.Control {
	return []catalog.Control{catalog.Control{Id: "ac-1"}}
}
func (m *mockProcessor) ProcessAlterations([]profile.Alter) *catalog.Catalog {
	return &catalog.Catalog{}
}
func (m *mockProcessor) ProcessSetParam([]profile.SetParam) *catalog.Catalog {
	return &catalog.Catalog{}
}
func (m *mockProcessor) ModifyParts(catalog.Part, []catalog.Part) []catalog.Part {
	return []catalog.Part{catalog.Part{Id: "ac-1_prm_1"}}
}
func (m *mockProcessor) GetMappedCatalogControlsFromImport(profileImport profile.Import) (catalog.Catalog, error) {
	return catalog.Catalog{}, nil
}
func (m *mockProcessor) GetCatalog() *catalog.Catalog { return &catalog.Catalog{} }
func (m *mockProcessor) Helper() impl.Catalog         { return &impl.NISTCatalog{} }

// NewMockAlterHandler ...
func NewMockAlterHandler(profile profile.Profile) AlterHandler {
	return &mockAltHandler{
		pro: profile,
		v:   NewValidator(),
	}
}

func (m *mockAltHandler) GetAlters() ([]profile.Alter, error) {
	return []profile.Alter{
		profile.Alter{
			ControlId: "ac-1",
		},
		profile.Alter{
			SubcontrolId: "ac-1.1",
		},
	}, nil
}

// Validator validates profile attributes

// NewValidator creates a new mock
func NewMockValidator() Validator {
	return &mockValidator{}
}

func (v *mockValidator) ValidateHref(href *catalog.Href) error {
	return nil
}
