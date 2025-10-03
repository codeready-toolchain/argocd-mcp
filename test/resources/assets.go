package resources

import (
	_ "embed"
)

//go:embed argocd-applications.json
var ApplicationsStr string

//go:embed argocd-applications-example.json
var ExampleApplicationStr string
