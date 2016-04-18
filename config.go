package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
)

func setRootCAs(c *tls.Config) error {
	caCert, err := ioutil.ReadFile(os.Getenv("DOCKER_CA_CERT_PATH"))
	if err != nil {
		fmt.Println(1)
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	c.RootCAs = caCertPool
	return nil
}

func setCerts(c *tls.Config) error {
	cert, err := tls.LoadX509KeyPair(os.Getenv("DOCKER_CERT_PATH"), os.Getenv("DOCKER_KEY_PATH"))
	if err != nil {
		fmt.Println(2)
		return err
	}
	c.Certificates = []tls.Certificate{cert}
	return nil
}

func tlsConfig() (*tls.Config, error) {
	c := &tls.Config{}
	err := setRootCAs(c)
	if err != nil {
		return nil, err
	}
	err = setCerts(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
