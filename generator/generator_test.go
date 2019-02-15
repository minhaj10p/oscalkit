package generator

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/docker/oscalkit/impl"
	"github.com/docker/oscalkit/types/oscal/catalog"

	"github.com/docker/oscalkit/types/oscal/profile"
)

func TestGetAlter(t *testing.T) {
	url, _ := url.Parse("../test_util/artifacts/FedRAMP_LOW-baseline_profile.xml")
	ctrlID := "ac-1"
	p := profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{URL: url},
				Include: &profile.Include{
					IdSelectors: []profile.Call{
						profile.Call{
							ControlId: "ac-1",
						},
						profile.Call{
							ControlId: "ac-2",
						},
						profile.Call{
							ControlId: "ac-0",
						},
					},
				},
			},
		},
		Modify: &profile.Modify{
			Alterations: []profile.Alter{profile.Alter{
				ControlId: "ac-1",
			}},
		},
	}
	altHandler := NewAlterHandler(p)
	alts, err := altHandler.GetAlters()
	if err != nil {
		t.Error(err)
		return
	}
	if len(alts) == 0 {
		t.Fail()
		return
	}
	if alts[0].ControlId != ctrlID {
		t.Fail()
		return
	}
}
func TestMakeURL(t *testing.T) {
	httpURI, _ := url.Parse("http://localhost:3000/v1/tests/profiles")
	relpath, _ := url.Parse("../users")

	expectedOutput, _ := url.Parse("http://localhost:3000/v1/users")

	out, _ := makeURL(httpURI, relpath)
	if expectedOutput.String() != out.String() {
		t.Fail()
	}
}

func TestValidateHref(t *testing.T) {

	err := NewValidator().ValidateHref(&catalog.Href{URL: &url.URL{RawPath: ":'//:://"}})
	if err != nil {
		t.Fail()
	}
}
func TestNilHref(t *testing.T) {
	if err := NewValidator().ValidateHref(nil); err == nil {
		t.Fail()
	}
}
func TestFindCatalog(t *testing.T) {
	url, _ := url.Parse("../test_util/artifacts/FedRAMP_LOW-baseline_profile.xml")
	profileImport := profile.Import{
		Href: &catalog.Href{URL: url},
	}
	handler := NewImportHandler(profile.Profile{})
	c, err := handler.FindCatalog(profileImport)
	if err != nil {
		t.Error(err)
		return
	}
	if c == nil {
		t.Fail()
		return
	}
}

func TestFailingFindCatalogWithBadHref(t *testing.T) {
	_, err := NewImportHandler(profile.Profile{}).FindCatalog(profile.Import{})
	if err == nil {
		t.Fail()
	}
}
func TestFailingFindCatalogWithInvalidHref(t *testing.T) {
	imp := profile.Import{
		Href: &catalog.Href{
			URL: &url.URL{},
		},
	}
	_, err := NewImportHandler(profile.Profile{}).FindCatalog(imp)
	if err == nil {
		t.Fail()
	}
}

