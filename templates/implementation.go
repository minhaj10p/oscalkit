package templates

import (
	"html/template"
)

// GetImplementationTemplate gets implementation template for implementation go struct file
func GetImplementationTemplate() (*template.Template, error) {
	return template.New("").Funcs(map[string]interface{}{"norm": Normalize}).Parse(implementationTemplate)
}

func Normalize(s string) template.HTML {
	return template.HTML(s)
}

const implementationTemplate = `
// Code generated by go implementation; DO NOT EDIT.
package {{.PackageName}}

import (
	"github.com/docker/oscalkit/types/oscal/implementation"
)

var ImplementationGenerated = implementation.Implementation{
	Capabilities: implementation.Capabilities{},
	ComponentDefinitions: []implementation.ComponentDefinition{
		{{range .Implementation.ComponentDefinitions}}
		implementation.ComponentDefinition{
			ID: ` + "`{{.ID}}`" + `,
			ComponentConfigurations: []*implementation.ComponentConfiguration{
					{{range .ComponentConfigurations}}
					{
							ID:                     ` + "`{{.ID}}`" + `,
							Name:                   ` + "`{{.Name}}`" + `,
							Description:            ` + "`{{.Description}}`" + `,
							ProvisioningMechanisms: []implementation.Mechanism{},
							ValidationMechanisms:   []implementation.Mechanism{},
							ConfigurableValues: []implementation.ConfigurableValue{
								{{range .ConfigurableValues}}
									{
											Value:   "{{.Value}}",
											ValueID: "{{.ValueID}}",
									},
								{{end}}
							},				
						},
					{{end}}
				},
				ImplementsProfiles: []*implementation.ImplementsProfile{
					{{range .ImplementsProfiles}}
						{
							ProfileID: "{{.ProfileID}}",
							ControlConfigurations: []implementation.ControlConfiguration{
								{{range .ControlConfigurations}}
									{
										ConfigurationIDRef: "{{.ConfigurationIDRef}}",
										Parameters:         []implementation.Parameter{
											{{range .Parameters}}
											{
												Guidance: []string{
													{{range .Guidance}}
														"{{.}}",
													{{end}}
												},
												ParameterID: "{{.ParameterID}}",
												ValueID: "{{.ValueID}}",
												PossibleValues: []string{
													{{range .PossibleValues}}
														"{{norm .}}",
													{{end}}
												},												
											},
											{{end}}
										},
									},
								{{end}}
							},
						},
					{{end}}
				},
				ControlImplementations: []*implementation.ControlImplementation{
					{{range .ControlImplementations}}
						{
							ID: "{{.ID}}",
							ControlIds: []implementation.ControlId{
								{{range .ControlIds}}
								{
										CatalogIDRef: "{{.CatalogIDRef}}",
										ControlID:	 ` + "`{{.ControlID}}`" + `,
										ItemID: 	 ` + "`{{.ItemID}}`" + `,
									},
								{{end}}
							},
							ControlConfigurations: []implementation.ControlConfiguration{
								{{range .ControlConfigurations}}
									{
										ConfigurationIDRef: "{{.ConfigurationIDRef}}",
										ProvisioningMechanisms: []implementation.ProvisioningMechanism{
											{{range .ProvisioningMechanisms}}
												{
													ProvisionedControls: []implementation.ControlId{
														{{range .ProvisionedControls}}
															{
																ControlID:    "{{.ControlID}}",
																CatalogIDRef: "{{.CatalogIDRef}}",
																ItemID:       "{{.ItemID}}",
															},
														{{end}}
													},
												},
							
											{{end}}
										},
										Parameters:         []implementation.Parameter{
											{{range .Parameters}}
											{
												Guidance: []string{
													{{range .Guidance}}
														"{{.}}",
													{{end}}
												},
												ParameterID: "{{.ParameterID}}",
												ValueID: "{{.ValueID}}",
												DefaultValue: "{{.DefaultValue}}",
											},
											{{end}}
										},
									},
								{{end}}
							},
						},
					{{end}}
				},
			},
		{{end}}
	},
}`
