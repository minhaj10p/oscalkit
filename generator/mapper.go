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

type Mapper interface{}
type mapper struct{}

// CreateCatalogsFromProfile maps profile controls to multiple catalogs
func CreateCatalogsFromProfile(profileArg *profile.Profile, v Validator, handler ImportHandler, pf ProcessorFactory, altHandler AlterHandler) ([]*catalog.Catalog, error) {

	t := time.Now()
	var outputCatalogs []*catalog.Catalog
	logrus.Info("fetching alterations...")
	alterations, err := altHandler.GetAlters()
	if err != nil {
		return nil, err
	}
	logrus.Info("fetching alterations from import chain complete")
	logrus.Debug("processing alteration and parameters... \nmapping to controls...")
	// Get first import of the profile (which is a catalog)
	for _, profileImport := range profileArg.Imports {
		err := v.ValidateHref(profileImport.Href)
		if err != nil {
			return nil, err
		}
		cat, err := handler.FindCatalog(profileImport)
		if err != nil {
			return nil, err
		}
		processor := pf.NewProcessor(cat, &impl.NISTCatalog{})
		outputCatalog, err := handler.ProcessProfile(processor, alterations, profileImport)
		if err != nil {
			return nil, err
		}
		outputCatalogs = append(outputCatalogs, outputCatalog)
	}
	logrus.Infof("successfully mapped controls in %f seconds", time.Since(t).Seconds())
	return outputCatalogs, nil
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

func getCatalogForImport(ctx context.Context, i profile.Import, c chan *catalog.Catalog, e chan error, basePath string, v Validator) {
	go func(i profile.Import) {
		err := v.ValidateHref(i.Href)
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
				getCatalogForImport(ctx, p, c, e, basePath, v)
			}(p)
		}
	}(i)
}
