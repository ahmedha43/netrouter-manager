package manager

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ahmedha43/netrouter-manager/native/internal/protocol"
)

func TestClientCallsRouterOverMutualTLS(t *testing.T) {
	caCertificate, caKey, caPEM := makeTestCA(t)
	serverCertificate := issueTestCertificate(t, caCertificate, caKey, "router.test", false)
	clientCertificate := issueTestCertificate(t, caCertificate, caKey, "manager.test", true)
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("cannot load test CA")
	}
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{serverCertificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer connection.Close()
		var request protocol.Request
		if err := json.NewDecoder(connection).Decode(&request); err != nil {
			done <- err
			return
		}
		result, _ := json.Marshal(protocol.SystemStatus{Identity: "lab-router", Architecture: "amd64"})
		done <- json.NewEncoder(connection).Encode(protocol.Response{Version: protocol.Version, ID: request.ID, Result: result})
	}()
	client, err := DialTLS(listener.Addr().String(), &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, Certificates: []tls.Certificate{clientCertificate}, ServerName: "router.test"})
	if err != nil {
		t.Fatalf("dial mTLS: %v", err)
	}
	defer client.Close()
	var status protocol.SystemStatus
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Call(ctx, protocol.GetSystemStatus, map[string]string{}, &status); err != nil {
		t.Fatalf("call: %v", err)
	}
	if status.Identity != "lab-router" || status.Architecture != "amd64" {
		t.Fatalf("unexpected status: %#v", status)
	}
	if err := <-done; err != nil {
		t.Fatalf("server: %v", err)
	}
}

func makeTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "NetRouter Test CA"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, key, pemEncode("CERTIFICATE", der)
}

func issueTestCertificate(t *testing.T, ca *x509.Certificate, caKey *ecdsa.PrivateKey, name string, client bool) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: name}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature}
	if client {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.DNSNames = []string{name}
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(pemEncode("CERTIFICATE", der), pemEncodeECKey(key))
	if err != nil {
		t.Fatal(err)
	}
	return certificate
}
