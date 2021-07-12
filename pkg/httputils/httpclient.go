package httputils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/iver-wharf/wharf-core/pkg/logger"
)

var log = logger.New()

// NewClientWithCerts creates a fresh net/http.Client populated with some
// root CA certificates from file.
// Argument must point to an existing file with PEM formatted certificates.
//
// Based on https://forfuncsake.github.io/post/2017/08/trust-extra-ca-cert-in-go-app/
func NewClientWithCerts(localCertFile string) (*http.Client, error) {
	rootCAs, _ := x509.SystemCertPool()

	// Get the SystemCertPool, continue with an empty pool on error
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
		log.Info().Message("Using empty cert pool.")
	} else {
		log.Info().Message("Using system cert pool.")
	}

	// Read in the cert file
	certs, err := ioutil.ReadFile(localCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to append %q to RootCAs: %v", localCertFile, err)
	}

	log.Info().WithString("certsFile", localCertFile).Message("Loaded certs from file.")

	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		log.Warn().WithString("certsFile", localCertFile).
			Message("No new certs appended. Using system certs only.")
	}

	// Trust the augmented cert pool in our client
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config}
	client := &http.Client{Transport: tr}

	return client, nil
}
