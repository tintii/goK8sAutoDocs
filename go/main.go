package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	// "time"

	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
)

// GVR struct for resource definitions
type GVRResource struct {
	Group    string
	Version  string
	Resource string
}

// RelatedResource represents a related Kubernetes resource
type RelatedResource struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Kind      string            `json:"kind"`
	Labels    map[string]string `json:"labels,omitempty"`
	Selector  map[string]string `json:"selector,omitempty"`
	// ImageName holds the name of the container image (excluding tag and digest), and will be serialized to JSON as "ImageName" if it is not empty.
	ImageName string `json:"ImageName,omitempty"`
	ImageTag  string `json:"ImageTag,omitempty"`
	ImageSha  string `json:"ImageSha,omitempty"`
}

var filteredResourcesGVR = []GVRResource{
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	},
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	},
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	},
	{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	},
	{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	},
	{
		Group:    "networking.k8s.io",
		Version:  "v1",
		Resource: "ingresses",
	},
	{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"},
}

var versionsDashboardGVR = []GVRResource{
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	},
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "statefulsets",
	},
	{
		Group:    "apps",
		Version:  "v1",
		Resource: "daemonsets",
	},
}

var headerMap = map[string]string{
	"Content-Type":                 "application/json",
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Headers": "*",
	"Access-Control-Allow-Methods": "OPTIONS,POST,GET",
	"Cache-Control":                "max-age=30",
}

// Legacy handler helpers retain their testable cache policy; the running server
// uses the shared inventory coordinator instead.
var debugMode = false
var cacheDuration = 60 * time.Second

// this func will get the resources needed for the versions dashboard using clientset
func getVersionsDashboardResourcesWithClientset(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]map[string]interface{}, error) {
	if debugMode {
		log.Println("DEBUG: func getVersionsDashboardResourcesWithClientset")
	}

	jsonPayload := make([]map[string]interface{}, 0)

	// Get Deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	} else {
		log.Printf("Found %d deployments", len(deployments.Items))
		for _, deployment := range deployments.Items {
			relatedResources, err := findRelatedDeploymentResources(ctx, clientset, deployment) //get the pods
			if err != nil {
				return nil, fmt.Errorf("list pods for deployment %s/%s: %w", deployment.Namespace, deployment.Name, err)
			}
			containers := extractContainersFromPodSpec(&deployment.Spec.Template.Spec)
			if debugMode {
				log.Print("DEBUG: relatedResources: ", relatedResources)
				log.Print("DEBUG: containers: ", containers)
			}
			// Inject imageSha from related resources (pods) into containers
			for _, container := range containers { //loop through containers
				for _, relatedResourceContainers := range relatedResources { //loop through all pods and their respective containers to get the sha
					// Match by container name first, then try to match by image name
					if container["name"] == relatedResourceContainers.Name {
						if debugMode {
							log.Printf("DEBUG: Matched by container name: %s", container["name"])
						}
						container["sha"] = relatedResourceContainers.ImageSha
						break
					}
					// Fallback: try to match by image name (handle short vs full registry path)
					deploymentImage := container["image"].(string)
					podImageName := relatedResourceContainers.ImageName
					if debugMode {
						log.Printf("DEBUG: Comparing deployment image '%s' with pod image '%s'", deploymentImage, podImageName)
					}
					if deploymentImage == podImageName ||
						strings.HasSuffix(podImageName, "/"+deploymentImage) ||
						strings.HasSuffix(podImageName, ":"+deploymentImage) {
						if debugMode {
							log.Printf("DEBUG: Matched by image name: %s", deploymentImage)
						}
						container["sha"] = relatedResourceContainers.ImageSha
						break
					}
				}
			}

			resourceData := map[string]interface{}{
				"name":             deployment.Name,
				"namespace":        deployment.Namespace,
				"kind":             "Deployment",
				"containers":       containers,
				"labels":           deployment.Labels,
				"relatedResources": relatedResources,
			}
			jsonPayload = append(jsonPayload, resourceData)
		}
	}

	// Get StatefulSets
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list statefulsets: %w", err)
	} else {
		log.Printf("Found %d statefulsets", len(statefulsets.Items))
		for _, statefulset := range statefulsets.Items {
			containers := extractContainersFromPodSpec(&statefulset.Spec.Template.Spec)
			resourceData := map[string]interface{}{
				"name":       statefulset.Name,
				"namespace":  statefulset.Namespace,
				"kind":       "StatefulSet",
				"containers": containers,
				"labels":     statefulset.Labels,
			}
			jsonPayload = append(jsonPayload, resourceData)
		}
	}

	// Get DaemonSets
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list daemonsets: %w", err)
	} else {
		log.Printf("Found %d daemonsets", len(daemonsets.Items))
		for _, daemonset := range daemonsets.Items {
			containers := extractContainersFromPodSpec(&daemonset.Spec.Template.Spec)
			resourceData := map[string]interface{}{
				"name":       daemonset.Name,
				"namespace":  daemonset.Namespace,
				"kind":       "DaemonSet",
				"containers": containers,
				"labels":     daemonset.Labels,
			}
			jsonPayload = append(jsonPayload, resourceData)
		}
	}

	return jsonPayload, nil
}

