package declcd

import (
	"github.com/kharf/declcd/schema/component"
)

_name: "gitops-controller"
_labels: {
	"declcd/control-plane": _name
}

// _crd is autogenerated into crd.cue
crd: component.#Manifest & {
	content: _crd
}

ns: component.#Manifest & {
	dependencies: [crd.id]
	content: {
		apiVersion: "v1"
		kind:       "Namespace"
		metadata: {
			name:   "declcd-system"
			labels: _labels
		}
	}
}

serviceAccount: component.#Manifest & {
	dependencies: [ns.id]
	content: {
		apiVersion: "v1"
		kind:       "ServiceAccount"
		metadata: {
			name:      _name
			namespace: ns.content.metadata.name
			labels:    _labels
		}
	}
}

_leaderRoleName: "leader-election"
_roleGroup:      "rbac.authorization.k8s.io"
_roleApiVersion: "\(_roleGroup)/v1"
leaderRole: component.#Manifest & {
	dependencies: [ns.id]
	content: {
		apiVersion: _roleApiVersion
		kind:       "Role"
		metadata: {
			name:      _leaderRoleName
			namespace: ns.content.metadata.name
			labels:    _labels
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

leaderRoleBinding: component.#Manifest & {
	dependencies: [
		ns.id,
		leaderRole.id,
	]
	content: {
		apiVersion: _roleApiVersion
		kind:       "RoleBinding"
		metadata: {
			name:      _leaderRoleName
			namespace: ns.content.metadata.name
			labels:    _labels
		}
		roleRef: {
			apiGroup: _roleGroup
			kind:     leaderRole.content.kind
			name:     leaderRole.content.metadata.name
		}
		subjects: [
			{
				kind:      serviceAccount.content.kind
				name:      serviceAccount.content.metadata.name
				namespace: serviceAccount.content.metadata.namespace
			},
		]
	}
}

clusterRole: component.#Manifest & {
	dependencies: [ns.id]
	content: {
		apiVersion: _roleApiVersion
		kind:       "ClusterRole"
		metadata: {
			name:      _name
			namespace: ns.content.metadata.name
			labels:    _labels
		}
		rules: [
			{
				apiGroups: ["gitops.declcd.io"]
				resources: ["gitopsprojects"]
				verbs: [
					"list",
					"watch",
				]
			},
			{
				apiGroups: ["gitops.declcd.io"]
				resources: ["gitopsprojects/status"]
				verbs: [
					"get",
					"patch",
					"update",
				]
			},
			{
				apiGroups: ["*"]
				resources: ["*"]
				verbs: [
					"*",
				]
			},
		]
	}
}

clusteRoleBinding: component.#Manifest & {
	dependencies: [
		ns.id,
		clusterRole.id,
	]
	content: {
		apiVersion: _roleApiVersion
		kind:       "ClusterRoleBinding"
		metadata: {
			name:      _name
			namespace: ns.content.metadata.name
			labels:    _labels
		}
		roleRef: {
			apiGroup: _roleGroup
			kind:     clusterRole.content.kind
			name:     clusterRole.content.metadata.name
		}
		subjects: [
			{
				kind:      serviceAccount.content.kind
				name:      serviceAccount.content.metadata.name
				namespace: serviceAccount.content.metadata.namespace
			},
		]
	}
}

statefulSet: component.#Manifest & {
	dependencies: [
		ns.id,
	]
	content: {
		apiVersion: "apps/v1"
		kind:       "StatefulSet"
		metadata: {
			name:      _name
			namespace: ns.content.metadata.name
			labels:    _labels
		}
		spec: {
			selector: matchLabels: _labels
			serviceName: _name
			replicas:    1
			volumeClaimTemplates: [
				{
					metadata: name: "declcd"
					spec: {
						accessModes: [
							"ReadWriteOnce",
						]
						resources: {
							requests: {
								storage: "20Mi"
							}
						}
					}
				},
			]
			template: {
				metadata: {
					labels: _labels
				}
				spec: {
					serviceAccountName: _name
					securityContext: {
						runAsNonRoot:        true
						fsGroup:             65532
						fsGroupChangePolicy: "OnRootMismatch"
					}
					volumes: [
						{
							name: "podinfo"
							downwardAPI: {
								items: [
									{
										path: "namespace"
										fieldRef: fieldPath: "metadata.namespace"
									},
								]
							}
						},
					]
					containers: [
						{
							name:  _name
							image: "ghcr.io/kharf/declcd:0.20.0"
							command: [
								"/controller",
							]
							args: [
								"--leader-elect",
								"--log-level=0",
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
									name:      "declcd"
									mountPath: "/inventory"
								},
								{
									name:      "podinfo"
									mountPath: "/podinfo"
								},
							]
						},
					]
				}
			}
		}
	}
}

service: component.#Manifest & {
	dependencies: [
		ns.id,
		statefulSet.id,
	]
	content: {
		apiVersion: "v1"
		kind:       "Service"
		metadata: {
			name:      _name
			namespace: ns.content.metadata.name
			labels:    _labels
		}
		spec: {
			clusterIP: "None"
			selector:  _labels
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