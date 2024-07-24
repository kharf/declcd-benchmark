package declcd

import (
	"github.com/kharf/declcd/schema/component"
)

benchmark: component.#Manifest & {
	dependencies: [
		crd.id,
		ns.id,
	]
	content: {
		apiVersion: "gitops.declcd.io/v1beta1"
		kind:       "GitOpsProject"
		metadata: {
			name:      "benchmark"
			namespace: "declcd-system"
			labels: _primaryLabels
		}
		spec: {
			branch:              "main"
			pullIntervalSeconds: 3600
			suspend:             false
			url:                 "/repository"
		}
	}
}
