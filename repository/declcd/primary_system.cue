package declcd

import (
	"github.com/kharf/declcd/schema/component"
)

_primaryLabels: {
	"\(_controlPlaneKey)": "project-controller-primary"
	"\(_shardKey)":        "primary"
}

primaryServiceAccount: component.#Manifest & {
	dependencies: [ns.id]
	content: {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata: {
			name:      "project-controller-primary"
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
	}
}

primaryClusteRoleBinding: component.#Manifest & {
	dependencies: [
		ns.id,
		clusterRole.id,
		primaryServiceAccount.id,
	]
	content: {
		apiVersion: "rbac.authorization.k8s.io/v1"
		kind:       "ClusterRoleBinding"
		metadata: {
			name:   "project-controller-primary"
			labels: _primaryLabels
		}
		roleRef: {
			apiGroup: "rbac.authorization.k8s.io"
			kind:     clusterRole.content.kind
			name:     clusterRole.content.metadata.name
		}
		subjects: [
			{
				kind:      primaryServiceAccount.content.kind
				name:      primaryServiceAccount.content.metadata.name
				namespace: primaryServiceAccount.content.metadata.namespace
			},
		]
	}
}

_primaryLeaderRoleName: "primary-leader-election"
primaryLeaderRole: component.#Manifest & {
	dependencies: [ns.id]
	content: {
		apiVersion: "rbac.authorization.k8s.io/v1"
		kind:       "Role"
		metadata: {
			name:      _primaryLeaderRoleName
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
		rules: [
			{
				apiGroups: ["coordination.k8s.io"]
				resources: ["leases"]
				verbs: [
					"get",
					"create",
					"update",
				]
			},
			{
				apiGroups: [""]
				resources: ["events"]
				verbs: [
					"create",
					"patch",
				]
			},
		]
	}
}

primaryLeaderRoleBinding: component.#Manifest & {
	dependencies: [
		ns.id,
		primaryLeaderRole.id,
	]
	content: {
		apiVersion: "rbac.authorization.k8s.io/v1"
		kind:       "RoleBinding"
		metadata: {
			name:      _primaryLeaderRoleName
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
		roleRef: {
			apiGroup: "rbac.authorization.k8s.io"
			kind:     primaryLeaderRole.content.kind
			name:     primaryLeaderRole.content.metadata.name
		}
		subjects: [
			{
				kind:      primaryServiceAccount.content.kind
				name:      primaryServiceAccount.content.metadata.name
				namespace: primaryServiceAccount.content.metadata.namespace
			},
		]
	}
}

primaryPVC: component.#Manifest & {
	dependencies: [
		ns.id,
		knownHostsCm.id,
	]
	content: {
		apiVersion: "v1"
		kind:       "PersistentVolumeClaim"
		metadata: {
			name:      "primary"
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
		spec: {
			accessModes: [
				"ReadWriteOnce",
			]
			resources: {
				requests: {
					storage: "200Mi"
				}
			}
		}
	}
}

primaryProjectControllerDeployment: component.#Manifest & {
	dependencies: [
		ns.id,
		primaryPVC.id,
		knownHostsCm.id,
	]
	content: {
		apiVersion: "apps/v1"
		kind:       "Deployment"
		metadata: {
			name:      "project-controller-primary"
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
		spec: {
			selector: matchLabels: _primaryLabels
			replicas: 1
			template: {
				metadata: {
					labels: _primaryLabels
				}
				spec: {
					serviceAccountName: "project-controller-primary"
					securityContext: {
						runAsUser: 0
					}
					volumes: [
						{
							name: "primary"
							persistentVolumeClaim: claimName: "primary"
						},
						{
							name: "podinfo"
							downwardAPI: {
								items: [
									{
										path: "namespace"
										fieldRef: fieldPath: "metadata.namespace"
									},
									{
										path: "name"
										fieldRef: fieldPath: "metadata.labels['\(_controlPlaneKey)']"
									},
									{
										path: "shard"
										fieldRef: fieldPath: "metadata.labels['\(_shardKey)']"
									},
								]
							}
						},
						{
							name: "repository"
							hostPath: {
								path: "/repository"
								type: "Directory"
							}
						},
						{
							name: "ssh"
							configMap: name: knownHostsCm.content.metadata.name
						},
						{
							name: "cache"
							emptyDir: {}
						},
					]
					containers: [
						{
							name:  "project-controller-primary"
							image: "ghcr.io/kharf/declcd:0.24.9"
							command: [
								"/controller",
							]
							args: [
								"--log-level=0",
								"--plain-http",
								"--insecure-skip-tls-verify",
							]
							securityContext: {
								allowPrivilegeEscalation: false
								capabilities: {
									drop: [
										"ALL",
									]
								}
							}
							resources: {
								limits: {
									memory: "1.5Gi"
									cpu:    "2000m"
								}
								requests: {
									memory: "1.5Gi"
									cpu:    "500m"
								}
							}
							ports: [
								{
									name:          "http"
									protocol:      "TCP"
									containerPort: 8080
								},
							]
							volumeMounts: [
								{
									name:      "primary"
									mountPath: "/inventory"
								},
								{
									name:      "podinfo"
									mountPath: "/podinfo"
									readOnly:  true
								},
								{
									name:      "ssh"
									mountPath: "/.ssh"
									readOnly:  true
								},
								{
									name:      "cache"
									mountPath: "/.cache"
								},
								{
									name:      "repository"
									mountPath: "/repository"
								},
							]
						},
					]
				}
			}
		}
	}
}

primaryService: component.#Manifest & {
	dependencies: [
		ns.id,
		primaryProjectControllerDeployment.id,
	]
	content: {
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			name:      "project-controller-primary"
			namespace: ns.content.metadata.name
			labels:    _primaryLabels
		}
		spec: {
			clusterIP: "None"
			selector:  _primaryLabels
			ports: [
				{
					name:       "http"
					protocol:   "TCP"
					port:       8080
					targetPort: "http"
				},
			]
		}
	}
}
