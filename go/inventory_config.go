package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type targetConfig struct {
	ID         string            `yaml:"id" json:"id"`
	Name       string            `yaml:"name" json:"name"`
	Labels     map[string]string `yaml:"labels" json:"labels,omitempty"`
	InCluster  bool              `yaml:"inCluster" json:"inCluster"`
	Kubeconfig *kubeconfigSource `yaml:"kubeconfig" json:"kubeconfig,omitempty"`
}
type kubeconfigSource struct {
	Path    string `yaml:"path" json:"path"`
	Context string `yaml:"context" json:"context"`
}
type classificationRule struct {
	LabelKeys     []string `yaml:"labelKeys" json:"labelKeys"`
	Sources       []string `yaml:"sources" json:"sources"`
	CaseSensitive bool     `yaml:"caseSensitive" json:"caseSensitive"`
	DisplayCase   string   `yaml:"displayCase" json:"displayCase"`
}
type inventoryConfig struct {
	Clusters        []targetConfig                `yaml:"clusters"`
	LegacyClusterID string                        `yaml:"legacyClusterID"`
	Classification  map[string]classificationRule `yaml:"classification"`
	Refresh         struct {
		Interval       string `yaml:"interval"`
		Cooldown       string `yaml:"cooldown"`
		ClusterTimeout string `yaml:"clusterTimeout"`
	} `yaml:"refresh"`
	Cache struct {
		Path string `yaml:"path"`
	} `yaml:"cache"`
}

