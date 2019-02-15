package generator

import (
	"context"

	"github.com/docker/oscalkit/types/oscal/catalog"
	"github.com/docker/oscalkit/types/oscal/profile"
)

// ImportHandler ...
type ImportHandler interface {
	FindCatalog(profile.Import) (*catalog.Catalog, error)
	ProcessProfile(processor Processor, alterations []profile.Alter, profImport profile.Import) (*catalog.Catalog, error)
}

// NewImportHandler ...
func NewImportHandler(profile profile.Profile) ImportHandler {
	return &impHandler{
		profile:   profile,
		validator: NewValidator(),
	}
}

type impHandler struct {
	profile   profile.Profile
	validator Validator
}

func (ih *impHandler) ProcessProfile(processor Processor, alterations []profile.Alter, profImport profile.Import) (*catalog.Catalog, error) {
	// Prepare a new catalog object to merge into the final List of OutputCatalogs
	if ih.profile.Modify != nil {
		processor.ProcessAlterations(alterations)
		processor.ProcessSetParam(ih.profile.Modify.ParamSettings)
	}
	newCatalog, err := processor.GetMappedCatalogControlsFromImport(profImport)
	if err != nil {
		return nil, err
	}
	return &newCatalog, nil
}

func (ih *impHandler) FindCatalog(profileImport profile.Import) (*catalog.Catalog, error) {
	c := make(chan *catalog.Catalog)
	e := make(chan error)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// ForEach Import's Href, Fetch the Catalog JSON file
	err := ih.validator.ValidateHref(profileImport.Href)
	if err != nil {
		return nil, err
	}
	getCatalogForImport(ctx, profileImport, c, e, profileImport.Href.String(), ih.validator)
	select {
	case importedCatalog := <-c:
		return importedCatalog, nil
	case err := <-e:
		return nil, err
	}
}