// Extract container information from PodSpec
func extractContainersFromPodSpec(podSpec *corev1.PodSpec) []map[string]interface{} {
	var containers []map[string]interface{}

	// Main containers

	for _, container := range podSpec.Containers {
		imageName, imageTag, imageSha := parseImage(container.Image)
		containerInfo := map[string]interface{}{
			"name":  container.Name,
			"image": imageName,
			"tag":   imageTag,
			"sha":   imageSha,
			"type":  "main",
		}
		containers = append(containers, containerInfo)
	}

	// Init containers
	for _, container := range podSpec.InitContainers {
		imageName, imageTag, imageSha := parseImage(container.Image)
		containerInfo := map[string]interface{}{
			"name":  container.Name,
			"image": imageName,
			"tag":   imageTag,
			"sha":   imageSha,
			"type":  "init",
		}
		containers = append(containers, containerInfo)
	}

	return containers
}

// Parse image string to separate name and tag
func parseImage(image string) (imageName string, imageTag string, imageSha string) {
	// Handle images with/without tags, with/without sha
	if debugMode {
		log.Printf("DEBUG: parseImage: %s\n", image)
	}
	splitInto := strings.Count(image, ":") //count number of : for split

	parts := strings.SplitN(image, ":", splitInto+1)
	switch splitInto {
	case 0:
		imageName = parts[0]
		imageTag = "latest"
		imageSha = ""
	case 1:
		if strings.Contains(parts[0], "@sha256") { //contains imagename and sha256 only
			imageName = strings.Replace(parts[0], "@sha256", "", 1)
			imageTag = "latest"
			imageSha = parts[1]
		} else { //contains imagename and image tag only
			imageName = parts[0]
			imageTag = parts[1]
			imageSha = ""
		}
	case 2: //contains imagename imagetag sha256
		imageName = parts[0]
		imageTag = strings.Replace(parts[1], "@sha256", "", 1)
		imageSha = parts[2]
	}

	return imageName, imageTag, imageSha
}

func findRelatedDeploymentResources(ctx context.Context, clientset kubernetes.Interface, deployment appsv1.Deployment) ([]RelatedResource, error) {
	var relatedResources []RelatedResource
	// Get pods for the deployment by matching the deployment's selector
	selector := deployment.Spec.Selector
	if selector != nil && len(selector.MatchLabels) > 0 {
		podList, err := clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		})
		if err != nil {
			log.Printf("Error getting pods for deployment %s: %v", deployment.Name, err)
			return relatedResources, err
		}
		// get pods
		if debugMode {
			log.Print("DEBUG: podList.Items", podList.Items)
		}
		for _, pod := range podList.Items {
			for _, container := range pod.Status.ContainerStatuses {
				imageName, imageTag, imageSha := parseImage(container.ImageID)
				relatedResources = append(relatedResources, RelatedResource{
					Name:      container.Name,
					Namespace: pod.Namespace,
					Kind:      "Container",
					Labels:    pod.Labels,
					ImageName: imageName,
					ImageTag:  imageTag,
					ImageSha:  imageSha,
				})
			}
		}
	}
	return relatedResources, nil
}

// The versions inventory covers every namespace and uses the same synchronized
// cache handler as the directory APIs. Failed collection is never cached.
func versionsDashboardHandler(client kubernetes.Interface) http.HandlerFunc {
	return cachedResourceHandler(cacheDuration, func(ctx context.Context) (any, []string, error) {
		resources, err := getVersionsDashboardResourcesWithClientset(ctx, client, metav1.NamespaceAll)
		return resources, nil, err
	})
}

