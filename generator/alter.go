package generator

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"sync"

	"github.com/docker/oscalkit/types/oscal"
	"github.com/docker/oscalkit/types/oscal/profile"
)

// HTTPFilePath map of http resource against filepath to lessen downloads
type HTTPFilePath struct {
	sync.Mutex
	m map[string]string
}

var pathmap = HTTPFilePath{
	m: make(map[string]string),
}

type AlterHandler interface {
	GetAlters() ([]profile.Alter, error)
}

type altHandler struct {
	pro profile.Profile
	v   Validator
}

func NewAlterHandler(profile profile.Profile) AlterHandler {
	return &altHandler{
		pro: profile,
		v:   NewValidator(),
	}
}

func (a *altHandler) findAlter(p *profile.Profile, call profile.Call) (*profile.Alter, bool, error) {

	if p.Modify == nil {
		p.Modify = &profile.Modify{
			Alterations:   []profile.Alter{},
			ParamSettings: []profile.SetParam{},
		}
	}
	for _, alt := range p.Modify.Alterations {
		if equateAlter(alt, call) {
			return &alt, true, nil
		}
	}
	for _, imp := range p.Imports {
		err := a.v.ValidateHref(imp.Href)
		if err != nil {
			return nil, false, err
		}
		path := imp.Href.String()
		if isHTTPResource(imp.Href.URL) {
			pathmap.Lock()
			if v, ok := pathmap.m[imp.Href.String()]; !ok {
				path, err = GetFilePath(imp.Href.String())
				if err != nil {
					return nil, false, err
				}
				pathmap.m[imp.Href.String()] = path
			} else {
				path = v
			}
			pathmap.Unlock()
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, false, err
		}
		defer f.Close()

		o, err := oscal.New(f)
		if err != nil {
			return nil, false, err
		}
		if o.Profile == nil {
			continue
		}
		p, err = SetBasePath(o.Profile, imp.Href.String())
		if err != nil {
			return nil, false, err
		}
		o.Profile = p
		alt, found, err := a.findAlter(o.Profile, call)
		if err != nil {
			return nil, false, err
		}
		if !found {
			continue
		}
		return alt, true, nil
	}
	return nil, false, nil
}

// EquateAlter equates alter with call
func equateAlter(alt profile.Alter, call profile.Call) bool {

	if alt.ControlId == "" && alt.SubcontrolId == call.SubcontrolId {
		return true
	}
	if alt.SubcontrolId == "" && alt.ControlId == call.ControlId {
		return true
	}
	return false
}

// GetAlters gets alter attributes from import chain
func (a *altHandler) GetAlters() ([]profile.Alter, error) {

	var alterations []profile.Alter
	for _, i := range a.pro.Imports {
		if i.Include == nil {
			i.Include = &profile.Include{
				IdSelectors: []profile.Call{},
			}
		}
		for _, call := range i.Include.IdSelectors {
			found := false
			if a.pro.Modify == nil {
				a.pro.Modify = &profile.Modify{
					Alterations:   []profile.Alter{},
					ParamSettings: []profile.SetParam{},
				}
			}
			for _, alt := range a.pro.Modify.Alterations {
				if equateAlter(alt, call) {
					alterations = append(alterations, alt)
					found = true
					break
				}
			}
			if !found {
				alt, found, err := a.findAlter(&a.pro, call)
				if err != nil {
					return nil, err
				}
				if !found {
					continue
				}
				alterations = append(alterations, *alt)
			}

		}
	}
	return alterations, nil

}

func makeURL(url, child *url.URL) (*url.URL, error) {
	newURL, err := url.Parse(fmt.Sprintf("%s://%s%s/%s", url.Scheme, url.Host, path.Dir(url.Path), child.String()))
	if err != nil {
		return nil, err
	}
	return newURL, nil
}
