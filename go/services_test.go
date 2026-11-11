package main

import (
	"context"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func serviceFixture() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "external", Namespace: "apps", Labels: map[string]string{publishLabel: "true"}, Annotations: map[string]string{externalDNSAlpha + "hostname": "App.Example.com."}},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName, ExternalName: "Vendor.Example.net."},
	}
}

func TestServiceEligibility(t *testing.T) {
	for _, kind := range []corev1.ServiceType{corev1.ServiceTypeClusterIP, corev1.ServiceTypeNodePort, corev1.ServiceTypeLoadBalancer, corev1.ServiceTypeExternalName} {
		for _, label := range []string{"true", "false", "True", "", "absent"} {
			for _, annotation := range []string{"absent", "", "bad://host", "app.example.com"} {
				s := serviceFixture()
				s.Spec.Type = kind
				s.Labels = map[string]string{publishLabel: label, "k8sautodocs.io/group": "media"}
				if label == "absent" {
					delete(s.Labels, publishLabel)
				}
				s.Annotations = map[string]string{externalDNSAlpha + "target": "192.0.2.1", publishLabel: "true"}
				if annotation != "absent" {
					s.Annotations[externalDNSStable+"hostname"] = annotation
				}
				want := label == "true" || label == "absent" && annotation != "absent"
				if eligibleService(s) != want {
					t.Fatalf("%s label=%q hostname=%q: eligibility != %v", kind, label, annotation, want)
				}
			}
		}
	}
	s := serviceFixture()
	s.Annotations = nil
	e := serviceEntry(s)
	if len(e.Issues) == 0 || len(e.Hostnames)+len(e.Endpoints) != 0 {
		t.Fatalf("missing hostname must stay unresolved: %+v", e)
	}
}

func TestServiceDNSNormalization(t *testing.T) {
	for _, tc := range []struct {
		name, alpha, stable string
		both, target        bool
		want                []string
		issue               bool
	}{
		{name: "normalized", alpha: " App.Example.com., app.example.com, second.example.com, ", want: []string{"app.example.com", "second.example.com"}},
		{name: "equivalent", alpha: "b.example.com,a.example.com", stable: "A.Example.Com., B.example.com", both: true, want: []string{"a.example.com", "b.example.com"}},
		{name: "conflict", alpha: "a.example.com", stable: "b.example.com", both: true, want: []string{}, issue: true},
		{name: "invalid conflict", alpha: "a.example.com,https://bad", stable: "a.example.com", both: true, want: []string{}, issue: true},
		{name: "partial", alpha: "good.example.com,https://bad.example/path,user@bad.example,evil:80,with space.example,192.0.2.1", want: []string{"good.example.com"}, issue: true},
		{name: "empty", alpha: " , , ", want: []string{}, issue: true},
		{name: "wildcard", alpha: "*.EXAMPLE.com.", want: []string{"*.example.com"}},
		{name: "browser IP forms", alpha: "127.1,2130706433,0x7f000001,0177.0.0.1,192.0.2.1.", want: []string{}, issue: true},
		{name: "unsafe wildcard", alpha: "a.*.example.com,*.com*,*", want: []string{}, issue: true},
		{name: "encoded IDNA", alpha: "xn--bcher-kva.example", want: []string{"xn--bcher-kva.example"}},
		{name: "invalid labels", alpha: "é.example,-a.example,a-.example,a..example," + strings.Repeat("a", 64) + ".example," + strings.Repeat("abc.", 64), want: []string{}, issue: true},
		{name: "IP targets", alpha: "2001:0db8:0000::1,192.0.2.10,Target.Example.,2001:db8::1", target: true, want: []string{"192.0.2.10", "2001:db8::1", "target.example"}},
		{name: "unsafe targets", alpha: "*.example.com,https://example.com,[2001:db8::1],host:443,fe80::1%eth0", target: true, want: []string{}, issue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field := "hostname"
			if tc.target {
				field = "target"
			}
			a := map[string]string{externalDNSAlpha + field: tc.alpha}
			if tc.both {
				a[externalDNSStable+field] = tc.stable
			}
			got, issues := annotationDNSValues(a, field)
			if !reflect.DeepEqual(got, tc.want) || (len(issues) > 0) != tc.issue {
				t.Fatalf("values=%v issues=%v; want=%v issue=%v", got, issues, tc.want, tc.issue)
			}
		})
	}
}

func TestServiceTargetPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		change       func(*corev1.Service)
		targets      []string
		issue        bool
	}{
		{"externalName", "externalName", func(*corev1.Service) {}, []string{"vendor.example.net"}, false},
		{"explicit", "annotation", func(s *corev1.Service) { s.Annotations[externalDNSStable+"target"] = "192.0.2.10" }, []string{"192.0.2.10"}, false},
		{"externalIPs", "externalIPs", func(s *corev1.Service) { s.Spec.ExternalIPs = []string{"2001:db8::1", "192.0.2.20"} }, []string{"192.0.2.20", "2001:db8::1"}, false},
		{"no fallback on invalid", "annotation", func(s *corev1.Service) { s.Annotations[externalDNSAlpha+"target"] = "" }, []string{}, true},
		{"no fallback on conflict", "annotation", func(s *corev1.Service) {
			s.Annotations[externalDNSAlpha+"target"] = "192.0.2.1"
			s.Annotations[externalDNSStable+"target"] = "192.0.2.2"
		}, []string{}, true},
		{"loadBalancer", "loadBalancer", func(s *corev1.Service) { s.Spec.Type = corev1.ServiceTypeLoadBalancer }, []string{"192.0.2.30", "lb.example.com"}, false},
		{"loadBalancer externalIPs", "externalIPs", func(s *corev1.Service) {
			s.Spec.Type = corev1.ServiceTypeLoadBalancer
			s.Spec.ExternalIPs = []string{"192.0.2.20"}
		}, []string{"192.0.2.20"}, false},
		{"no ClusterIP inference", "", func(s *corev1.Service) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.ExternalIPs = []string{"192.0.2.20"}
		}, []string{}, false},
		{"no headless inference", "", func(s *corev1.Service) { s.Spec.Type = corev1.ServiceTypeClusterIP; s.Spec.ClusterIP = "None" }, []string{}, false},
		{"no NodePort inference", "", func(s *corev1.Service) { s.Spec.Type = corev1.ServiceTypeNodePort }, []string{}, false},
		{"partially invalid fields", "externalIPs", func(s *corev1.Service) { s.Spec.ExternalIPs = []string{"bad://ip", "192.0.2.10"} }, []string{"192.0.2.10"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := serviceFixture()
			s.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: "192.0.2.30"}, {Hostname: "LB.Example.com."}}
			s.Annotations[serviceScheme] = "https"
			tc.change(&s)
			e := serviceEntry(s)
			if e.DNS.TargetSource != tc.source || !reflect.DeepEqual(e.DNS.Targets, tc.targets) || (len(e.Issues) > 0) != tc.issue || e.ExternalName != s.Spec.ExternalName || len(e.Endpoints) != 1 || e.Endpoints[0].URL != "https://app.example.com" {
				t.Fatalf("entry=%+v dns=%+v", e, e.DNS)
			}
		})
	}
}

func TestServiceWebIntent(t *testing.T) {
	for _, tc := range []struct {
		name        string
		annotations map[string]string
		url         string
		issue       bool
	}{
		{"DNS only", nil, "", false},
		{"HTTPS default", map[string]string{serviceScheme: " https "}, "https://app.example.com", false},
		{"HTTP default", map[string]string{serviceScheme: "http", servicePort: " 080 "}, "http://app.example.com", false},
		{"custom", map[string]string{serviceScheme: "https", servicePort: "8443"}, "https://app.example.com:8443", false},
		{"port only", map[string]string{servicePort: "443"}, "", true},
		{"empty scheme", map[string]string{serviceScheme: ""}, "", true},
		{"unsafe scheme", map[string]string{serviceScheme: "javascript"}, "", true},
		{"scheme case", map[string]string{serviceScheme: "HTTPS"}, "", true},
		{"empty port", map[string]string{serviceScheme: "https", servicePort: ""}, "", true},
		{"zero port", map[string]string{serviceScheme: "https", servicePort: "0"}, "", true},
		{"high port", map[string]string{serviceScheme: "https", servicePort: "65536"}, "", true},
		{"sign port", map[string]string{serviceScheme: "https", servicePort: "+443"}, "", true},
		{"decimal port", map[string]string{serviceScheme: "https", servicePort: "443.0"}, "", true},
		{"text port", map[string]string{serviceScheme: "https", servicePort: "https"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := serviceFixture()
			s.Spec.Ports = []corev1.ServicePort{{Name: "https", Port: 443}}
			s.Annotations["k8sautodocs.io/fqdn-override"] = "ignored.example.com"
			for k, v := range tc.annotations {
				s.Annotations[k] = v
			}
			e := serviceEntry(s)
			if (len(e.Issues) > 0) != tc.issue || (len(e.Endpoints) > 0) != (tc.url != "") || tc.url != "" && e.Endpoints[0].URL != tc.url {
				t.Fatalf("endpoints=%+v issues=%v", e.Endpoints, e.Issues)
			}
		})
	}
	s := serviceFixture()
	s.Annotations[serviceScheme] = "https"
	s.Annotations[externalDNSAlpha+"hostname"] = "b.example.com,a.example.com,a.example.com,*.example.com"
	e := serviceEntry(s)
	if len(e.Endpoints) != 2 || len(e.Hostnames) != 3 || len(e.Issues) != 0 {
		t.Fatalf("multiple hosts: %+v", e)
	}
	s.Annotations[externalDNSAlpha+"hostname"] = "*.example.com"
	e = serviceEntry(s)
	if len(e.Endpoints) != 0 || len(e.Issues) == 0 {
		t.Fatalf("wildcard: %+v", e)
	}
	s.Annotations[externalDNSStable+"hostname"] = "a.example.com"
	e = serviceEntry(s)
	if len(e.Endpoints) != 0 || len(e.Hostnames) != 0 || len(e.Issues) == 0 {
		t.Fatalf("hostname conflict: %+v", e)
	}
}

func TestServiceLegacyCollection(t *testing.T) {
	s := serviceFixture()
	s.Labels = nil
	kube := fake.NewSimpleClientset(&s)
	entries, warnings, err := collectAutoDocsView(context.Background(), kube, dynamicFixture(t), "", discoveredView)
	if err != nil || len(warnings) != 0 || len(entries) != 1 || entries[0].Kind != "Service" {
		t.Fatalf("entries=%+v warnings=%v err=%v", entries, warnings, err)
	}
	for _, action := range kube.Actions() {
		if action.GetResource().Resource != "ingresses" && action.GetResource().Resource != "services" {
			t.Fatalf("standalone Service made unrelated read: %v", action)
		}
	}
}
