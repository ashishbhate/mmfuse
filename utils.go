package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func loadCert(caCertPath string) *http.Client {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	// Read in the cert file
	certs, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		log.Fatalf("Failed to append %q to RootCAs: %v", caCertPath, err)
	}
	// Append our cert to the system pool
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		log.Println("No certs appended, using system certs only")
	}
	// Trust the augmented cert pool in our client
	config := &tls.Config{
		RootCAs: rootCAs,
	}
	tr := &http.Transport{TLSClientConfig: config}
	return &http.Client{Transport: tr}
}

func humanTime(msec int64) string {
	return time.Unix(msec/1000, 0).Format("2006-01-02 15:04:05")
}
