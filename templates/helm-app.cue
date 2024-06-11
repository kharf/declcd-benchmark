package {{ .Package }}

import (
	"github.com/kharf/declcd/schema/component"
	"github.com/kharf/declcd-benchmark/repository/apps"
)

release: component.#HelmRelease & {
	#Name:      "{{ .HelmApp }}"
	#Namespace: "{{ .Namespace }}"
	dependencies: [
		apps.{{ .Namespace }}.id
	]
	name: #Name
	namespace: #Namespace
	chart: {
		name: "{{ .ChartName }}"
		repoURL: "{{ .RepoURL }}"
		version: "0.1.0"
	}
	values: {
		replicaCount: 0
	}
}
