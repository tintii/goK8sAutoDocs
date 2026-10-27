package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type resourceReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type endpoint struct {
	URL      string             `json:"url"`
	Host     string             `json:"host"`
	Scheme   string             `json:"scheme"`
	Port     int32              `json:"port"`
	Gateway  *resourceReference `json:"gateway,omitempty"`
	Listener string             `json:"listener,omitempty"`
}

type directoryEntry struct {
	Name             string              `json:"name"`
	Namespace        string              `json:"namespace"`
	Kind             string              `json:"kind"`
	IngressClass     *string             `json:"ingress-class,omitempty"`
	Hostnames        []map[string]string `json:"hostnames"`
	Labels           map[string]string   `json:"labels"`
	Annotations      map[string]string   `json:"annotations"`
	RelatedResources []RelatedResource   `json:"relatedResources"`
	Endpoints        []endpoint          `json:"endpoints"`
	Issues           []string            `json:"issues"`
}

func newDirectoryEntry(object metav1.Object, kind string) directoryEntry {
	return directoryEntry{
		Name: object.GetName(), Namespace: object.GetNamespace(), Kind: kind,
		Labels: object.GetLabels(), Annotations: object.GetAnnotations(),
		Hostnames: []map[string]string{}, RelatedResources: []RelatedResource{},
		Endpoints: []endpoint{}, Issues: []string{},
	}
}

func (entry *directoryEntry) addHost(host string) {
	for _, existing := range entry.Hostnames {
		if existing["host"] == host {
			return
		}
	}
	if host != "" {
		entry.Hostnames = append(entry.Hostnames, map[string]string{"host": host})
	}
}

func (entry *directoryEntry) addEndpoint(host, scheme string, port int32, gateway *resourceReference, listener string) {
	authority := host
	if (scheme == "http" && port != 80) || (scheme == "https" && port != 443) {
		authority = net.JoinHostPort(host, strconv.Itoa(int(port)))
	}
	destination := (&url.URL{Scheme: scheme, Host: authority}).String()
	for _, existing := range entry.Endpoints {
		if existing.URL == destination {
			return
		}
	}
	entry.addHost(host)
	entry.Endpoints = append(entry.Endpoints, endpoint{
		URL: destination, Host: host, Scheme: scheme, Port: port, Gateway: gateway, Listener: listener,
	})
}

func (entry *directoryEntry) addRelated(resource RelatedResource) {
	for _, existing := range entry.RelatedResources {
		if existing.Kind == resource.Kind && existing.Namespace == resource.Namespace && existing.Name == resource.Name {
			return
		}
	}
	entry.RelatedResources = append(entry.RelatedResources, resource)
}

func ingressEntry(ingress networkingv1.Ingress) directoryEntry {
	entry := newDirectoryEntry(&ingress, "ingress") // Preserve the existing Ingress API kind.
	entry.IngressClass = ingress.Spec.IngressClassName
	for _, rule := range ingress.Spec.Rules {
		host := rule.Host
		entry.addHost(host)
		if host == "" || strings.Contains(host, "*") {
			entry.Issues = append(entry.Issues, "Ingress rule has no concrete hostname")
			continue
		}
		scheme, port := "http", int32(80)
		for _, tls := range ingress.Spec.TLS {
			for _, tlsHost := range tls.Hosts {
				if tlsHost == host || (strings.HasPrefix(tlsHost, "*.") && strings.Count(host, ".") == strings.Count(tlsHost, ".") && strings.HasSuffix(host, tlsHost[1:])) {
					scheme, port = "https", 443
				}
			}
		}
		entry.addEndpoint(host, scheme, port, nil, "")
	}
	if len(ingress.Spec.Rules) == 0 {
		entry.Issues = append(entry.Issues, "Ingress has no rule hostname")
	}
	return entry
}

// These structs decode only the Gateway API v1 fields needed by the directory.
type parentReference struct {
	Group       *string `json:"group"`
	Kind        *string `json:"kind"`
	Namespace   string  `json:"namespace"`
	Name        string  `json:"name"`
	SectionName string  `json:"sectionName"`
	Port        *int32  `json:"port"`
}

func (parent parentReference) effective(namespace string) parentReference {
	if parent.Namespace == "" {
		parent.Namespace = namespace
	}
	if parent.Group == nil {
		group := "gateway.networking.k8s.io"
		parent.Group = &group
	}
	if parent.Kind == nil {
		kind := "Gateway"
		parent.Kind = &kind
	}
	return parent
}

