package generator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/oscalkit/impl"
	"github.com/docker/oscalkit/types/oscal"
	"github.com/docker/oscalkit/types/oscal/catalog"
	"github.com/docker/oscalkit/types/oscal/profile"
	"github.com/sirupsen/logrus"
)

// CreateCatalogsFromProfile maps profile controls to multiple catalogs
func CreateCatalogsFromProfile(profileArg *profile.Profile) ([]*catalog.Catalog, error) {

	t := time.Now()
	done := 0
	errChan := make(chan error)
	catalogChan := make(chan *catalog.Catalog)
	var outputCatalogs []*catalog.Catalog
	logrus.Info("fetching alterations...")
	alterations, err := GetAlters(profileArg)
	if err != nil {
		return nil, err
	}
	logrus.Info("fetching alterations from import chain complete")

	logrus.Debug("processing alteration and parameters... \nmapping to controls...")
	// Get first import of the profile (which is a catalog)
	for _, profileImport := range profileArg.Imports {
		err := ValidateHref(profileImport.Href)
		if err != nil {
			return nil, err
		}
		go func(profileImport profile.Import) {
			catalogHelper := impl.NISTCatalog{}
			c := make(chan *catalog.Catalog)
			e := make(chan error)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			// ForEach Import's Href, Fetch the Catalog JSON file
			getCatalogForImport(ctx, profileImport, c, e, profileImport.Href.String())
			select {
			case importedCatalog := <-c:
				// Prepare a new catalog object to merge into the final List of OutputCatalogs
				if profileArg.Modify != nil {
					nc := impl.NISTCatalog{}
					importedCatalog = ProcessAlterations(alterations, importedCatalog)
					importedCatalog = ProcessSetParam(profileArg.Modify.ParamSettings, importedCatalog, &nc)
				}
				newCatalog, err := GetMappedCatalogControlsFromImport(importedCatalog, profileImport, &catalogHelper)
				if err != nil {
					errChan <- err
					return
				}
				catalogChan <- &newCatalog

			case err := <-e:
				errChan <- err
				return
			}

		}(profileImport)

	}
	for {
		select {
		case err := <-errChan:
			return nil, err
		case newCatalog := <-catalogChan:
			done++
			if newCatalog != nil {
				outputCatalogs = append(outputCatalogs, newCatalog)
			}
			if done == len(profileArg.Imports) {
				logrus.Infof("successfully mapped controls in %f seconds", time.Since(t).Seconds())
				return outputCatalogs, nil
			}
		}
	}
}
func getSubControl(call profile.Call, ctrls []catalog.Control, helper impl.Catalog) (catalog.Subcontrol, error) {
	for _, ctrl := range ctrls {
		if ctrl.Id == helper.GetControl(call.SubcontrolId) {
			for _, subctrl := range ctrl.Subcontrols {
				if subctrl.Id == call.SubcontrolId {
					return subctrl, nil
				}
			}
		}
	}
	return catalog.Subcontrol{}, fmt.Errorf("could not find subcontrol %s in catalog", call.SubcontrolId)
}

func doesCallContainSubcontrol(c profile.Call) bool {
	return c.ControlId == ""
}

// AddControlToGroup adds control to a group
func AddControlToGroup(g *catalog.Group, ctrl catalog.Control, catalogHelper impl.Catalog) {
	ctrlExists := false
	for _, x := range g.Controls {
		if x.Id == ctrl.Id {
			ctrlExists = true
			continue
		}
	}
	if !ctrlExists {
		g.Controls = append(g.Controls,
			catalog.Control{
				Id:          ctrl.Id,
				Class:       ctrl.Class,
				Title:       ctrl.Title,
				Subcontrols: []catalog.Subcontrol{},
				Params:      ctrl.Params,
				Parts:       ctrl.Parts,
			},
		)
	}
}

// AddSubControlToControls adds subcontrols to a group. If parent ctrl of subctrl doesn't exists, it adds its parent ctrl as well
func AddSubControlToControls(g *catalog.Group, ctrl catalog.Control, sc catalog.Subcontrol, catalogHelper impl.Catalog) {
	ctrlExistsInGroup := false
	for i, mappedCtrl := range g.Controls {
		if mappedCtrl.Id == strings.ToLower(catalogHelper.GetControl(sc.Id)) {
			ctrlExistsInGroup = true
			g.Controls[i].Subcontrols = append(g.Controls[i].Subcontrols, sc)
		}
	}
	if !ctrlExistsInGroup {
		g.Controls = append(g.Controls,
			catalog.Control{
				Id:          ctrl.Id,
				Class:       ctrl.Class,
				Title:       ctrl.Title,
				Params:      ctrl.Params,
				Parts:       ctrl.Parts,
				Subcontrols: []catalog.Subcontrol{sc},
			})
	}
}

// GetMappedCatalogControlsFromImport gets mapped controls in catalog per profile import
func GetMappedCatalogControlsFromImport(importedCatalog *catalog.Catalog, profileImport profile.Import, catalogHelper impl.Catalog) (catalog.Catalog, error) {

	newCatalog := catalog.CreateCatalog(importedCatalog.Title, []catalog.Group{})
	for _, group := range importedCatalog.Groups {
		newGroup := catalog.CreateGroup(group.Title, []catalog.Control{})
		for _, ctrl := range group.Controls {
			for _, call := range profileImport.Include.IdSelectors {
				if doesCallContainSubcontrol(call) {
					if strings.ToLower(ctrl.Id) == strings.ToLower(catalogHelper.GetControl(call.SubcontrolId)) {
						sc, err := getSubControl(call, group.Controls, &impl.NISTCatalog{})
						if err != nil {
							return catalog.Catalog{}, err
						}
						AddSubControlToControls(&newGroup, ctrl, sc, catalogHelper)
					}
				}
				if strings.ToLower(call.ControlId) == strings.ToLower(ctrl.Id) {
					AddControlToGroup(&newGroup, ctrl, catalogHelper)
				}
			}
		}
		if len(newGroup.Controls) > 0 {
			newCatalog.Groups = append(newCatalog.Groups, newGroup)
		}
	}
	return newCatalog, nil
}

func getCatalogForImport(ctx context.Context, i profile.Import, c chan *catalog.Catalog, e chan error, basePath string) {
	go func(i profile.Import) {
		err := ValidateHref(i.Href)
		if err != nil {
			e <- fmt.Errorf("href cannot be nil")
			return
		}
		path, err := GetFilePath(i.Href.String())
		if err != nil {
			e <- err
			return
		}
		f, err := os.Open(path)
		if err != nil {
			e <- err
			return
		}
		defer f.Close()
		o, err := oscal.New(f)
		if err != nil {
			e <- err
			return
		}
		if o.Catalog != nil {
			c <- o.Catalog
			return
		}
		newP, err := SetBasePath(o.Profile, basePath)
		if err != nil {
			e <- err
			return
		}
		o.Profile = newP
		for _, p := range o.Profile.Imports {
			go func(p profile.Import) {
				getCatalogForImport(ctx, p, c, e, basePath)
			}(p)
		}
	}(i)
}
