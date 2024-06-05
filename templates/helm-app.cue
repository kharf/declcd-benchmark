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
		repoURL: "oci://twuni-docker-registry.default.svc:5000/charts"
		version: "0.1.0"
	}
	values: {
		replicaCount: 0
	}
}
