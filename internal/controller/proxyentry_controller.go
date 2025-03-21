package controller

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sproxyv1alpha1 "app-scape.de/api/v1alpha1"
)

// var finalizerName string = "k8sproxy.app-scape.de/finalizer"

// ProxyEntryReconciler reconciles a ProxyEntry object
type ProxyEntryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8sproxy.app-scape.de,resources=proxyentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8sproxy.app-scape.de,resources=proxyentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8sproxy.app-scape.de,resources=proxyentries/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

func (r *ProxyEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	var err error

	// Fetch the ProxyEntry instance-----------------------------------------------------------
	pe := &k8sproxyv1alpha1.ProxyEntry{}
	if err := r.Get(ctx, req.NamespacedName, pe); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ProxyEntry resource not found. Ignoring since it must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get ProxyEntry")
		return ctrl.Result{}, err
	}

	// Manage Sub Resources--------------------------------------------------------------------
	subResourceName := subResourcePreFix + pe.Name

	// The Service-------------
	// Check if the Service already exists, if not create a new one
	svcFound := &v1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: subResourceName, Namespace: pe.Namespace}, svcFound)

	// Create The Service---------------------------------------------
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new Service
		svc := r.serviceForProxyentry(pe)
		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		if err := r.Client.Create(context.TODO(), svc); err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return ctrl.Result{}, err
		}
		// Requeue the request to ensure the Service is created
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	// Reset The Service-------------
	if svcFound.Spec.Type != v1.ServiceType(pe.Spec.Service.Type) ||
		svcFound.Spec.Ports[0].Port != pe.Spec.Service.Port {

		log.Info("Resetting Service", "Service.Namespace", svcFound.Namespace, "Service.Name", svcFound.Name)

		svcFound.Spec.Type = v1.ServiceType(pe.Spec.Service.Type)
		svcFound.Spec.Ports[0].Port = pe.Spec.Service.Port
		svcFound.Spec.Ports[0].TargetPort = intstr.FromInt(int(pe.Spec.Service.Port))

		if err := r.Update(ctx, svcFound); err != nil {
			log.Error(err, "Failed to update Service Specs", "Service.Namespace", svcFound.Namespace, "Service.Name", svcFound.Name)
			return ctrl.Result{}, err
		}
		// Requeue the request to ensure the correct state is achieved
		return ctrl.Result{Requeue: true}, nil
	}

	// The Endpoints-------------------------------------------------
	// Check if the Endpoints already exists, if not create a new one
	epsFound := &v1.Endpoints{}
	err = r.Get(ctx, types.NamespacedName{Name: subResourceName, Namespace: pe.Namespace}, epsFound)

	// Create The Endpoints-------------
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new Endpoints
		eps := r.endpointsForProxyentry(pe)
		log.Info("Creating a new Endpoints", "Endpoints.Namespace", eps.Namespace, "Endpoints.Name", eps.Name)
		if err := r.Client.Create(context.TODO(), eps); err != nil {
			log.Error(err, "Failed to create new Endpoints", "Endpoints.Namespace", eps.Namespace, "Endpoints.Name", eps.Name)
			return ctrl.Result{}, err
		}
		// Requeue the request to ensure the Endpoints is created
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Endpoints")
		return ctrl.Result{}, err
	}

	// Reset The Endpoints-------------
	if epsFound.Subsets[0].Addresses[0].IP != pe.Spec.Endpoints.Ip ||
		epsFound.Subsets[0].Ports[0].Port != pe.Spec.Endpoints.Port ||
		epsFound.Subsets[0].Ports[0].Protocol != v1.ProtocolTCP {

		log.Info("Resetting Endpoints", "Endpoints.Namespace", svcFound.Namespace, "Endpoints.Name", svcFound.Name)

		epsFound.Subsets[0].Addresses[0].IP = pe.Spec.Endpoints.Ip
		epsFound.Subsets[0].Ports[0].Port = pe.Spec.Endpoints.Port
		epsFound.Subsets[0].Ports[0].Protocol = v1.ProtocolTCP

		if err := r.Update(ctx, epsFound); err != nil {
			log.Error(err, "Failed to update Endpoints Specs", "Endpoints.Namespace", epsFound.Namespace, "Endpoints.Name", epsFound.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// The Ingress-----------------------------------------------------
	// Check if the Ingress already exists, if not create a new one
	ing := r.ingressForProxyentry(pe)
	ingFound := &networkingv1.Ingress{}
	err = r.Get(ctx, types.NamespacedName{Name: subResourceName, Namespace: pe.Namespace}, ingFound)

	// Create The Ingress-------------
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new Ingress
		log.Info("Creating a new Ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
		if err := r.Client.Create(context.TODO(), ing); err != nil {
			log.Error(err, "Failed to create new Ingress", "Ingress.Namespace", ing.Namespace, "Ingress.Name", ing.Name)
			return ctrl.Result{}, err
		}
		// Requeue the request to ensure the Ingress is created
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Ingress")
		return ctrl.Result{}, err
	}

	// Reset The Ingress-------------
	chk := false
	cmAnno := "cert-manager.io/cluster-issuer"
	bepAnnos := []string{
		"nginx.ingress.kubernetes.io/backend-protocol",
		"traefik.ingress.kubernetes.io/service.serversscheme",
	}

	if ingFound.ObjectMeta.Annotations[bepAnnos[0]] != ing.ObjectMeta.Annotations[bepAnnos[0]] {
		for idx, _ := range bepAnnos {
			ingFound.ObjectMeta.Annotations[bepAnnos[idx]] = ing.ObjectMeta.Annotations[bepAnnos[idx]]
		}
		chk = true
	}
	if *ingFound.Spec.IngressClassName != *ing.Spec.IngressClassName {
		*ingFound.Spec.IngressClassName = *ing.Spec.IngressClassName
		chk = true
	}
	if ingFound.Spec.Rules[0].Host != ing.Spec.Rules[0].Host {
		ingFound.Spec.Rules[0].Host = ing.Spec.Rules[0].Host
		if ing.Spec.TLS != nil {
			ingFound.Spec.TLS[0].Hosts[0] = ing.Spec.Rules[0].Host
		}
		chk = true
	}
	if ingFound.Spec.TLS == nil && ing.Spec.TLS != nil {
		ingFound.Spec.TLS = ing.Spec.TLS
		chk = true
	}
	if ingFound.Spec.TLS != nil && ing.Spec.TLS == nil {
		ingFound.Spec.TLS = nil
		chk = true
	}
	if ingFound.Spec.TLS != nil && ingFound.Spec.TLS[0].SecretName != ing.Spec.TLS[0].SecretName {
		ingFound.Spec.TLS[0].SecretName = ing.Spec.TLS[0].SecretName
		chk = true
	}
	if ingFound.ObjectMeta.Annotations[cmAnno] != ing.ObjectMeta.Annotations[cmAnno] {
		ingFound.ObjectMeta.Annotations[cmAnno] = ing.ObjectMeta.Annotations[cmAnno]
		chk = true
	}

	if chk {
		log.Info("Resetting Ingress", "Ingress.Namespace", ingFound.Namespace, "Ingress.Name", ingFound.Name)
		if err := r.Update(ctx, ingFound); err != nil {
			log.Error(err, "Failed to update Ingress Specs", "Ingress.Namespace", ingFound.Namespace, "Ingress.Name", ingFound.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager-------------------------------------
func (r *ProxyEntryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sproxyv1alpha1.ProxyEntry{}).
		// Named("proxyentry").
		Owns(&v1.Service{}).
		Owns(&v1.Endpoints{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