func sameParent(a, b parentReference, namespace string) bool {
	a, b = a.effective(namespace), b.effective(namespace)
	return *a.Group == *b.Group && *a.Kind == *b.Kind && a.Name == b.Name && a.Namespace == b.Namespace &&
		a.SectionName == b.SectionName && ((a.Port == nil && b.Port == nil) || (a.Port != nil && b.Port != nil && *a.Port == *b.Port))
}

type backendReference struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type routeSpec struct {
	Hostnames  []string          `json:"hostnames"`
	ParentRefs []parentReference `json:"parentRefs"`
	Rules      []struct {
		BackendRefs []backendReference `json:"backendRefs"`
	} `json:"rules"`
}

type gatewayListener struct {
	Name          string `json:"name"`
	Hostname      string `json:"hostname"`
	Protocol      string `json:"protocol"`
	Port          int32  `json:"port"`
	AllowedRoutes *struct {
		Namespaces *struct {
			From     string                `json:"from"`
			Selector *metav1.LabelSelector `json:"selector"`
		} `json:"namespaces"`
		Kinds []struct {
			Group *string `json:"group"`
			Kind  string  `json:"kind"`
		} `json:"kinds"`
	} `json:"allowedRoutes"`
}

type gatewayDocument struct {
	Spec struct {
		Listeners []gatewayListener `json:"listeners"`
	} `json:"spec"`
	Status struct {
		Listeners []struct {
			Name       string             `json:"name"`
			Conditions []metav1.Condition `json:"conditions"`
		} `json:"listeners"`
	} `json:"status"`
}

func decodeObject(object any, into any) error {
	data, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, into)
}

// Gateway wildcards match one or more DNS labels. Empty hostnames match everything.
func intersectHostname(a, b string) (string, bool) {
	if a == "" || a == b {
		return b, true
	}
	if b == "" {
		return a, true
	}
	if strings.HasPrefix(a, "*.") && strings.HasSuffix(b, a[1:]) {
		return b, true
	}
	if strings.HasPrefix(b, "*.") && strings.HasSuffix(a, b[1:]) {
		return a, true
	}
	return "", false
}

type gatewayLookup struct {
	object *unstructured.Unstructured
	err    error
}

type endpointCollector struct {
	kube       kubernetes.Interface
	dynamic    dynamic.Interface
	gateways   map[resourceReference]gatewayLookup
	namespaces map[string]map[string]string
}

func collectAutoDocs(ctx context.Context, kube kubernetes.Interface, client dynamic.Interface, namespace string) ([]directoryEntry, []string, error) {
	return collectAutoDocsView(ctx, kube, client, namespace, publishedView)
}

func collectAutoDocsView(ctx context.Context, kube kubernetes.Interface, client dynamic.Interface, namespace string, view directoryView) ([]directoryEntry, []string, error) {
	entries := make([]directoryEntry, 0)
	warnings := make([]string, 0)
	ingresses, err := kube.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{LabelSelector: view.selector()})
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Cannot list Ingresses: %v", err))
	} else {
		for _, ingress := range ingresses.Items {
			if !view.includes(ingress.Labels) {
				continue
			}
			entry := ingressEntry(ingress)
			related, err := findRelatedIngressResources(ctx, kube, ingress)
			if err != nil {
				entry.Issues = append(entry.Issues, fmt.Sprintf("Related resources unavailable: %v", err))
			}
			for _, resource := range related {
				entry.addRelated(resource)
			}
			entries = append(entries, entry)
		}
	}
	routes, err := client.Resource(httpRouteGVR).Namespace(namespace).List(ctx, metav1.ListOptions{LabelSelector: view.selector()})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			warnings = append(warnings, fmt.Sprintf("Cannot list HTTPRoutes: %v", err))
		}
	} else {
		collector := endpointCollector{kube: kube, dynamic: client, gateways: map[resourceReference]gatewayLookup{}, namespaces: map[string]map[string]string{}}
		for _, route := range routes.Items {
			if view.includes(route.GetLabels()) {
				entries = append(entries, collector.routeEntry(ctx, route))
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		return a.Namespace+"/"+a.Kind+"/"+a.Name < b.Namespace+"/"+b.Kind+"/"+b.Name
	})
	return entries, warnings, ctx.Err()
}

