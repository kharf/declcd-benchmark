package {{ .Package }}

import (
	"github.com/kharf/declcd/schema/component"
	corev1 "k8s.io/api/core/v1"
)

{{ .Namespace }}: component.#Manifest & {
	content: corev1.#Namespace & {
		apiVersion: string | *"v1"
		kind:       "Namespace"
		metadata: name: "{{ .Namespace }}"
	}
}
