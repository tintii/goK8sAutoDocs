package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// "time"

	// "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Annotations to filter for
var targetAnnotations = []string{
	"k8sautodocs.io/publish",       // boolean annotation for k8sautodocs API to indicate publish status true/false
	"k8sautodocs.io/fqdn-override", // boolean annotation for k8sautodocs API to indicate publish status true/false
}

// Labels to filter for
var targetLabels = []string{}

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
	// {
	// 	Group:    "getambassador.io",
	// 	Version:  "v2",
	// 	Resource: "mappings",
	// },
}

var autoDocsResourcesGVR = []GVRResource{
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

// Global clientset
var clientset *kubernetes.Clientset

// Global dynamic client
var dynamicClient *dynamic.DynamicClient

// Global cluster name
var clusterName string

// Debug mode flag
var debugMode = false

// Cache variables
var cacheDuration = 60 * time.Second

var cachedJSONData []byte
var lastCacheTime time.Time

var cachedJSONDataVersionsDashboard []byte
var lastCacheTimeVersionsDashboard time.Time

var cachedJSONDataAutoDocsDashboard []byte
var lastCacheTimeAutoDocsDashboard time.Time

var cachedJSONDataClusterInfo []byte
var lastCacheTimeClusterInfo time.Time

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func connectToK8sInCluster() { //use when running in a pod
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	handleError(err)
	log.Println("func main: config.Host", config.Host)
	// note the clientset is for the default builtin kubernetes constructs and apis. you need a dynamic client for non-default CRDs.
	clientset, err = kubernetes.NewForConfig(config)
	handleError(err)

	dynamicClient, err = dynamic.NewForConfig(config)
	handleError(err)
}

func connectToK8sOutCluster() { //use for running locally (will use your current kube context)
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	log.Println("func main: using kubeconfig", kubeconfig)

	// build k8s config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	handleError(err)
	log.Println("func main: config.Host", config.Host)

	// note the clientset is for the default builtin kubernetes constructs and apis. you need a dynamic client for non-default CRDs.
	clientset, err = kubernetes.NewForConfig(config)
	handleError(err)

	// Initialize dynamic client
	dynamicClient, err = dynamic.NewForConfig(config)
	handleError(err)
}

// HTTP handler for serving cluster information
func clusterInfoHandler(w http.ResponseWriter, r *http.Request) {
	for key, val := range headerMap { //set common headers
		w.Header().Set(key, val)
	}
	// Check if cache is still valid
	if time.Since(lastCacheTime) < cacheDuration && len(cachedJSONDataClusterInfo) > 0 {
		if debugMode {
			log.Printf("Returning cached data (age: %v)", time.Since(lastCacheTimeClusterInfo))
		}
		w.Write(cachedJSONDataClusterInfo)
		return
	}
	// Get cluster info from kubeconfig context
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	// Load kubeconfig to get current context and cluster name
	configLoadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configLoadingRules.ExplicitPath = kubeconfig
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, &clientcmd.ConfigOverrides{})

	// Get current context and cluster name
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		clusterInfo := map[string]interface{}{
			"clusterName": "unknown",
			"context":     "unknown",
			"server":      "unknown",
			// "timestamp":   time.Now().Format(time.RFC3339),
			"error": err.Error(),
		}
		cachedJSONDataClusterInfo, _ := json.MarshalIndent(clusterInfo, "", "  ")
		w.Write(cachedJSONDataClusterInfo)
		return
	}

	currentContext := rawConfig.CurrentContext
	context := rawConfig.Contexts[currentContext]
	clusterName := context.Cluster
	cluster := rawConfig.Clusters[clusterName]

	clusterInfo := map[string]interface{}{
		"clusterName": clusterName,
		"context":     currentContext,
		"server":      cluster.Server,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	cachedJSONDataClusterInfo, err := json.MarshalIndent(clusterInfo, "", "  ")
	if err != nil {
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}

	w.Write(cachedJSONDataClusterInfo)
}