func (collector *endpointCollector) routeEntry(ctx context.Context, route unstructured.Unstructured) directoryEntry {
	entry := newDirectoryEntry(&route, "HTTPRoute")
	var spec routeSpec
	if err := decodeObject(route.Object["spec"], &spec); err != nil {
		entry.Issues = append(entry.Issues, fmt.Sprintf("Cannot read HTTPRoute spec: %v", err))
		return entry
	}
	for _, host := range spec.Hostnames {
		entry.addHost(host)
	}
	for _, reference := range spec.ParentRefs {
		parent := reference.effective(route.GetNamespace())
		if *parent.Group != gatewayGVR.Group || *parent.Kind != "Gateway" {
			entry.Issues = append(entry.Issues, fmt.Sprintf("Unsupported parent %s/%s", *parent.Kind, parent.Name))
			continue
		}
		key := resourceReference{Name: parent.Name, Namespace: parent.Namespace}
		lookup, exists := collector.gateways[key]
		if !exists {
			lookup.object, lookup.err = collector.dynamic.Resource(gatewayGVR).Namespace(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
			collector.gateways[key] = lookup
		}
		entry.addRelated(RelatedResource{Name: key.Name, Namespace: key.Namespace, Kind: "Gateway"})
		if lookup.err != nil {
			entry.Issues = append(entry.Issues, fmt.Sprintf("Cannot read Gateway %s/%s: %v", key.Namespace, key.Name, lookup.err))
			continue
		}
		if rejected, issue := routeAttachment(route, parent); issue != "" {
			entry.Issues = append(entry.Issues, fmt.Sprintf("Gateway %s/%s: %s", key.Namespace, key.Name, issue))
			if rejected {
				continue
			}
		}
		var gateway gatewayDocument
		if err := decodeObject(lookup.object.Object, &gateway); err != nil {
			entry.Issues = append(entry.Issues, fmt.Sprintf("Cannot read Gateway listeners: %v", err))
			continue
		}
		before := len(entry.Endpoints)
		matched := false
		for _, listener := range gateway.Spec.Listeners {
			if (parent.SectionName != "" && listener.Name != parent.SectionName) || (parent.Port != nil && listener.Port != *parent.Port) {
				continue
			}
			if listener.Protocol != "HTTP" && listener.Protocol != "HTTPS" {
				continue
			}
			if listener.Port < 1 || listener.Port > 65535 {
				continue
			}
			if err := collector.listenerAllows(ctx, listener, route.GetNamespace(), key.Namespace); err != nil {
				entry.Issues = append(entry.Issues, fmt.Sprintf("Listener %s: %v", listener.Name, err))
				continue
			}
			rejected := false
			for _, status := range gateway.Status.Listeners {
				if status.Name == listener.Name {
					for _, condition := range status.Conditions {
						if condition.Type == "Accepted" && condition.ObservedGeneration == lookup.object.GetGeneration() && condition.Status == metav1.ConditionFalse {
							entry.Issues = append(entry.Issues, fmt.Sprintf("Listener %s rejected: %s", listener.Name, condition.Reason))
							rejected = true
						}
					}
				}
			}
			if rejected {
				continue
			}
			hosts := spec.Hostnames
			if len(hosts) == 0 {
				hosts = []string{""}
			}
			for _, host := range hosts {
				effective, intersects := intersectHostname(host, listener.Hostname)
				if !intersects {
					continue
				}
				matched = true
				if effective == "" || strings.Contains(effective, "*") {
					entry.Issues = append(entry.Issues, fmt.Sprintf("Listener %s has no concrete hostname for this route", listener.Name))
					continue
				}
				entry.addEndpoint(effective, strings.ToLower(listener.Protocol), listener.Port, &key, listener.Name)
			}
		}
		if !matched && len(entry.Endpoints) == before {
			entry.Issues = append(entry.Issues, fmt.Sprintf("Gateway %s/%s has no compatible listener and hostname for this route", key.Namespace, key.Name))
		}
	}
	if len(spec.ParentRefs) == 0 {
		entry.Issues = append(entry.Issues, "HTTPRoute has no Gateway parent")
	}
	collector.addRouteBackends(ctx, spec, &entry)
	return entry
}

func routeAttachment(route unstructured.Unstructured, parent parentReference) (bool, string) {
	var status struct {
		Parents []struct {
			ParentRef  parentReference    `json:"parentRef"`
			Conditions []metav1.Condition `json:"conditions"`
		} `json:"parents"`
	}
	if err := decodeObject(route.Object["status"], &status); err != nil {
		return false, "attachment status unknown"
	}
	accepted := false
	var issues []string
	for _, observed := range status.Parents {
		if !sameParent(parent, observed.ParentRef, route.GetNamespace()) {
			continue
		}
		for _, condition := range observed.Conditions {
			if condition.ObservedGeneration != route.GetGeneration() {
				continue
			}
			if condition.Type == "Accepted" {
				if condition.Status == metav1.ConditionFalse {
					return true, "attachment rejected: " + condition.Reason
				}
				accepted = condition.Status == metav1.ConditionTrue
			}
			if condition.Type == "ResolvedRefs" && condition.Status == metav1.ConditionFalse {
				issues = append(issues, "unresolved references: "+condition.Reason)
			}
		}
	}
	if !accepted {
		issues = append(issues, "attachment status unknown or stale; URL reflects configuration only")
	}
	return false, strings.Join(issues, "; ")
}

func (collector *endpointCollector) listenerAllows(ctx context.Context, listener gatewayListener, routeNamespace, gatewayNamespace string) error {
	from := "Same"
	if allowed := listener.AllowedRoutes; allowed != nil {
		if len(allowed.Kinds) > 0 {
			found := false
			for _, kind := range allowed.Kinds {
				if kind.Kind == "HTTPRoute" && (kind.Group == nil || *kind.Group == gatewayGVR.Group) {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("HTTPRoutes are not allowed")
			}
		}
		if allowed.Namespaces != nil && allowed.Namespaces.From != "" {
			from = allowed.Namespaces.From
		}
	}
	switch from {
	case "All":
		return nil
	case "Same":
		if routeNamespace == gatewayNamespace {
			return nil
		}
	case "Selector":
		selectorSpec := listener.AllowedRoutes.Namespaces.Selector
		if selectorSpec == nil {
			return fmt.Errorf("namespace selector is missing")
		}
		selector, err := metav1.LabelSelectorAsSelector(selectorSpec)
		if err != nil {
			return fmt.Errorf("invalid namespace selector: %w", err)
		}
		namespaceLabels, exists := collector.namespaces[routeNamespace]
		if !exists {
			namespace, err := collector.kube.CoreV1().Namespaces().Get(ctx, routeNamespace, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("cannot check namespace selector: %w", err)
			}
			namespaceLabels = namespace.Labels
			collector.namespaces[routeNamespace] = namespaceLabels
		}
		if selector.Matches(labels.Set(namespaceLabels)) {
			return nil
		}
	}
	return fmt.Errorf("namespace %s is not allowed", routeNamespace)
}

func (collector *endpointCollector) addRouteBackends(ctx context.Context, spec routeSpec, entry *directoryEntry) {
	seen := map[resourceReference]bool{}
	for _, rule := range spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Group != "" || (backend.Kind != "" && backend.Kind != "Service") {
				entry.Issues = append(entry.Issues, fmt.Sprintf("Unsupported backend %s/%s", backend.Kind, backend.Name))
				continue
			}
			if backend.Namespace == "" {
				backend.Namespace = entry.Namespace
			}
			key := resourceReference{Name: backend.Name, Namespace: backend.Namespace}
			if seen[key] {
				continue
			}
			seen[key] = true
			entry.addRelated(RelatedResource{Name: key.Name, Namespace: key.Namespace, Kind: "Service"})
			service, err := collector.kube.CoreV1().Services(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
			if err != nil {
				entry.Issues = append(entry.Issues, fmt.Sprintf("Cannot read backend Service %s/%s: %v", key.Namespace, key.Name, err))
				continue
			}
			// Replace the reference with the observed metadata.
			for i := range entry.RelatedResources {
				resource := &entry.RelatedResources[i]
				if resource.Kind == "Service" && resource.Name == key.Name && resource.Namespace == key.Namespace {
					resource.Labels, resource.Selector = service.Labels, service.Spec.Selector
				}
			}
			if len(service.Spec.Selector) == 0 {
				continue // Selectorless Services must not match every Pod in a namespace.
			}
			pods, err := collector.kube.CoreV1().Pods(key.Namespace).List(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(service.Spec.Selector).String()})
			if err != nil {
				entry.Issues = append(entry.Issues, fmt.Sprintf("Cannot read Pods for Service %s/%s: %v", key.Namespace, key.Name, err))
				continue
			}
			for _, pod := range pods.Items {
				resource := RelatedResource{Name: pod.Name, Namespace: pod.Namespace, Kind: "Pod", Labels: pod.Labels}
				if len(pod.Status.ContainerStatuses) > 0 {
					resource.ImageName, resource.ImageTag, resource.ImageSha = parseImage(pod.Status.ContainerStatuses[0].ImageID)
				}
				entry.addRelated(resource)
			}
		}
	}
}
