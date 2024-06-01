package {{ .Package }}

import (
	"github.com/kharf/declcd/schema/component"
	"github.com/kharf/declcd-benchmark/repository/apps"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
)

service: component.#Manifest & {
	#Name:      "{{ .App }}"
	#Namespace: "{{ .Namespace }}"
	dependencies: [
		apps.{{ .Namespace }}.id
	]
	content: corev1.#Service & {
		apiVersion: string | *"v1"
		kind:       "Service"
		metadata: {
			name:      #Name
			namespace: #Namespace
			labels: app: #Name
		}
		spec: {
			ports: [{
				port:       8080
				name:       "high"
				protocol:   "TCP"
				targetPort: 8080
			}]
			selector: app: #Name
		}
	}
}

deployment: component.#Manifest & {
	#Name:      "{{ .App }}"
	#Namespace: "{{ .Namespace }}"
	dependencies: [
		service.id
	]
	content: appsv1.#Deployment & {
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: {
			name:      #Name
			namespace: #Namespace
			labels: app: #Name
		}
		spec: {
			replicas: 1
			selector: matchLabels: app: #Name
			template: {
				metadata: labels: app: #Name
				spec: containers: [{
					image: "gcr.io/kubernetes-e2e-test-images/echoserver:2.2"
					name:  #Name
					ports: [{
						containerPort: 8080
					}]
				}]
			}
		}
	}
}
