package cert

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	"github.com/vmware-tanzu/nsx-operator/pkg/controllers/common"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"math/big"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

var (
	log                            = logger.Log
	ResultNormal                   = common.ResultNormal
	ResultRequeue                  = common.ResultRequeue
	ResultRequeueAfter5mins        = common.ResultRequeueAfter5mins
	MetricResTypeSubnetSet         = common.MetricResTypeSubnetSet
	tlsCert                        = "tls.cert"
	tlsKey                         = "tls.key"
	validatingWebhookConfiguration = "nsx-operator-validating-webhook-configuration"
)

func generateWebhookCerts(client client.Client, secret *v1.Secret) error {
	var caPEM, serverCertPEM, serverKeyPEM *bytes.Buffer
	// CA config
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	ca := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"broadcom.com"},
			CommonName:   "webhook",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Error(err, "Failed to generate private key")
		return err
	}

	// Self-signed CA certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		log.Error(err, "Failed to generate CA")
		return err
	}

	// PEM encode CA cert
	caPEM = new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	dnsNames := []string{"subnetset", "subnetset.vmware-system-nsx", "subnetset.vmware-system-nsx.svc"}
	commonName := "subnetset.vmware-system-nsx.svc"

	serialNumber, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	// server cert config
	cert := &x509.Certificate{
		DNSNames:     dnsNames,
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"broadcom.com"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		log.Error(err, "Failed to generate server key")
		return err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &serverKey.PublicKey, caKey)
	if err != nil {
		log.Error(err, "Failed to sign server certificate")
		return err
	}

	// PEM encode the  server cert and key
	serverCertPEM = new(bytes.Buffer)
	pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	serverKeyPEM = new(bytes.Buffer)
	pem.Encode(serverKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverKey),
	})

	if err = os.MkdirAll(config.WebhookCertDir, 0755); err != nil {
		log.Error(err, "Failed to create directory", "Dir", config.WebhookCertDir)
		return err
	}
	secret.Data[tlsCert] = serverCertPEM.Bytes()
	secret.Data[tlsKey] = serverKeyPEM.Bytes()
	if err := client.Update(context.TODO(), secret); err != nil {
		log.Error(err, "failed to patch secret")
	}
	if err = updateWebhookConfig(caPEM); err != nil {
		return err
	}
	return nil
}

func updateWebhookConfig(caCert *bytes.Buffer) error {
	config := ctrl.GetConfigOrDie()
	kubeClient := kubernetes.NewForConfigOrDie(config)
	webhookCfg, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), validatingWebhookConfiguration, metav1.GetOptions{})
	if err != nil {
		return err
	}
	updated := false
	for idx, webhook := range webhookCfg.Webhooks {
		if bytes.Equal(webhook.ClientConfig.CABundle, caCert.Bytes()) {
			continue
		}
		updated = true
		webhook.ClientConfig.CABundle = caCert.Bytes()
		webhookCfg.Webhooks[idx] = webhook
	}
	if updated {
		if _, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.TODO(), webhookCfg, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

// SubnetSetReconciler reconciles a SubnetSet object
type CertReconciler struct {
	Client client.Client
	Scheme *apimachineryruntime.Scheme
}

func (r *CertReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &v1.Secret{}
	log.Info("reconciling secret", "secret", req.NamespacedName)

	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(err, "unable to fetch ", "req", req.NamespacedName)
		return ResultNormal, client.IgnoreNotFound(err)
	}
	if _, ok := obj.Data[tlsCert]; ok && len(obj.Data[tlsCert]) > 0 {
		log.Info("Cert already exists")
		return ResultNormal, nil
	}
	generateWebhookCerts(r.Client, obj)
	return ctrl.Result{}, nil
}

var predicateFuncs = predicate.Funcs{
	CreateFunc: func(e event.CreateEvent) bool {
		if e.Object.GetNamespace() == "vmware-system-nsx" {
			return true
		}
		return false
	},
	UpdateFunc: func(e event.UpdateEvent) bool {
		//oldObj := e.ObjectOld.(*v1.Secret)
		//newObj := e.ObjectNew.(*v1.Secret)
		return false
	},
	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},
}

func (r *CertReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Watches(
			&v1.Secret{},
			&handler.EnqueueRequestForObject{},
		).Complete(r)
}

func StartCertController(mgr ctrl.Manager) error {
	certReconciler := &CertReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	if err := certReconciler.Start(mgr); err != nil {
		log.Error(err, "failed to create controller", "controller", "Subnet")
		return err
	}
	return nil
}

// Start setup manager
func (r *CertReconciler) Start(mgr ctrl.Manager) error {
	err := r.setupWithManager(mgr)
	if err != nil {
		return err
	}
	return nil
}
