# goK8sAutoDocs

A Kubernetes resource monitoring and documentation tool that provides consolidated access to cluster resources, versions dashboard, and auto-documentation capabilities.

## Features

- **Versions Dashboard**: Monitor deployment, statefulset, and daemonset versions across namespaces
- **Auto-Docs Dashboard**: Filter resources by specific annotations and labels for documentation purposes
- **Resource Filtering**: Dynamic filtering of Kubernetes resources by GVR (Group, Version, Resource)
- **Cluster Information**: Display current cluster context and connection details
- **Caching**: Built-in caching for improved performance
- **Flexible Authentication**: Supports both in-cluster and out-of-cluster Kubernetes connections

## Prerequisites

- Go 1.19 or higher
- Kubernetes cluster access (via kubeconfig or service account)
- Access to the following Kubernetes resources:
  - Deployments, StatefulSets, DaemonSets
  - Services, ConfigMaps
  - Ingresses
  - Pods

## Installation

### From Source

```bash
git clone https://github.com/yourusername/goK8sAutoDocs.git
cd goK8sAutoDocs/go
go mod tidy
go build -o goK8sAutoDocs main.go
```

## Configuration

### Environment Variables

- `DEBUG=true` - Enable debug logging
- `PORT=8080` - Server port (default: 8080)

### Kubernetes Connection

The application automatically detects the connection method:

- **In-Cluster**: When running inside a Kubernetes pod, uses service account authentication
- **Out-of-Cluster**: When running locally, uses your current kubeconfig context

## Usage

### Running the Application

Before running, if using your local k8s context make sure you set it up `kubectl config use-context <context-name>`

```bash
# Run locally
./goK8sAutoDocs

# Run with debug mode
DEBUG=true ./goK8sAutoDocs

# Run in Kubernetes
kubectl apply -f deployment.yaml
```

### Accessing the Dashboards

- **Main Dashboard**: http://localhost:8080/
- **Versions Dashboard**: http://localhost:8080/versions-dashboard-page
- **Auto-Docs Dashboard**: http://localhost:8080/autodocs-dashboard-page

## API Endpoints

### GET /api/v1/cluster-info
Returns current cluster information including context and server details.

### GET /api/v1/versions-dashboard
Returns version information for deployments, statefulsets, and daemonsets.

### GET /api/v1/auto-docs-dashboard
Returns resources filtered by target annotations and labels.

### GET /api/v1/filtered-resources
Returns resources filtered by specific GVR definitions.

## Resource Types Supported

### Versions Dashboard
- Deployments
- StatefulSets
- DaemonSets

### Auto-Docs Dashboard
- Ingresses
- Services
- ConfigMaps
- Secrets

### Filtered Resources
- Deployments
- StatefulSets
- DaemonSets
- ConfigMaps
- Services
- Ingresses

## Annotations and Labels

The application filters resources based on these target annotations:
- `k8sautodocs.io/publish` - Boolean flag for publishing status
- `k8sautodocs.io/fqdn-override` - FQDN override configuration

## Caching

- **Cache Duration**: 60 seconds (configurable)
- **Cached Endpoints**: All API endpoints support caching for improved performance
- **Cache Invalidation**: Automatic cache refresh on expiration

## Development

### Project Structure

```
goK8sAutoDocs/
├── go/
│   ├── main.go          # Main application
│   ├── go.mod           # Go module file
│   └── static/          # Static HTML files
└── README.md
```

### Building

```bash
cd go
go build -o goK8sAutoDocs main.go
```

### Testing

```bash
go test ./...
```

## Security Considerations

- **CORS**: Currently allows all origins (`*`) - consider restricting in production
- **Authentication**: No built-in authentication - implement if exposing publicly
- **Namespace Filtering**: Some endpoints use hardcoded namespaces - make configurable for production

## Troubleshooting

### Common Issues

1. **Connection Errors**: Verify kubeconfig or service account permissions
2. **Resource Access**: Ensure RBAC permissions for target resources
3. **Port Conflicts**: Check if port 8080 is available

### Debug Mode

Enable debug logging to see detailed information:

```bash
DEBUG=true ./goK8sAutoDocs
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

**GNU GPL v3.0**

The GNU General Public License is a free, copyleft license for software and other kinds of works. It ensures that the software remains free and open source, requiring that any derivative works also be licensed under the GPL.

---

## Changelog

### [Unreleased]
- Initial development version

### [v0.1.0] - 2024-01-XX
#### Added
- Initial release of goK8sAutoDocs
- Versions dashboard for monitoring deployment versions
- Auto-docs dashboard for resource filtering
- Kubernetes resource discovery and filtering
- Caching system for improved performance
- Support for in-cluster and out-of-cluster connections
- RESTful API endpoints for all dashboards
- Static HTML dashboard pages
- Resource relationship mapping (pods, services, ingresses)
- Image parsing and SHA extraction
- Configurable resource filtering by GVR

#### Technical Features
- Go-based Kubernetes client implementation
- Dynamic client support for custom resources
- Structured logging with debug mode
- CORS support for web dashboard access
- Graceful error handling and fallbacks
- Modular handler architecture
- Configurable cache durations

#### Supported Resources
- Deployments, StatefulSets, DaemonSets
- Services, ConfigMaps, Ingresses
- Pods and container information
- Image metadata extraction
