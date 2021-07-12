package kclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComponentPortAnnotationName annotation is used on the secrets that are created for each exposed port of the component
const ComponentPortAnnotationName = "component-port"

// CreateTLSSecret creates a TLS Secret with the given certificate and private key
// serviceName is the name of the service for the target reference
// ingressDomain is the ingress domain to use for the ingress
func (c *Client) CreateTLSSecret(tlsCertificate []byte, tlsPrivKey []byte, objectMeta metav1.ObjectMeta) (*corev1.Secret, error) {
	if objectMeta.Name == "" {
		return nil, fmt.Errorf("tlsSecret name is empty")
	}
	data := make(map[string][]byte)
	data["tls.crt"] = tlsCertificate
	data["tls.key"] = tlsPrivKey
	secretTemplate := corev1.Secret{
		ObjectMeta: objectMeta,
		Type:       corev1.SecretTypeTLS,
		Data:       data,
	}

	secret, err := c.KubeClient.CoreV1().Secrets(c.Namespace).Create(context.TODO(), &secretTemplate, metav1.CreateOptions{FieldManager: FieldManager})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create secret %s", objectMeta.Name)
	}
	return secret, nil
}

// SelfSignedCertificate struct is the return type of function GenerateSelfSignedCertificate
// CertPem is the byte array for certificate pem encode
// KeyPem is the byte array for key pem encode
type SelfSignedCertificate struct {
	CertPem []byte
	KeyPem  []byte
}

// GenerateSelfSignedCertificate creates a self-signed SSl certificate
func GenerateSelfSignedCertificate(host string) (SelfSignedCertificate, error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return SelfSignedCertificate{}, errors.Wrap(err, "unable to generate rsa key")
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName:   "Odo self-signed certificate",
			Organization: []string{"Odo"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		DNSNames:              []string{"*." + host},
	}

	certificateDerEncoding, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return SelfSignedCertificate{}, errors.Wrap(err, "unable to create certificate")
	}
	out := &bytes.Buffer{}
	err = pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: certificateDerEncoding})
	if err != nil {
		return SelfSignedCertificate{}, errors.Wrap(err, "unable to encode certificate")
	}
	certPemEncode := out.String()
	certPemByteArr := []byte(certPemEncode)

	tlsPrivKeyEncoding := x509.MarshalPKCS1PrivateKey(privateKey)
	err = pem.Encode(out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: tlsPrivKeyEncoding})
	if err != nil {
		return SelfSignedCertificate{}, errors.Wrap(err, "unable to encode rsa private key")
	}
	keyPemEncode := out.String()
	keyPemByteArr := []byte(keyPemEncode)

	return SelfSignedCertificate{CertPem: certPemByteArr, KeyPem: keyPemByteArr}, nil
}

// GetSecret returns the Secret object in the given namespace
func (c *Client) GetSecret(name, namespace string) (*corev1.Secret, error) {
	secret, err := c.KubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the secret %s", secret)
	}
	return secret, nil
}

// UpdateSecret updates the given Secret object in the given namespace
func (c *Client) UpdateSecret(secret *corev1.Secret, namespace string) (*corev1.Secret, error) {
	secret, err := c.KubeClient.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to update the secret %s", secret)
	}
	return secret, nil
}

// DeleteSecret updates the given Secret object in the given namespace
func (c *Client) DeleteSecret(secretName, namespace string) error {
	err := c.KubeClient.CoreV1().Secrets(namespace).Delete(context.TODO(), secretName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to delete the secret %s", secretName)
	}
	return nil
}

// CreateSecret generates and creates the secret
// commonObjectMeta is the ObjectMeta for the service
func (c *Client) CreateSecret(objectMeta metav1.ObjectMeta, data map[string]string, ownerReference metav1.OwnerReference) error {

	secret := corev1.Secret{
		ObjectMeta: objectMeta,
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
	}
	secret.SetOwnerReferences(append(secret.GetOwnerReferences(), ownerReference))
	_, err := c.KubeClient.CoreV1().Secrets(c.Namespace).Create(context.TODO(), &secret, metav1.CreateOptions{FieldManager: FieldManager})
	if err != nil {
		return errors.Wrapf(err, "unable to create secret for %s", objectMeta.Name)
	}
	return nil
}

// CreateSecrets creates a secret for each port, containing the host and port of the component
// This is done so other components can later inject the secret into the environment
// and have the "coordinates" to communicate with this component
func (c *Client) CreateSecrets(componentName string, commonObjectMeta metav1.ObjectMeta, svc *corev1.Service, ownerReference metav1.OwnerReference) error {
	originalName := commonObjectMeta.Name
	for _, svcPort := range svc.Spec.Ports {
		portAsString := fmt.Sprintf("%v", svcPort.Port)

		// we need to create multiple secrets, so each one has to contain the port in it's name
		// so we change the name of each secret by adding the port number
		commonObjectMeta.Name = fmt.Sprintf("%v-%v", originalName, portAsString)

		// we also add the port as an annotation to the secret
		// this comes in handy when we need to "query" for the appropriate secret
		// of a component based on the port
		commonObjectMeta.Annotations[ComponentPortAnnotationName] = portAsString

		err := c.CreateSecret(
			commonObjectMeta,
			map[string]string{
				secretKeyName(componentName, "host"): svc.Name,
				secretKeyName(componentName, "port"): portAsString,
			},
			ownerReference)

		if err != nil {
			return errors.Wrapf(err, "unable to create Secret for %s", commonObjectMeta.Name)
		}
	}

	// restore the original values of the fields we changed
	commonObjectMeta.Name = originalName
	delete(commonObjectMeta.Annotations, ComponentPortAnnotationName)

	return nil
}

// ListSecrets lists all the secrets based on the given label selector
func (c *Client) ListSecrets(labelSelector string) ([]corev1.Secret, error) {
	listOptions := metav1.ListOptions{}
	if len(labelSelector) > 0 {
		listOptions = metav1.ListOptions{
			LabelSelector: labelSelector,
		}
	}

	secretList, err := c.KubeClient.CoreV1().Secrets(c.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secret list")
	}

	return secretList.Items, nil
}

// WaitAndGetSecret blocks and waits until the secret is available
func (c *Client) WaitAndGetSecret(name string, namespace string) (*corev1.Secret, error) {
	klog.V(3).Infof("Waiting for secret %s to become available", name)

	w, err := c.KubeClient.CoreV1().Secrets(namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to watch secret")
	}
	defer w.Stop()
	for {
		val, ok := <-w.ResultChan()
		if !ok {
			break
		}
		if e, ok := val.Object.(*corev1.Secret); ok {
			klog.V(3).Infof("Secret %s now exists", e.Name)
			return e, nil
		}
	}
	return nil, errors.Errorf("unknown error while waiting for secret '%s'", name)
}

func secretKeyName(componentName, baseKeyName string) string {
	return fmt.Sprintf("COMPONENT_%v_%v", strings.Replace(strings.ToUpper(componentName), "-", "_", -1), strings.ToUpper(baseKeyName))
}