// findRelatedIngressResources finds resources related to an ingress
func findRelatedIngressResources(ctx context.Context, clientset kubernetes.Interface, ingress networkingv1.Ingress) ([]RelatedResource, error) {
	var relatedResources []RelatedResource

	// Extract service names from ingress rules
	serviceNames := make(map[string]bool)
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil {
				serviceNames[path.Backend.Service.Name] = true
			}
		}
	}

	// Find related services
	for serviceName := range serviceNames {
		service, err := clientset.CoreV1().Services(ingress.Namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			if debugMode {
				log.Printf("Error getting service %s: %v", serviceName, err)
			}
			continue
		}

		relatedResources = append(relatedResources, RelatedResource{
			Name:      service.Name,
			Namespace: service.Namespace,
			Kind:      "Service",
			Labels:    service.Labels,
			Selector:  service.Spec.Selector,
		})

		// Find pods that match the service selector
		if len(service.Spec.Selector) > 0 {
			pods, err := clientset.CoreV1().Pods(ingress.Namespace).List(ctx, metav1.ListOptions{
				LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
					MatchLabels: service.Spec.Selector,
				}),
			})
			if err != nil {
				if debugMode {
					log.Printf("Error getting pods for service %s: %v", serviceName, err)
				}
				continue
			}

			for _, pod := range pods.Items {
				var imageName, imageTag, imageSha string
				if len(pod.Status.ContainerStatuses) > 0 {
					imageName, imageTag, imageSha = parseImage(pod.Status.ContainerStatuses[0].ImageID)
				}
				relatedResources = append(relatedResources, RelatedResource{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Kind:      "Pod",
					Labels:    pod.Labels,
					ImageName: imageName,
					ImageTag:  imageTag,
					ImageSha:  imageSha,
				})
			}
		}
	}

	// Find related deployments that might be managing the pods
	deployments, err := clientset.AppsV1().Deployments(ingress.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deployment := range deployments.Items {
			// Check if deployment labels match any of the service selectors
			for _, resource := range relatedResources {
				if resource.Kind == "Service" && len(resource.Selector) > 0 {
					if labelsMatch(resource.Selector, deployment.Labels) {
						relatedResources = append(relatedResources, RelatedResource{
							Name:      deployment.Name,
							Namespace: deployment.Namespace,
							Kind:      "Deployment",
							Labels:    deployment.Labels,
						})
					}
				}
			}
		}
	}

	// Find related configmaps and secrets that might be mounted
	configmaps, err := clientset.CoreV1().ConfigMaps(ingress.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, configmap := range configmaps.Items {
			// Check if configmap is referenced by any related pods
			for _, resource := range relatedResources {
				if resource.Kind == "Pod" && resource.Name == configmap.Name {
					relatedResources = append(relatedResources, RelatedResource{
						Name:      configmap.Name,
						Namespace: configmap.Namespace,
						Kind:      "ConfigMap",
						Labels:    configmap.Labels,
					})
				}
			}
		}
	}

	return relatedResources, nil
}

// labelsMatch checks if selector labels match resource labels
func labelsMatch(selector, labels map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}

func main() {
	log.SetFlags(3)
	log.Println("func main")

	// Check for debug mode via environment variable
	if os.Getenv("DEBUG") == "true" {
		debugMode = true
		log.Println("Debug mode enabled")
	}

	config, err := loadInventoryConfig(os.Getenv("AUTODOCS_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}
	inventory := newInventoryService(config)
	inventory.start(context.Background())
	http.HandleFunc("/api/v1/filtered-resources", inventory.dataHandler("filtered"))
	http.HandleFunc("/api/v1/auto-docs-dashboard", inventory.dataHandler("directory"))
	http.HandleFunc("/api/v1/versions-dashboard", inventory.dataHandler("workloads"))
	http.HandleFunc("/api/v1/service-status", inventory.dataHandler("workloads"))
	http.HandleFunc("/api/v1/cluster-info", inventory.clusterHandler)
	http.HandleFunc("/api/v1/discovery-status", inventory.statusHandler)
	http.HandleFunc("/api/v1/discovery-refresh", inventory.refreshHandler)
	http.HandleFunc("/api/v1/home", homeConfigHandler)
	http.HandleFunc("/api/v1/health", serviceHealthHandler)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve the versions dashboard HTML file
	http.HandleFunc("/versions-dashboard-page", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/versions-dashboard.html")
	})

	// Serve the auto-docs dashboard HTML file
	http.HandleFunc("/autodocs-dashboard-page", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/autodocs-dashboard.html")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	log.Println("Starting web server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