func TestProcessAdditionWithSameClass(t *testing.T) {

	partID := "ac-10_prt"
	class := "guidance"
	alters := []profile.Alter{
		{
			ControlId: "ac-10",
			Additions: []profile.Add{
				profile.Add{
					Parts: []catalog.Part{
						catalog.Part{
							Id:    partID,
							Class: class,
						},
					},
				},
			},
		},
		profile.Alter{
			SubcontrolId: "ac-10.1",
			Additions: []profile.Add{
				profile.Add{
					Parts: []catalog.Part{
						catalog.Part{
							Id:    partID,
							Class: class,
						},
					},
				},
			},
		},
	}
	c := catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: []catalog.Control{
					catalog.Control{
						Id: "ac-10",
						Parts: []catalog.Part{
							catalog.Part{
								Id:    partID,
								Class: class,
							},
						},
						Subcontrols: []catalog.Subcontrol{
							catalog.Subcontrol{
								Id: "ac-10.1",
								Parts: []catalog.Part{
									catalog.Part{
										Id:    partID,
										Class: class,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	processor, err := NewProcessor(&c, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	o := processor.ProcessAlterations(alters)
	for _, g := range o.Groups {
		for _, c := range g.Controls {
			for i := range c.Parts {
				expected := fmt.Sprintf("%s_%d", partID, i+1)
				if c.Parts[i].Id != expected {
					t.Errorf("%s and %s are not identical", c.Parts[i].Id, expected)
					return
				}
			}
			for i, sc := range c.Subcontrols {
				expected := fmt.Sprintf("%s_%d", partID, i+1)
				if sc.Parts[i].Id != expected {
					t.Errorf("%s and %s are not identical", sc.Parts[i].Id, expected)
					return
				}
			}
		}
	}
}

func TestProcessAdditionWithDifferentPartClass(t *testing.T) {

	ctrlID := "ac-10"
	subctrlID := "ac-10.1"
	partID := "ac-10_stmt.a"

	alters := []profile.Alter{
		profile.Alter{
			ControlId: ctrlID,
			Additions: []profile.Add{
				profile.Add{
					Parts: []catalog.Part{
						catalog.Part{
							Id:    partID,
							Class: "c1",
						},
					},
				},
			},
		},
		profile.Alter{
			SubcontrolId: subctrlID,
			Additions: []profile.Add{
				profile.Add{
					Parts: []catalog.Part{
						catalog.Part{
							Id:    partID,
							Class: "c2",
						},
					},
				},
			},
		},
	}
	c := catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: []catalog.Control{
					catalog.Control{
						Id: ctrlID,
						Parts: []catalog.Part{
							catalog.Part{
								Id:    partID,
								Class: "c3",
							},
						},
						Subcontrols: []catalog.Subcontrol{
							catalog.Subcontrol{
								Id: subctrlID,
								Parts: []catalog.Part{
									catalog.Part{
										Id:    partID,
										Class: "c4",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	processor, err := NewProcessor(&c, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	o := processor.ProcessAlterations(alters)
	if len(o.Groups[0].Controls[0].Parts) != 2 {
		t.Error("parts for controls not getting added properly")
	}
	if len(o.Groups[0].Controls[0].Subcontrols[0].Parts) != 2 {
		t.Error("parts for sub-controls not getting added properly")
	}

}

func TestProcessSetParam(t *testing.T) {
	parameterID := "ac-1_prm_1"
	parameterVal := "777"
	ctrl := "ac-1"
	shouldChange := fmt.Sprintf(`this should change. <insert param-id="%s">`, parameterID)
	afterChange := fmt.Sprintf(`this should change. %s`, parameterVal)
	sp := []profile.SetParam{
		profile.SetParam{
			Id: parameterID,
			Constraints: []catalog.Constraint{
				catalog.Constraint{
					Value: parameterVal,
				},
			},
		},
		profile.SetParam{
			Id:          parameterID,
			Constraints: []catalog.Constraint{},
		},
	}
	controls := []catalog.Control{
		catalog.Control{
			Id: ctrl,
			Parts: []catalog.Part{
				catalog.Part{
					Prose: &catalog.Prose{
						P: []catalog.P{
							catalog.P{
								Raw: shouldChange,
							},
						},
					},
				},
			},
		},
	}
	ctlg := &catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: controls,
			},
		},
	}
	processor, err := NewProcessor(ctlg, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	ctlg = processor.ProcessSetParam(sp)
	if ctlg.Groups[0].Controls[0].Parts[0].Prose.P[0].Raw != afterChange {
		t.Error("failed to parse set param template")
	}
}

func TestProcessSetParamWithUnmatchParam(t *testing.T) {
	parameterID := "ac-1_prm_1"
	parameterVal := "777"
	ctrl := "ac-1"
	shouldChange := fmt.Sprintf(`this should change. <insert param-id="%s">`, parameterID)
	afterChange := fmt.Sprintf(`this should change. %s`, parameterVal)
	sp := []profile.SetParam{
		profile.SetParam{
			Id: "ac-1_prm_2",
			Constraints: []catalog.Constraint{
				catalog.Constraint{
					Value: parameterVal,
				},
			},
		},
	}
	controls := []catalog.Control{
		catalog.Control{
			Id: ctrl,
			Parts: []catalog.Part{
				catalog.Part{
					Prose: &catalog.Prose{
						P: []catalog.P{
							catalog.P{
								Raw: shouldChange,
							},
						},
					},
				},
			},
		},
	}
	ctlg := &catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: controls,
			},
		},
	}
	processor, err := NewProcessor(ctlg, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	ctlg = processor.ProcessSetParam(sp)
	if ctlg.Groups[0].Controls[0].Parts[0].Prose.P[0].Raw == afterChange {
		t.Error("should not change parameter with mismatching parameter id")
	}
}

func TestInvalidGetMappedCatalogControlsFromImport(t *testing.T) {
	importedCatalog := catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: []catalog.Control{
					catalog.Control{
						Id: "ac-2",
					},
				},
			},
		},
	}
	profileImport := profile.Import{
		Include: &profile.Include{
			IdSelectors: []profile.Call{
				profile.Call{
					SubcontrolId: "ac-2.1",
				},
			},
		},
	}
	processor, err := NewProcessor(&importedCatalog, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	_, err = processor.GetMappedCatalogControlsFromImport(profileImport)
	if err == nil {
		t.Fail()
	}
}

func TestGetMappedCatalogControlsFromImport(t *testing.T) {
	importedCatalog := catalog.Catalog{
		Groups: []catalog.Group{
			catalog.Group{
				Controls: []catalog.Control{
					catalog.Control{
						Id: "ac-2",
						Subcontrols: []catalog.Subcontrol{
							catalog.Subcontrol{
								Id: "ac-2.1",
							},
						},
					},
				},
			},
		},
	}
	profileImport := profile.Import{
		Include: &profile.Include{
			IdSelectors: []profile.Call{
				profile.Call{
					SubcontrolId: "ac-2.1",
				},
			},
		},
	}
	processor, err := NewProcessor(&importedCatalog, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	cat, err := processor.GetMappedCatalogControlsFromImport(profileImport)
	if err != nil {
		t.Fail()
		return
	}

	if cat.Groups[0].Controls[0].Id != "ac-2" {
		t.Fail()
		return
	}
	if cat.Groups[0].Controls[0].Subcontrols[0].Id != "ac-2.1" {
		t.Fail()
		return
	}
}
func TestProcessProfile(t *testing.T) {
	p := profile.Profile{
		Modify: &profile.Modify{
			Alterations: []profile.Alter{
				profile.Alter{ControlId: "ac-1"},
			},
		},
	}
	c := catalog.Catalog{}
	imp := profile.Import{}
	importHandler := NewImportHandler(p)
	processor := NewMockProcessor(&c, &impl.NISTCatalog{})
	alters := []profile.Alter{}
	cat, err := importHandler.ProcessProfile(processor, alters, imp)
	if err != nil {
		t.Error(err)
		return
	}
	if cat == nil {
		t.Fail()
		return
	}
}
func TestAddSubControlToControls(t *testing.T) {
	controlDetails := catalog.Control{Id: "ac-2"}
	subCtrlToAdd := catalog.Subcontrol{Id: "ac-2.1"}
	g := catalog.Group{
		Controls: []catalog.Control{catalog.Control{Id: "ac-1"}},
	}
	AddSubControlToControls(&g, controlDetails, subCtrlToAdd, &impl.NISTCatalog{})
	ctrlFound := false
	for _, x := range g.Controls {
		if x.Id == "ac-2" {
			ctrlFound = true
			subCtrlFound := false
			for _, y := range x.Subcontrols {
				if y.Id == "ac-2.1" {
					subCtrlFound = true
				}
			}
			if !subCtrlFound {
				t.Fail()
			}
			break
		}
	}
	if !ctrlFound {
		t.Fail()
	}
}

func TestAddExistingControlToGroup(t *testing.T) {
	ctrlToAdd := catalog.Control{Id: "ac-1"}
	g := catalog.Group{
		Controls: []catalog.Control{
			catalog.Control{
				Id: "ac-1",
			},
		},
	}
	AddControlToGroup(&g, ctrlToAdd, &impl.NISTCatalog{})
	if len(g.Controls) > 1 {
		t.Fail()
	}
}

func TestInvalidGetSubControl(t *testing.T) {
	c := profile.Call{SubcontrolId: "ac-2.1"}
	controls := []catalog.Control{catalog.Control{Id: "ac-1"}}
	_, err := getSubControl(c, controls, &impl.NISTCatalog{})
	if err == nil {
		t.Fail()
	}
}

func TestGetSubControl(t *testing.T) {
	c := profile.Call{SubcontrolId: "ac-2.1"}
	controls := []catalog.Control{catalog.Control{Id: "ac-2", Subcontrols: []catalog.Subcontrol{
		catalog.Subcontrol{Id: "ac-2.1"},
	}}}
	sc, err := getSubControl(c, controls, &impl.NISTCatalog{})
	if err != nil {
		t.Error(err)
		return
	}
	if sc.Id != "ac-2.1" {
		t.Fail()
		return
	}
}

func TestGetCatalogInvalidFilePath(t *testing.T) {

	url := "http://[::1]a"
	_, err := GetFilePath(url)
	if err == nil {
		t.Error("should fail")
	}
}

func failTest(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
	}
}

func TestSetBasePathWithRelPath(t *testing.T) {
	relativePath := "./something.xml"
	absoulePath, _ := filepath.Abs(relativePath)
	p := &profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{
					URL: func() *url.URL {
						x, _ := url.Parse(relativePath)
						return x
					}(),
				},
			},
		},
	}
	p, err := SetBasePath(p, "")
	if err != nil {
		t.Error(err)
	}
	if p.Imports[0].Href.String() != absoulePath {
		t.Fail()
	}
}

func TestSetBasePathWithHttpPath(t *testing.T) {
	httpPath := "http://localhost:3000/v1/test/profiles/p1.xml"
	relativePath := "./p2.xml"
	outputPath := "http://localhost:3000/v1/test/profiles/p2.xml"
	p := &profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{
					URL: func() *url.URL {
						x, _ := url.Parse(relativePath)
						return x
					}(),
				},
			},
			profile.Import{
				Href: &catalog.Href{
					URL: func() *url.URL {
						x, _ := url.Parse(httpPath)
						return x
					}(),
				},
			},
		},
	}
	p, err := SetBasePath(p, httpPath)
	if err != nil {
		t.Error(err)
	}
	if p.Imports[0].Href.String() != outputPath {
		t.Fail()
	}
	if p.Imports[1].Href.String() != httpPath {
		t.Fail()
	}
}

func TestGetAltersWithAltersPresent(t *testing.T) {

	alterTitle := catalog.Title("test-title")
	ctrlID := "ctrl-1"
	p := &profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{
					URL: func() *url.URL {
						u, _ := url.Parse("p1.xml")
						return u
					}(),
				},
				Include: &profile.Include{
					IdSelectors: []profile.Call{
						profile.Call{
							ControlId: ctrlID,
						},
					},
				},
			},
		},
		Modify: &profile.Modify{
			Alterations: []profile.Alter{
				profile.Alter{
					ControlId: ctrlID,
					Additions: []profile.Add{
						profile.Add{
							Title: alterTitle,
						},
					},
				},
			},
		},
	}

	altHandler := NewAlterHandler(*p)
	alters, err := altHandler.GetAlters()
	if err != nil {
		t.Error(err)
	}
	if len(alters) == 0 {
		t.Error("no alters found")
	}
	if alters[0].ControlId != ctrlID {
		t.Fail()
	}
}

