package declcd

import (
	"github.com/kharf/declcd/schema/component"
)

_projectName: "benchmark"

project: component.#Manifest & {
	dependencies: [crd.id]
	content: {
		apiVersion: "gitops.declcd.io/v1beta1"
		kind:       "GitOpsProject"
		metadata: {
			name:      _projectName
			namespace: "declcd-system"
		}
		spec: {
			branch:              "main"
			pullIntervalSeconds: 3600
			name:                _projectName
			suspend:             false
			url:                 "/repository"
		}
	}
}
