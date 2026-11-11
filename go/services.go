package main

import (
	"fmt"
	"net/netip"
	"slices"
	"sort"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const externalDNSAlpha = "external-dns.alpha.kubernetes.io/"
const externalDNSStable = "external-dns.kubernetes.io/"
const serviceScheme = "k8sautodocs.io/scheme"
const servicePort = "k8sautodocs.io/port"

type serviceDNS struct {
	Targets      []string `json:"targets"`
	TargetSource string   `json:"targetSource,omitempty"`
}

func hasDNSAnnotation(annotations map[string]string, field string) bool {
	_, alpha := annotations[externalDNSAlpha+field]
	_, stable := annotations[externalDNSStable+field]
	return alpha || stable
}

func eligibleService(service corev1.Service) bool {
	if value, present := service.Labels[publishLabel]; present {
		return value == "true"
	}
	return hasDNSAnnotation(service.Annotations, "hostname")
}

// Accept DNS declarations only, never URL authority syntax. No network lookups.
func normalizeDNSValue(value string, target bool) (string, bool) {
	value = strings.TrimSpace(value)
	if address, err := netip.ParseAddr(value); err == nil {
		return address.String(), target && address.Zone() == ""
	}
	value = strings.ToLower(strings.TrimSuffix(value, "."))
	if _, err := netip.ParseAddr(value); err == nil {
		// An advertised name cannot be an IP literal, even with a final dot.
		return "", false
	}
	if len(value) == 0 || len(value) > 253 {
		return "", false
	}
	name := value
	if !target {
		name = strings.TrimPrefix(name, "*.")
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for _, char := range label {
			if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-') {
				return "", false
			}
		}
	}
	// Browsers also interpret shortened, integer, octal and hex IPv4 forms as
	// addresses. They must not masquerade as DNS names in a generated link.
	parts := strings.Split(name, ".")
	ipv4Like := len(parts) <= 4
	for _, label := range parts {
		hex := strings.HasPrefix(label, "0x")
		if hex {
			label = label[2:]
		}
		numeric := label != ""
		for _, char := range label {
			numeric = numeric && (char >= '0' && char <= '9' || hex && char >= 'a' && char <= 'f')
		}
		ipv4Like = ipv4Like && numeric
	}
	if ipv4Like {
		return "", false
	}
	return value, true
}

func parseDNSValues(values []string, target bool, field string) ([]string, []string) {
	set := map[string]bool{}
	issues := []string{}
	for _, value := range values {
		normalized, valid := normalizeDNSValue(value, target)
		if !valid {
			issues = append(issues, fmt.Sprintf("Invalid %s value: %q", field, value))
			continue
		}
		set[normalized] = true
	}
	result := []string{}
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	if len(values) == 0 {
		issues = append(issues, "Empty "+field+" declaration")
	}
	return result, issues
}

func annotationDNSValues(annotations map[string]string, field string) ([]string, []string) {
	parse := func(raw string) ([]string, []string) {
		values := []string{}
		for _, value := range strings.Split(raw, ",") {
			if value = strings.TrimSpace(value); value != "" {
				values = append(values, value)
			}
		}
		return parseDNSValues(values, field == "target", "ExternalDNS "+field)
	}
	alpha, a := annotations[externalDNSAlpha+field]
	stable, b := annotations[externalDNSStable+field]
	if a && b {
		av, ai := parse(alpha)
		bv, bi := parse(stable)
		if len(ai)+len(bi) != 0 || !slices.Equal(av, bv) {
			return []string{}, []string{"Conflicting ExternalDNS " + field + " annotations; both prefixes must declare the same valid values"}
		}
		return av, nil
	}
	if a {
		return parse(alpha)
	}
	if b {
		return parse(stable)
	}
	return []string{}, nil
}

func serviceEntry(service corev1.Service) directoryEntry {
	entry := newDirectoryEntry(&service, "Service")
	entry.ServiceType = string(service.Spec.Type)
	if entry.ServiceType == "" {
		entry.ServiceType = string(corev1.ServiceTypeClusterIP)
	}
	entry.ExternalName = service.Spec.ExternalName
	entry.DNS = &serviceDNS{Targets: []string{}}
	hosts, issues := annotationDNSValues(service.Annotations, "hostname")
	entry.Issues = append(entry.Issues, issues...)
	if !hasDNSAnnotation(service.Annotations, "hostname") {
		entry.Issues = append(entry.Issues, "Service has no ExternalDNS hostname declaration")
	}
	for _, host := range hosts {
		entry.addHost(host)
	}

	var targets []string
	switch {
	case hasDNSAnnotation(service.Annotations, "target"):
		entry.DNS.TargetSource = "annotation"
		entry.DNS.Targets, issues = annotationDNSValues(service.Annotations, "target")
		entry.Issues = append(entry.Issues, issues...)
	case (service.Spec.Type == corev1.ServiceTypeExternalName || service.Spec.Type == corev1.ServiceTypeLoadBalancer) && len(service.Spec.ExternalIPs) > 0:
		entry.DNS.TargetSource, targets = "externalIPs", service.Spec.ExternalIPs
	case service.Spec.Type == corev1.ServiceTypeExternalName:
		entry.DNS.TargetSource, targets = "externalName", []string{service.Spec.ExternalName}
	case service.Spec.Type == corev1.ServiceTypeLoadBalancer:
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				targets = append(targets, ingress.IP)
			}
			if ingress.Hostname != "" {
				targets = append(targets, ingress.Hostname)
			}
		}
		if len(targets) > 0 {
			entry.DNS.TargetSource = "loadBalancer"
		}
	}
	if targets != nil {
		entry.DNS.Targets, issues = parseDNSValues(targets, true, entry.DNS.TargetSource+" target")
		entry.Issues = append(entry.Issues, issues...)
	}

	scheme, schemePresent := service.Annotations[serviceScheme]
	portText, portPresent := service.Annotations[servicePort]
	scheme, portText = strings.TrimSpace(scheme), strings.TrimSpace(portText)
	validWeb := false
	port := int64(80)
	if schemePresent || portPresent {
		validWeb = schemePresent && (scheme == "http" || scheme == "https")
		if !validWeb {
			entry.Issues = append(entry.Issues, "Service web links require an explicit scheme of http or https")
		}
		if scheme == "https" {
			port = 443
		}
		if portPresent {
			parsed, err := strconv.ParseUint(portText, 10, 16)
			digits := portText != ""
			for _, char := range portText {
				digits = digits && char >= '0' && char <= '9'
			}
			if err != nil || !digits || parsed == 0 {
				entry.Issues = append(entry.Issues, "Service web port must be decimal digits in 1..65535")
				validWeb = false
			} else {
				port = int64(parsed)
			}
		}
	}
	concrete := 0
	for _, host := range hosts {
		if !strings.HasPrefix(host, "*.") {
			concrete++
			if validWeb {
				entry.addEndpoint(host, scheme, int32(port), nil, "")
			}
		}
	}
	if len(hosts) > 0 && concrete == 0 {
		entry.Issues = append(entry.Issues, "Wildcard DNS names have no concrete browser destination")
	}
	return entry
}