func TestSubControlsMapping(t *testing.T) {
	profile := profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{
					URL: func() *url.URL {
						url, _ := url.Parse("https://raw.githubusercontent.com/usnistgov/OSCAL/master/content/nist.gov/SP800-53/rev4/NIST_SP-800-53_rev4_catalog.xml")
						return url
					}(),
				},
				Include: &profile.Include{
					IdSelectors: []profile.Call{
						profile.Call{
							ControlId: "ac-1",
						},
						profile.Call{
							ControlId: "ac-2",
						},
						profile.Call{
							SubcontrolId: "ac-2.1",
						},
						profile.Call{
							SubcontrolId: "ac-2.2",
						},
					},
				},
			},
		},
		Modify: &profile.Modify{
			Alterations: []profile.Alter{
				profile.Alter{
					ControlId: "ac-1",
					Additions: []profile.Add{profile.Add{
						Parts: []catalog.Part{
							catalog.Part{
								Id: "ac-1_obj",
							},
						},
					}},
				},
				profile.Alter{
					ControlId: "ac-2",
					Additions: []profile.Add{profile.Add{
						Parts: []catalog.Part{
							catalog.Part{
								Id: "ac-2_obj",
							},
						},
					}},
				},
				profile.Alter{
					SubcontrolId: "ac-2.1",
					Additions: []profile.Add{profile.Add{
						Parts: []catalog.Part{
							catalog.Part{
								Id: "ac-2.1_obj",
							},
						},
					}},
				},
				profile.Alter{
					SubcontrolId: "ac-2.2",
					Additions: []profile.Add{profile.Add{
						Parts: []catalog.Part{
							catalog.Part{
								Id: "ac-2.2_obj",
							},
						},
					}},
				},
			},
		},
	}

	v := NewMockValidator()
	pf := NewProcessorFactory()
	altHandler := NewMockAlterHandler(profile)
	c, err := CreateCatalogsFromProfile(&profile, v, NewImportHandler(profile), pf, altHandler)
	if err != nil {
		t.Error("error should be nil")
	}
	if c[0].Groups[0].Controls[1].Subcontrols[0].Id != "ac-2.1" {
		t.Errorf("does not contain ac-2.1 in subcontrols")
	}

}

