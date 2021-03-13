package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"os"
	"strings"

	cmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"

	"github.com/go-resty/resty/v2"
)

// DNSRecord a DNS record
//{
//	"id": "497f6eca-6276-4993-bfeb-53cbbbba6f08",
//	"type": "a",
//	"name": "string",
//	"value": { },
//	"ttl": 120,
//	"cloud": false,
//	"upstream_https": "default",
//	"ip_filter_mode":
//	{},
//	"can_delete": true,
//	"is_protected": false,
//	"created_at": "2019-08-24T14:15:22Z",
//	"updated_at": "2019-08-24T14:15:22Z"
//}
type DNSRecord struct {
	ID    string            `json:"id,omitempty"`
	Type  string            `json:"type"`
	Name  string            `json:"name"`
	Value map[string]string `json:"value"`
	Cloud bool              `json:"cloud"`
	TTL   int               `json:"ttl,omitempty"`
}

type DNSRecords struct {
	Data []DNSRecord `json:"data"`
}

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&arvanDNSProviderSolver{},
	)
}

// arvanDNSProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/jetstack/cert-manager/pkg/acme/webhook.Solver`
// interface.
type arvanDNSProviderSolver struct {
	// If a Kubernetes 'clientset' is needed, you must:
	// 1. uncomment the additional `client` field in this structure below
	// 2. uncomment the "k8s.io/client-go/kubernetes" import at the top of the file
	// 3. uncomment the relevant code in the Initialize method below
	// 4. ensure your webhook's service account has the required RBAC role
	//    assigned to it for interacting with the Kubernetes APIs you need.
	client *kubernetes.Clientset
}

// arvanDNSProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type arvanDNSProviderConfig struct {
	// Change the two fields below according to the format of the configuration
	// to be decoded.
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.

	AuthAPIKey       string                  `json:"authApiKey"`
	AuthAPISecretRef cmeta.SecretKeySelector `json:"authApiSecretRef"`
	BaseURL          string                  `json:"baseUrl"`
	TTL              int                     `json:"ttl"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *arvanDNSProviderSolver) Name() string {
	return "arvancloud"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *arvanDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Error(err)
		return err
	}

	// TODO: do something more useful with the decoded configuration
	fmt.Printf("Decoded configuration %v", cfg)

	apiSecret, err := c.validateAndGetSecret(&cfg, ch.ResourceNamespace)
	if err != nil {
		klog.Errorf("Failed to validate config: %v", err)
		return fmt.Errorf("Failed to validate config: %v", err)
	}

	recordName, domain := c.extractRecordName(ch.ResolvedFQDN)
	//{"type":"TXT","ttl":120,"name":"asds","cloud":false,"value":{"text":"asd"}}
	vals := make(map[string]string)
	vals["text"] = ch.Key
	record := DNSRecord{
		Type:  "TXT",
		Name:  recordName,
		Value: vals,
		Cloud: false,
		TTL:   cfg.TTL,
	}

	client := resty.New()
	// See we are not setting content-type header, since go-resty automatically detects Content-Type for you
	resp, err := client.R().
		SetBody(record).
		SetHeader("Accept", "application/json").
		SetAuthToken(apiSecret).
		SetAuthScheme("Apikey").
		Post(
			c.urlFactory(
				&cfg,
				"/cdn/4.0/domains/{domain}/dns-records",
				"{domain}", domain,
			))

	klog.Info(resp.Request.URL, resp.Request.Header, resp.Request.Body, resp.StatusCode(), string(resp.Body()))

	if err == nil {
		if resp.StatusCode() != 201 {
			err = fmt.Errorf("Error in creating dns record: %s", string(resp.Body()))
			klog.Error(err)
		}
	}
	return err
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *arvanDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	id, err := c.getRecordID(ch)
	if err != nil {
		klog.Error(err)
		return err
	}
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Error(err)
		return err
	}
	apiSecret, err := c.validateAndGetSecret(&cfg, ch.ResourceNamespace)
	if err != nil {
		klog.Errorf("Failed to validate config: %v", err)
		return fmt.Errorf("Failed to validate config: %v", err)
	}
	// See we are not setting content-type header, since go-resty automatically detects Content-Type for you

	domain := ch.ResolvedZone[:len(ch.ResolvedZone)-1]

	client := resty.New()
	resp, err := client.R().
		SetAuthToken(apiSecret).
		SetAuthScheme("Apikey").
		SetHeader("Accept", "application/json").
		Delete(
			c.urlFactory(
				&cfg,
				"/cdn/4.0/domains/{domain}/dns-records/{id}",
				"{domain}", domain,
				"{id}", id,
			))

	klog.Info(resp.Request.URL, resp.Request.Header, resp.Request.Body, resp.StatusCode(), string(resp.Body()))

	if err != nil {
		if resp == nil {
			err = fmt.Errorf("Api call has no resutl")
			klog.Error(err)
		} else if resp.StatusCode() != 200 {
			err = fmt.Errorf("Error in creating dns record: %s", string(resp.Body()))
			klog.Error(err)
		}
	}
	return err
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *arvanDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	///// UNCOMMENT THE BELOW CODE TO MAKE A KUBERNETES CLIENTSET AVAILABLE TO
	///// YOUR CUSTOM DNS PROVIDER

	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.client = cl
	///// END OF CODE TO MAKE KUBERNETES CLIENTSET AVAILABLE
	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (arvanDNSProviderConfig, error) {
	cfg := arvanDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func (c *arvanDNSProviderSolver) validateAndGetSecret(cfg *arvanDNSProviderConfig, namespace string) (string, error) {
	fmt.Printf("validateAndGetSecret...")
	// Check that the host is defined
	if cfg.AuthAPIKey != "" {
		return cfg.AuthAPIKey, nil
	}

	// Try to load the API key
	if cfg.AuthAPISecretRef.LocalObjectReference.Name == "" {
		return "", errors.New("No Arvan API secret provided")
	}

	sec, err := c.client.CoreV1().Secrets(namespace).Get(context.TODO(), cfg.AuthAPISecretRef.LocalObjectReference.Name, metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return "", err
	}

	secBytes, ok := sec.Data[cfg.AuthAPISecretRef.Key]
	if !ok {
		return "", fmt.Errorf("Key %q not found in secret \"%s/%s\"", cfg.AuthAPISecretRef.Key, cfg.AuthAPISecretRef.LocalObjectReference.Name, namespace)
	}

	apiKey := string(secBytes)

	return apiKey, nil
}

func (c *arvanDNSProviderSolver) urlFactory(cfg *arvanDNSProviderConfig, uri string, args ...string) string {
	r := strings.NewReplacer(args...)
	urlFormat := "https://napi.arvancloud.com" + uri
	if cfg.BaseURL != "" {
		urlFormat = cfg.BaseURL + uri
	}
	return r.Replace(urlFormat)
}
func (c *arvanDNSProviderSolver) extractRecordName(fqdn string) (record, domain string) {
	fqdn = util.UnFqdn(fqdn)
	parts := strings.Split(fqdn ,".")
	domain = strings.Join(parts[len(parts)-2:], ".")
	klog.Infof("Request : %s => %s", fqdn, domain)
	if idx := strings.Index(fqdn, "."+domain); idx != -1 {
		return fqdn[:idx], domain
	}
	return fqdn, domain
}

func (c *arvanDNSProviderSolver) getRecordID(ch *v1alpha1.ChallengeRequest) (string, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Error(err)
		return "", err
	}
	apiSecret, err := c.validateAndGetSecret(&cfg, ch.ResourceNamespace)
	if err != nil {
		klog.Errorf("Failed to validate config: %v", err)
		return "", fmt.Errorf("Failed to validate config: %v", err)
	}
	recordName, domain := c.extractRecordName(ch.ResolvedFQDN)
	recordName = strings.Replace(recordName, "-", "_", -1)

	client := resty.New()
	resp, err := client.R().
		SetAuthToken(apiSecret).
		SetAuthScheme("Apikey").
		SetHeader("Accept", "application/json").
		SetQueryString(fmt.Sprintf("search=%s&page=1&per_page=25", recordName)).
		Get(
			c.urlFactory(
				&cfg,
				"/cdn/4.0/domains/{domain}/dns-records",
				"{domain}", domain,
			))
	klog.Info(resp.Request.URL, resp.Request.Header, resp.Request.Body, resp.StatusCode(), string(resp.Body()))

	if err != nil {
		klog.Error(err)
		return "", err
	}

	recs := DNSRecords{}
	err = json.Unmarshal(resp.Body(), &recs)
	if err != nil {
		klog.Error(err)
		return "", err
	}
	if len(recs.Data) == 1 {
		return recs.Data[0].ID, nil
	} else {
		err = fmt.Errorf("Domain not Found")
		klog.Error(err)
		return "", err
	}
}
