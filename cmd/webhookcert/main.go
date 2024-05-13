/* Copyright © 2024 Broadcom, Inc. All Rights Reserved.
   SPDX-License-Identifier: Apache-2.0 */

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"math/big"
	"os"
	"path"
	"sync"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/vmware-tanzu/nsx-operator/pkg/config"
	"github.com/vmware-tanzu/nsx-operator/pkg/logger"
)

var (
	log                            = logger.Log
	validatingWebhookConfiguration = "nsx-operator-validating-webhook-configuration"
	namespace                      = "vmware-system-nsx"
	webhookSecret                  = "nsx-operator-webhook-secret"
)

func main() {
	fmt.Println("start...")
	wg := &sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go createSecret(wg)
	}
	wg.Wait()
	fmt.Println("end...")
	//log.Info("Generating webhook certificates...")
	//if err := generateWebhookCerts(); err != nil {
	//	panic(err)
	//}
}

// WriteFile writes data in the file at the given path
func writeFile(filepath string, cert *bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(cert.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func generateWebhookCerts() error {
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
	if err = writeFile(path.Join(config.WebhookCertDir, "tls.crt"), serverCertPEM); err != nil {
		log.Error(err, "Failed to write tls cert", "Path", path.Join(config.WebhookCertDir, "tls.crt"))
		return err
	}

	if err = writeFile(path.Join(config.WebhookCertDir, "tls.key"), serverKeyPEM); err != nil {
		log.Error(err, "Failed to write tls cert", "Path", path.Join(config.WebhookCertDir, "tls.key"))
		return err
	}
	if err = updateWebhookConfig(caPEM); err != nil {
		return err
	}
	return nil
}

func updateWebhookConfig(caCert *bytes.Buffer) error {
	config := ctrl.GetConfigOrDie()
	kubeClient := kubernetes.NewForConfigOrDie(config)
	webhookCfg, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), validatingWebhookConfiguration, v1.GetOptions{})
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
		if _, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.TODO(), webhookCfg, v1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func createSecret(wg *sync.WaitGroup) error {
	defer wg.Done()
	config := ctrl.GetConfigOrDie()
	kubeClient := kubernetes.NewForConfigOrDie(config)
	secret := &corev1.Secret{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      "jsui-secret",
			Namespace: "default",
		},
		Immutable:  nil,
		Data:       nil,
		StringData: nil,
		Type:       "",
	}
	_, err := kubeClient.CoreV1().Secrets("default").Create(context.TODO(), secret, v1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Println("secret already exists, ignore creating")
		} else {
			fmt.Printf("failed to create secret, err: %s\n", err)
		}
	} else {
		fmt.Println("secret created")
	}
	return nil
}