func TestIsHttp(t *testing.T) {

	httpRoute := "http://localhost:3000"
	expectedOutputForHTTP := true

	nonHTTPRoute := "NIST.GOV.JSON"
	expectedOutputForNonHTTP := false

	r, err := url.Parse(httpRoute)
	if err != nil {
		t.Error(err)
	}
	if isHTTPResource(r) != expectedOutputForHTTP {
		t.Error("Invalid output for http routes")
	}

	r, err = url.Parse(nonHTTPRoute)
	if err != nil {
		t.Error(err)
	}
	if isHTTPResource(r) != expectedOutputForNonHTTP {
		t.Error("Invalid output for non http routes")
	}

}

func TestReadCatalog(t *testing.T) {

	catalogTitle := "NIST SP800-53"
	r := bytes.NewReader([]byte(string(
		fmt.Sprintf(`
		{
			"catalog": {
				"title": "%s",
				"declarations": {
					"href": "NIST_SP-800-53_rev4_declarations.xml"
				},
				"groups": [
					{
						"controls": [
							{
								"id": "at-1",
								"class": "SP800-53",
								"title": "Security Awareness and Training Policy and Procedures",
								"params": [
									{
										"id": "at-1_prm_1",
										"label": "organization-defined personnel or roles"
									},
									{
										"id": "at-1_prm_2",
										"label": "organization-defined frequency"
									},
									{
										"id": "at-1_prm_3",
										"label": "organization-defined frequency"
									}
								]
							}
						]
					}
				]
			}
		}`, catalogTitle))))

	c, err := ReadCatalog(r)
	if err != nil {
		t.Error(err)
	}

	if c.Title != catalog.Title(catalogTitle) {
		t.Error("title not equal")
	}

}

