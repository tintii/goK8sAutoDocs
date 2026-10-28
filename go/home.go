package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type homeLink struct {
	Label string `json:"label" yaml:"label"`
	URL   string `json:"url" yaml:"url"`
}

type homeConfig struct {
	Title       string     `json:"title" yaml:"title"`
	Description string     `json:"description" yaml:"description"`
	Environment string     `json:"environment" yaml:"environment"`
	MOTD        string     `json:"motd" yaml:"motd"`
	Links       []homeLink `json:"links" yaml:"links"`
	Footer      string     `json:"footer" yaml:"footer"`
}

func defaultHomeConfig() homeConfig {
	return homeConfig{
		Title: "Cluster overview", Description: "Application links and container versions.",
		Environment: "Kubernetes", Links: []homeLink{}, Footer: "AutoDocs · Kubernetes resource directory",
	}
}

func safeHomeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || strings.ContainsAny(raw, "\\\r\n\t") || u.User != nil {
		return false
	}
	return ((u.Scheme == "http" || u.Scheme == "https") && u.Host != "") ||
		(u.Scheme == "" && u.Host == "" && strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//"))
}

// Read the mounted file on each request so Kubernetes' atomic volume updates
// take effect without a restart. Mount the directory, rather than a subPath.
func loadHomeConfig(path string) (homeConfig, error) {
	config := defaultHomeConfig()
	if path == "" {
		return config, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(data) > 65536 {
		return config, fmt.Errorf("home configuration must be at most 64 KiB")
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("decode home configuration: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return config, fmt.Errorf("home configuration must contain one YAML document")
	}
	if strings.TrimSpace(config.Title) == "" || len(config.Links) > 12 {
		return config, fmt.Errorf("home configuration requires a title and at most 12 links")
	}
	for _, link := range config.Links {
		if strings.TrimSpace(link.Label) == "" || !safeHomeURL(link.URL) {
			return config, fmt.Errorf("home links require a label and an http(s) or root-relative URL")
		}
	}
	if config.Links == nil {
		config.Links = []homeLink{}
	}
	return config, nil
}

func homeConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	config, err := loadHomeConfig(os.Getenv("HOMEPAGE_CONFIG"))
	if err != nil {
		http.Error(w, "Home page configuration is unavailable or invalid. Check HOMEPAGE_CONFIG and its YAML fields.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// Application liveness only. Cluster connectivity and workload health are
// separate concerns; neither is implied by a successful response here.
func serviceHealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "running"})
}

type workloadStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Ready     int32  `json:"ready"`
	Desired   int32  `json:"desired"`
	State     string `json:"state"`
}

type serviceStatus struct {
	CheckedAt time.Time        `json:"checkedAt"`
	Ready     int              `json:"ready"`
	Attention int              `json:"attention"`
	Inactive  int              `json:"inactive"`
	Workloads []workloadStatus `json:"workloads"`
}

func serviceStatusHandler(client kubernetes.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		result, err := collectServiceStatus(ctx, client)
		if err != nil {
			// Partial lists must never look like a healthy cluster.
			http.Error(w, "Workload status is unavailable. Check Kubernetes access and read permissions.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func collectServiceStatus(ctx context.Context, client kubernetes.Interface) (serviceStatus, error) {
	result := serviceStatus{Workloads: []workloadStatus{}}
	add := func(name, namespace, kind string, ready, desired int32, observed, generation int64) {
		state := "attention"
		switch {
		case desired == 0:
			state = "inactive"
			result.Inactive++
		case ready >= desired && observed >= generation:
			state = "ready"
			result.Ready++
		default:
			result.Attention++
		}
		result.Workloads = append(result.Workloads, workloadStatus{name, namespace, kind, ready, desired, state})
	}
	deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, d := range deployments.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		add(d.Name, d.Namespace, "Deployment", d.Status.AvailableReplicas, desired, d.Status.ObservedGeneration, d.Generation)
	}
	statefulSets, err := client.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, s := range statefulSets.Items {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		add(s.Name, s.Namespace, "StatefulSet", s.Status.AvailableReplicas, desired, s.Status.ObservedGeneration, s.Generation)
	}
	daemonSets, err := client.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return result, err
	}
	for _, d := range daemonSets.Items {
		add(d.Name, d.Namespace, "DaemonSet", d.Status.NumberAvailable, d.Status.DesiredNumberScheduled, d.Status.ObservedGeneration, d.Generation)
	}
	priority := map[string]int{"attention": 0, "ready": 1, "inactive": 2}
	sort.Slice(result.Workloads, func(i, j int) bool {
		a, b := result.Workloads[i], result.Workloads[j]
		if a.State != b.State {
			return priority[a.State] < priority[b.State]
		}
		return a.Namespace+"/"+a.Name+"/"+a.Kind < b.Namespace+"/"+b.Name+"/"+b.Kind
	})
	result.CheckedAt = time.Now().UTC()
	return result, nil
}
