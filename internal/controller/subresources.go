package controller

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8sproxyv1alpha1 "app-scape.de/api/v1alpha1"
)

var subResourcePreFix string = "k8sproxy-"

// Create the Service SubResource Object for a ProxyEntry
func (r *ProxyEntryReconciler) serviceForProxyentry(pe *k8sproxyv1alpha1.ProxyEntry) *v1.Service {
	port := pe.Spec.Service.Port
	portName := strings.ToLower(pe.Spec.Ingress.BackendProtocol)
	typ := pe.Spec.Service.Type
	name := subResourcePreFix + pe.Name

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pe.Namespace,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceType(typ),
			Ports: []v1.ServicePort{
				{
					Name:       portName,
					Protocol:   v1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromInt(int(port)), // WAS FÜR EINE SCH.....!!!!
				},
			},
		},
	}
	controllerutil.SetControllerReference(pe, svc, r.Scheme)
	return svc
}

// Create the Endpoints SubResource Object for a ProxyEntry
func (r *ProxyEntryReconciler) endpointsForProxyentry(pe *k8sproxyv1alpha1.ProxyEntry) *v1.Endpoints {
	port := pe.Spec.Endpoints.Port
	portName := strings.ToLower(pe.Spec.Ingress.BackendProtocol)
	ip := pe.Spec.Endpoints.Ip
	name := subResourcePreFix + pe.Name

	eps := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pe.Namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: ip,
					},
				},
				Ports: []v1.EndpointPort{
					{
						Name:     portName,
						Port:     port,
						Protocol: v1.ProtocolTCP,
					},
				},
			},
		},
	}
	controllerutil.SetControllerReference(pe, eps, r.Scheme)
	return eps
}

// Create the Ingress SubResource Object for a ProxyEntry
func (r *ProxyEntryReconciler) ingressForProxyentry(pe *k8sproxyv1alpha1.ProxyEntry) *networkingv1.Ingress {
	className := pe.Spec.Ingress.ClassName
	beProtocol := pe.Spec.Ingress.BackendProtocol
	host := pe.Spec.Ingress.Host
	port := pe.Spec.Service.Port
	clusterIssuer := pe.Spec.Ingress.ClusterIssuer
	name := subResourcePreFix + pe.Name

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pe.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/backend-protocol":        beProtocol,
				"traefik.ingress.kubernetes.io/service.serversscheme": beProtocol,
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &className,
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path: "/",
									PathType: func() *networkingv1.PathType {
										pt := networkingv1.PathTypePrefix
										return &pt
									}(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: name,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if pe.Spec.Ingress.Tls {
		var secretName string = name + "-tls"
		if pe.Spec.Ingress.TlsSecretName != "" {
			secretName = pe.Spec.Ingress.TlsSecretName
		}
		ing.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      []string{pe.Spec.Ingress.Host},
				SecretName: secretName,
			},
		}
		if clusterIssuer != "" {
			ing.ObjectMeta.Annotations["cert-manager.io/cluster-issuer"] = clusterIssuer
		}
	}
	controllerutil.SetControllerReference(pe, ing, r.Scheme)
	return ing
}