func TestReadInvalidCatalog(t *testing.T) {

	r := bytes.NewReader([]byte(string(`{ "catalog": "some dummy bad json"}`)))
	_, err := ReadCatalog(r)
	if err == nil {
		t.Error("successfully parsed invalid catalog file")
	}
}

func TestCreateCatalogsFromProfile(t *testing.T) {

	href, _ := url.Parse("https://raw.githubusercontent.com/usnistgov/OSCAL/master/content/nist.gov/SP800-53/rev4/NIST_SP-800-53_rev4_catalog.xml")
	p := profile.Profile{
		Imports: []profile.Import{
			profile.Import{
				Href: &catalog.Href{
					URL: href,
				},
				Include: &profile.Include{
					IdSelectors: []profile.Call{
						profile.Call{
							ControlId: "ac-1",
						},
					},
				},
			},
		},
		Modify: &profile.Modify{
			Alterations: []profile.Alter{
				profile.Alter{
					ControlId: "ac-1",
					Additions: []profile.Add{profile.Add{
						Parts: []catalog.Part{
							catalog.Part{
								Id: "ac-1_obj",
							},
						},
					}},
				},
			},
		},
	}
	v := NewMockValidator()
	pf := NewProcessorFactory()
	altHandler := NewMockAlterHandler(p)
	x, err := CreateCatalogsFromProfile(&p, v, NewImportHandler(p), pf, altHandler)
	if err != nil {
		t.Errorf("error should be null")
	}
	if len(x) != 1 {
		t.Error("there must be one catalog")
	}
	if x[0].Groups[0].Controls[0].Id != "ac-1" {
		t.Error("Invalid control Id")
	}

}