func defaultInventoryConfig() inventoryConfig {
	c := inventoryConfig{Classification: map[string]classificationRule{}}
	c.Refresh.Interval, c.Refresh.Cooldown, c.Refresh.ClusterTimeout = "5m", "30s", "30s"
	return c
}
func boundedFile(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errors.New("file exceeds size limit")
	}
	return b, nil
}
func loadInventoryConfig(path string) (inventoryConfig, error) {
	c := defaultInventoryConfig()
	if path == "" {
		t := targetConfig{ID: "local", Name: "Local cluster"}
		if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
			t.InCluster = true
		} else {
			home, _ := os.UserHomeDir()
			p := filepath.Join(home, ".kube", "config")
			b, err := boundedFile(p, 4<<20)
			if err != nil {
				return c, errors.New("default kubeconfig unavailable")
			}
			raw, err := clientcmd.Load(b)
			if err != nil {
				return c, errors.New("default kubeconfig invalid")
			}
			t.Kubeconfig = &kubeconfigSource{Path: p, Context: raw.CurrentContext}
		}
		c.Clusters = []targetConfig{t}
		c.LegacyClusterID = "local"
	} else {
		b, err := boundedFile(path, 1<<20)
		if err != nil {
			return c, errors.New("AutoDocs configuration unavailable")
		}
		d := yaml.NewDecoder(bytes.NewReader(b))
		d.KnownFields(true)
		if err := d.Decode(&c); err != nil {
			return c, errors.New("invalid AutoDocs configuration schema")
		}
		var extra any
		if err := d.Decode(&extra); err != io.EOF {
			return c, errors.New("configuration requires one YAML document")
		}
	}
	return c, c.validate()
}
func (c *inventoryConfig) validate() error {
	if len(c.Clusters) == 0 || len(c.Clusters) > 100 {
		return errors.New("configure between 1 and 100 cluster targets")
	}
	ids := map[string]bool{}
	for i := range c.Clusters {
		t := &c.Clusters[i]
		if strings.TrimSpace(t.ID) == "" || t.ID != strings.TrimSpace(t.ID) || ids[t.ID] {
			return errors.New("cluster IDs must be nonempty and unique")
		}
		ids[t.ID] = true
		if t.Name == "" {
			t.Name = t.ID
		}
		if t.InCluster == (t.Kubeconfig != nil) {
			return fmt.Errorf("target %s requires exactly one connection method", t.ID)
		}
		if t.Kubeconfig != nil && (t.Kubeconfig.Path == "" || t.Kubeconfig.Context == "") {
			return errors.New("kubeconfig path and explicit context are required")
		}
	}
	if c.LegacyClusterID != "" && !ids[c.LegacyClusterID] {
		return errors.New("legacyClusterID must identify a configured target")
	}
	for name, r := range c.Classification {
		if name != "environment" && name != "datacentre" {
			return errors.New("unsupported classification dimension")
		}
		if r.DisplayCase == "" {
			r.DisplayCase = "preserve"
		}
		if r.DisplayCase != "preserve" && r.DisplayCase != "upper" && r.DisplayCase != "lower" {
			return errors.New("invalid classification displayCase")
		}
		if len(r.Sources) == 0 {
			r.Sources = []string{"resource", "namespace"}
		}
		for _, s := range r.Sources {
			if s != "resource" && s != "namespace" && s != "cluster" {
				return errors.New("invalid classification source")
			}
		}
		for _, k := range r.LabelKeys {
			if k == "" {
				return errors.New("classification label keys must be nonempty")
			}
		}
		c.Classification[name] = r
	}
	for _, v := range []string{c.Refresh.Interval, c.Refresh.Cooldown, c.Refresh.ClusterTimeout} {
		d, e := time.ParseDuration(v)
		if e != nil || d < time.Second || d > 24*time.Hour {
			return errors.New("refresh durations must be between 1s and 24h")
		}
	}
	return nil
}
func duration(s string) time.Duration { d, _ := time.ParseDuration(s); return d }
func fingerprint(v any) string {
	b, _ := json.Marshal(v)
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
func (c inventoryConfig) targetFingerprint(t targetConfig) string {
	return fingerprint(struct {
		Target targetConfig
		Rules  map[string]classificationRule
	}{t, c.Classification})
}

type targetClients struct {
	Kube       kubernetes.Interface
	Dynamic    dynamic.Interface
	EndpointID string
}

func connectTarget(t targetConfig) (targetClients, error) {
	var cfg *rest.Config
	var err error
	if t.InCluster {
		cfg, err = rest.InClusterConfig()
	} else {
		var b []byte
		b, err = boundedFile(t.Kubeconfig.Path, 4<<20)
		if err != nil {
			return targetClients{}, errors.New("credentials unavailable")
		}
		cfg, err = configFromKubeconfig(b, t.Kubeconfig.Context)
	}
	if err != nil {
		return targetClients{}, errors.New("invalid or unsupported credentials")
	}
	cfg.Timeout = 10 * time.Second
	cfg.QPS = 10
	cfg.Burst = 15
	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return targetClients{}, errors.New("cannot initialize Kubernetes client")
	}
	d, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return targetClients{}, errors.New("cannot initialize resource client")
	}
	return targetClients{Kube: k, Dynamic: d, EndpointID: fingerprint(struct {
		Host string
		CA   []byte
	}{cfg.Host, cfg.TLSClientConfig.CAData})}, nil
}
func configFromKubeconfig(b []byte, contextName string) (*rest.Config, error) {
	raw, err := clientcmd.Load(b)
	if err != nil {
		return nil, errors.New("invalid kubeconfig")
	}
	ctx := raw.Contexts[contextName]
	if ctx == nil {
		return nil, errors.New("selected context missing")
	}
	cl, u := raw.Clusters[ctx.Cluster], raw.AuthInfos[ctx.AuthInfo]
	if cl == nil || u == nil {
		return nil, errors.New("selected context references missing entries")
	}
	server, err := url.Parse(cl.Server)
	if err != nil || server.Scheme != "https" || server.Host == "" || server.User != nil || server.RawQuery != "" || server.Fragment != "" || cl.InsecureSkipTLSVerify || cl.CertificateAuthority != "" {
		return nil, errors.New("verified self-contained HTTPS configuration required")
	}
	if u.Exec != nil || u.AuthProvider != nil || u.TokenFile != "" || u.ClientCertificate != "" || u.ClientKey != "" || u.Impersonate != "" || u.ImpersonateUID != "" || len(u.ImpersonateGroups) > 0 || len(u.ImpersonateUserExtra) > 0 || u.Username != "" || u.Password != "" {
		return nil, errors.New("external or interactive authentication unsupported")
	}
	hasToken, hasCert := u.Token != "", len(u.ClientCertificateData) > 0 && len(u.ClientKeyData) > 0
	if hasToken == hasCert || (!hasCert && (len(u.ClientCertificateData) > 0 || len(u.ClientKeyData) > 0)) {
		return nil, errors.New("one embedded token or certificate/key pair required")
	}
	cfg, err := clientcmd.NewNonInteractiveClientConfig(*raw, contextName, &clientcmd.ConfigOverrides{}, nil).ClientConfig()
	if err != nil {
		return nil, errors.New("invalid selected credentials")
	}
	return cfg, nil
}

type classifiedValue struct {
	ID      string `json:"id"`
	Display string `json:"display"`
}

func classify(rule classificationRule, sources map[string]map[string]string) *classifiedValue {
	for _, source := range rule.Sources {
		for _, key := range rule.LabelKeys {
			v := sources[source][key]
			if v == "" {
				continue
			}
			id := v
			if !rule.CaseSensitive {
				id = strings.ToLower(v)
			}
			display := v
			switch rule.DisplayCase {
			case "upper":
				display = strings.ToUpper(v)
			case "lower":
				display = strings.ToLower(v)
			}
			return &classifiedValue{ID: id, Display: display}
		}
	}
	return nil
}
