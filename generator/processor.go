package generator

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/oscalkit/impl"
	"github.com/docker/oscalkit/types/oscal/catalog"
	"github.com/docker/oscalkit/types/oscal/profile"
)

// Processor ...
type Processor interface {
	ProcessAddition(profile.Alter, []catalog.Control) []catalog.Control
	ProcessAlterations([]profile.Alter) *catalog.Catalog
	ProcessSetParam([]profile.SetParam) *catalog.Catalog
	ModifyParts(catalog.Part, []catalog.Part) []catalog.Part
	GetMappedCatalogControlsFromImport(profileImport profile.Import) (catalog.Catalog, error)
	GetCatalog() *catalog.Catalog
	Helper() impl.Catalog
}

type processor struct {
	catalog       *catalog.Catalog
	catalogHelper impl.Catalog
}

// ProcessorFactory ...
type ProcessorFactory interface {
	NewProcessor(c *catalog.Catalog, catalogHelper impl.Catalog) Processor
}

// NewProcessorFactory ..
func NewProcessorFactory() ProcessorFactory {
	return &pf{}
}

type pf struct{}

func (p *pf) NewProcessor(c *catalog.Catalog, helper impl.Catalog) Processor {
	return &processor{catalog: c, catalogHelper: helper}
}

// NewProcessor creates a new profile processor against a catalog
func NewProcessor(c *catalog.Catalog, catalogHelper impl.Catalog) (Processor, error) {
	if c == nil || catalogHelper == nil {
		return nil, fmt.Errorf("incomplete args to create a profile processor")
	}
	catalog := processor{catalog: c, catalogHelper: catalogHelper}
	return &catalog, nil
}

func (pro *processor) Helper() impl.Catalog {
	return pro.catalogHelper
}
func (pro *processor) GetCatalog() *catalog.Catalog {
	return pro.catalog
}

// ProcessAddition processes additions of a profile
func (pro *processor) ProcessAddition(alt profile.Alter, controls []catalog.Control) []catalog.Control {
	for j, ctrl := range controls {
		if ctrl.Id == alt.ControlId {
			for _, add := range alt.Additions {
				for _, p := range add.Parts {
					appended := false
					for _, catalogPart := range ctrl.Parts {
						if p.Class == catalogPart.Class {
							appended = true
							// append with all the parts with matching classes
							parts := pro.ModifyParts(p, ctrl.Parts)
							ctrl.Parts = parts
						}
					}
					if !appended {
						ctrl.Parts = append(ctrl.Parts, p)
					}
				}
			}
			controls[j] = ctrl
		}
		for k, subctrl := range controls[j].Subcontrols {
			if subctrl.Id == alt.SubcontrolId {
				for _, add := range alt.Additions {
					for _, p := range add.Parts {
						appended := false
						for _, catalogPart := range subctrl.Parts {
							if p.Class == catalogPart.Class {
								appended = true
								// append with all the parts
								parts := pro.ModifyParts(p, subctrl.Parts)
								subctrl.Parts = parts
							}
						}
						if !appended {
							subctrl.Parts = append(subctrl.Parts, p)
						}
					}

				}
			}
			controls[j].Subcontrols[k] = subctrl
		}
	}
	return controls
}

// ProcessAlterations processes alteration section of a profile
func (pro *processor) ProcessAlterations(alterations []profile.Alter) *catalog.Catalog {
	for _, alt := range alterations {
		for i, g := range pro.catalog.Groups {
			pro.catalog.Groups[i].Controls = pro.ProcessAddition(alt, g.Controls)
		}
	}
	return pro.catalog
}

// ProcessSetParam processes set-param of a profile
func (pro *processor) ProcessSetParam(setParams []profile.SetParam) *catalog.Catalog {
	for _, sp := range setParams {
		ctrlID := pro.catalogHelper.GetControl(sp.Id)
		for i, g := range pro.catalog.Groups {
			for j, catalogCtrl := range g.Controls {
				if ctrlID == catalogCtrl.Id {
					for k := range catalogCtrl.Parts {
						if len(sp.Constraints) == 0 {
							continue
						}
						pro.catalog.Groups[i].Controls[j].Parts[k].ModifyProse(sp.Id, sp.Constraints[0].Value)
					}
				}
			}
		}
	}
	return pro.catalog
}

// ModifyParts modifies parts
func (pro *processor) ModifyParts(p catalog.Part, controlParts []catalog.Part) []catalog.Part {

	// append with all the parts
	var parts []catalog.Part
	for i, part := range controlParts {
		if p.Class != part.Class {
			parts = append(parts, part)
			continue
		}
		id := part.Id
		part.Id = fmt.Sprintf("%s_%d", id, i+1)
		parts = append(parts, part)
		part.Id = fmt.Sprintf("%s_%d", id, i+2)
		parts = append(parts, part)
	}
	return parts
}

// SetBasePath sets up base paths for profiles
func SetBasePath(p *profile.Profile, parentPath string) (*profile.Profile, error) {
	for i, x := range p.Imports {
		err := NewValidator().ValidateHref(x.Href)
		if err != nil {
			return nil, err
		}
		parentURL, err := url.Parse(parentPath)
		if err != nil {
			return nil, err
		}
		// If the import href is http. Do nothing as it doesn't depend on the parent path
		if isHTTPResource(x.Href.URL) {
			continue
		}
		//if parent is HTTP, and imports are relative, modify imports to http
		if !isHTTPResource(x.Href.URL) && isHTTPResource(parentURL) {
			url, err := makeURL(parentURL, x.Href.URL)
			if err != nil {
				return nil, err
			}
			p.Imports[i].Href = &catalog.Href{URL: url}
			continue
		}
		path := fmt.Sprintf("%s/%s", path.Dir(parentPath), path.Base(x.Href.String()))
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		uri, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		p.Imports[i].Href = &catalog.Href{URL: uri}
	}
	return p, nil
}

// GetMappedCatalogControlsFromImport gets mapped controls in catalog per profile import
func (pro *processor) GetMappedCatalogControlsFromImport(profileImport profile.Import) (catalog.Catalog, error) {

	newCatalog := catalog.CreateCatalog(pro.catalog.Title, []catalog.Group{})
	for _, group := range pro.catalog.Groups {
		newGroup := catalog.CreateGroup(group.Title, []catalog.Control{})
		for _, ctrl := range group.Controls {
			for _, call := range profileImport.Include.IdSelectors {
				if doesCallContainSubcontrol(call) {
					if strings.ToLower(ctrl.Id) == strings.ToLower(pro.catalogHelper.GetControl(call.SubcontrolId)) {
						sc, err := getSubControl(call, group.Controls, &impl.NISTCatalog{})
						if err != nil {
							return catalog.Catalog{}, err
						}
						AddSubControlToControls(&newGroup, ctrl, sc, pro.catalogHelper)
					}
				}
				if strings.ToLower(call.ControlId) == strings.ToLower(ctrl.Id) {
					AddControlToGroup(&newGroup, ctrl, pro.catalogHelper)
				}
			}
		}
		if len(newGroup.Controls) > 0 {
			newCatalog.Groups = append(newCatalog.Groups, newGroup)
		}
	}
	return newCatalog, nil
}