// this func will get the resources needed for the versions dashboard using clientset
func getVersionsDashboardResourcesWithClientset(clientset *kubernetes.Clientset, namespace string) ([]map[string]interface{}, error) {
	if debugMode {
		log.Println("DEBUG: func getVersionsDashboardResourcesWithClientset")
	}

	var jsonPayload []map[string]interface{}

	// Get Deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting deployments: %v", err)
	} else {
		log.Printf("Found %d deployments", len(deployments.Items))
		for _, deployment := range deployments.Items {
			relatedResources, err := findRelatedDeploymentResources(clientset, deployment) //get the pods
			handleError(err)
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
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting statefulsets: %v", err)
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
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting daemonsets: %v", err)
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

func findRelatedDeploymentResources(clientset *kubernetes.Clientset, deployment appsv1.Deployment) ([]RelatedResource, error) {
	var relatedResources []RelatedResource
	// Get pods for the deployment by matching the deployment's selector
	selector := deployment.Spec.Selector
	if selector != nil && len(selector.MatchLabels) > 0 {
		podList, err := clientset.CoreV1().Pods(deployment.Namespace).List(context.TODO(), metav1.ListOptions{
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

// HTTP handler for serving versions dashboard json payload - surely there's a better way to do this all, but whatever.
// This bit is to wrap the handler function so that the targetResourcesGVR can be dynamically passed in
func versionsDashboardHandler(w http.ResponseWriter, r *http.Request) {
	for key, val := range headerMap { //set common headers
		w.Header().Set(key, val)
	}

	// Check if cache is still valid
	if time.Since(lastCacheTimeVersionsDashboard) < cacheDuration && len(cachedJSONDataVersionsDashboard) > 0 {
		if debugMode {
			log.Printf("Returning cached versions dashboard data (age: %v)", time.Since(lastCacheTimeVersionsDashboard))
		}
		w.Write(cachedJSONDataVersionsDashboard)
		return
	}

	// Get latest data for each request using clientset
	jsonPayload, err := getVersionsDashboardResourcesWithClientset(clientset, "npe01-headless-jackpots") //namspace filter needed
	if err != nil {
		http.Error(w, "Error getting resources", http.StatusInternalServerError)
		return
	}

	jsonData, err := json.MarshalIndent(jsonPayload, "", "  ")
	if err != nil {
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}

	// Update cache
	cachedJSONDataVersionsDashboard = jsonData
	lastCacheTimeVersionsDashboard = time.Now()

	// Print pretty JSON only in debug mode
	if debugMode {
		fmt.Printf("Versions Dashboard JSON Payload:\n%s\n", string(jsonData))
		log.Printf("Versions dashboard cache updated at %v", lastCacheTimeVersionsDashboard)
	}

	w.Write(jsonData)
}

// this func will get the resources and filter by annotation/label
func getFilteredResourcesWithDynamicClient(dynamicClient *dynamic.DynamicClient, namespace string, targetGVR []GVRResource) ([]map[string]interface{}, error) {
	if debugMode {
		log.Println("DEBUG: func getResources")
	}

	var jsonPayload []map[string]interface{}

	// for _, resource := range targetResources {
	for _, resource := range targetGVR {
		gvr := schema.GroupVersionResource{
			Group:    resource.Group,
			Version:  resource.Version,
			Resource: resource.Resource,
		}
		log.Printf("Getting: %s (Group: %s, Version: %s)", gvr.Resource, gvr.Group, gvr.Version)
		resources, err := dynamicClient.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Error getting %s: %v", resource.Resource, err)
			return jsonPayload, err
		}

		log.Printf("Found %d %s resources", len(resources.Items), gvr.Resource)

		for _, resource := range resources.Items {
			// Get labels from the unstructured resource
			labels := resource.GetLabels()
			filteredLabels := make(map[string]string)
			for _, targetKey := range targetLabels {
				if value, exists := labels[targetKey]; exists {
					filteredLabels[targetKey] = value
				}
			}

			// Get annotations from the unstructured resource
			annotations := resource.GetAnnotations()
			filteredAnnotations := make(map[string]string)
			for _, targetKey := range targetAnnotations {
				if value, exists := annotations[targetKey]; exists {
					filteredAnnotations[targetKey] = value
				}
			}

			// Store filtered data in JSON format
			if len(filteredAnnotations) > 0 || len(filteredLabels) > 0 {
				resourceData := map[string]interface{}{
					"name":        resource.GetName(),
					"namespace":   resource.GetNamespace(),
					"kind":        resource.GetKind(),
					"labels":      filteredLabels,
					"annotations": filteredAnnotations,
				}
				// Append to local slice
				jsonPayload = append(jsonPayload, resourceData)
				if debugMode {
					log.Printf("Added %s/%s to payload", resource.GetNamespace(), resource.GetName())
				}
			} else {
				if debugMode {
					log.Printf("Skipped %s/%s (no matching labels/annotations)", resource.GetNamespace(), resource.GetName())
				}
			}
		}
	}

	return jsonPayload, nil
}

// HTTP handler for serving resources json payload
// This bit is to wrap the handler function so that the targetResourcesGVR can be dynamically passed in
func filteredResourcesHandler(targetResourcesGVR []GVRResource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for key, val := range headerMap { //set common headers
			w.Header().Set(key, val)
		}
		// Check if cache is still valid
		if time.Since(lastCacheTime) < cacheDuration && len(cachedJSONData) > 0 {
			if debugMode {
				log.Printf("Returning cached data (age: %v)", time.Since(lastCacheTime))
			}
			w.Write(cachedJSONData)
			return
		}

		// Get latest data for each request
		jsonPayload, err := getFilteredResourcesWithDynamicClient(dynamicClient, "", targetResourcesGVR)
		if err != nil {
			http.Error(w, "Error getting resources", http.StatusInternalServerError)
			// return
		}

		jsonData, err := json.MarshalIndent(jsonPayload, "", "  ")
		if err != nil {
			http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
			return
		}

		// Update cache
		cachedJSONData = jsonData
		lastCacheTime = time.Now()

		// Print pretty JSON only in debug mode
		if debugMode {
			fmt.Printf("JSON Payload:\n%s\n", string(jsonData))
			log.Printf("Cache updated at %v", lastCacheTime)
		}

		w.Write(jsonData)
	}
}

// Extract hosts from ingress spec
// extractHostsFromIngressSpec extracts hostnames from the ingress spec rules.
func extractHostsFromIngressSpec(rules []networkingv1.IngressRule) []map[string]interface{} {
	var hosts []map[string]interface{}
	for _, rule := range rules {
		if rule.Host != "" {
			hostInfo := map[string]interface{}{
				"host": rule.Host,
			}
			hosts = append(hosts, hostInfo)
		}
	}
	return hosts
}

// findRelatedIngressResources finds resources related to an ingress
func findRelatedIngressResources(clientset *kubernetes.Clientset, ingress networkingv1.Ingress) ([]RelatedResource, error) {
	var relatedResources []RelatedResource

	// Extract service names from ingress rules
	serviceNames := make(map[string]bool)
	for _, rule := range ingress.Spec.Rules {
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil {
				serviceNames[path.Backend.Service.Name] = true
			}
		}
	}

	// Find related services
	for serviceName := range serviceNames {
		service, err := clientset.CoreV1().Services(ingress.Namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
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
		if service.Spec.Selector != nil {
			pods, err := clientset.CoreV1().Pods(ingress.Namespace).List(context.TODO(), metav1.ListOptions{
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
				imageName, imageTag, imageSha := parseImage(pod.Status.ContainerStatuses[0].ImageID)
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
	deployments, err := clientset.AppsV1().Deployments(ingress.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, deployment := range deployments.Items {
			// Check if deployment labels match any of the service selectors
			for _, resource := range relatedResources {
				if resource.Kind == "Service" && resource.Selector != nil {
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
	configmaps, err := clientset.CoreV1().ConfigMaps(ingress.Namespace).List(context.TODO(), metav1.ListOptions{})
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

func getAutoDocsResourcesWithClientset(clientset *kubernetes.Clientset, namespace string) ([]map[string]interface{}, error) {
	if debugMode {
		log.Println("DEBUG: func getAutoDocsResourcesWithClientset")
	}

	var jsonPayload []map[string]interface{}

	// Get Ingresses
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Error getting ingresses: %v", err)
	} else {
		if debugMode {
			log.Printf("Found %d ingresses", len(ingresses.Items))
		}
		for _, ingress := range ingresses.Items {
			annotations := ingress.Annotations
			for _, targetKey := range targetAnnotations {
				if _, exists := annotations[targetKey]; exists {
					ingressHosts := extractHostsFromIngressSpec(ingress.Spec.Rules)

					// Find related resources
					relatedResources, err := findRelatedIngressResources(clientset, ingress)
					if err != nil && debugMode {
						log.Printf("Error finding related resources for ingress %s: %v", ingress.Name, err)
					}

					resourceData := map[string]interface{}{
						"name":             ingress.Name,
						"namespace":        ingress.Namespace,
						"kind":             "ingress",
						"ingress-class":    ingress.Spec.IngressClassName,
						"hostnames":        ingressHosts,
						"labels":           ingress.Labels,
						"annotations":      ingress.Annotations,
						"relatedResources": relatedResources,
					}
					jsonPayload = append(jsonPayload, resourceData)
				}
			}
		}
	}

	return jsonPayload, nil
}

// HTTP handler for serving resources json payload
// This bit is to wrap the handler function so that the targetResourcesGVR can be dynamically passed in
func autoDocsResourcesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for key, val := range headerMap { //set common headers
			w.Header().Set(key, val)
		}

		// Check if cache is still valid
		if time.Since(lastCacheTime) < cacheDuration && len(cachedJSONData) > 0 {
			if debugMode {
				log.Printf("Returning cached data (age: %v)", time.Since(lastCacheTime))
			}
			w.Write(cachedJSONData)
			return
		}

		// Get latest data for each request
		jsonPayload, err := getAutoDocsResourcesWithClientset(clientset, "")
		if err != nil {
			http.Error(w, "Error getting resources", http.StatusInternalServerError)
			// return
		}

		jsonData, err := json.MarshalIndent(jsonPayload, "", "  ")
		if err != nil {
			http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
			return
		}

		// Update cache
		cachedJSONData = jsonData
		lastCacheTime = time.Now()

		// Print pretty JSON only in debug mode
		if debugMode {
			fmt.Printf("JSON Payload:\n%s\n", string(jsonData))
			log.Printf("Cache updated at %v", lastCacheTime)
		}

		w.Write(jsonData)
	}
}

func main() {
	log.SetFlags(3)
	log.Println("func main")

	// Check for debug mode via environment variable
	if os.Getenv("DEBUG") == "true" {
		debugMode = true
		log.Println("Debug mode enabled")
	}

	// Determine if /var/run/secrets/kubernetes.io/serviceaccount exists, if yes then use InCluster, otherwise use OutCluster
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err != nil {
		if os.IsNotExist(err) {
			log.Println("Connecting to k8s using local context")
			connectToK8sOutCluster()
		} else {
			log.Println("Connecting to k8s using service account")
			connectToK8sInCluster()
		}
	}

	// Start web server
	http.HandleFunc("/api/v1/filtered-resources", filteredResourcesHandler(filteredResourcesGVR))
	http.HandleFunc("/api/v1/auto-docs-dashboard", autoDocsResourcesHandler())
	http.HandleFunc("/api/v1/versions-dashboard", versionsDashboardHandler)
	http.HandleFunc("/api/v1/cluster-info", clusterInfoHandler)

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
