package bitbucketserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	log "github.com/Sirupsen/logrus"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"net/http"
)

func NewClient(ConsumerRSA string, ConsumerKey string, URL string) *oauth.Consumer {
	//TODO: make this configurable
	privateKeyFileContents, err := ioutil.ReadFile(ConsumerRSA)
	log.Info("Tried to read the key")
	if err != nil {
		log.Error(err)
	}

	block, _ := pem.Decode([]byte(privateKeyFileContents))
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Error(err)
	}

	c := oauth.NewRSAConsumer(
		ConsumerKey,
		privateKey,
		oauth.ServiceProvider{
			RequestTokenUrl:   URL + "/plugins/servlet/oauth/request-token",
			AuthorizeTokenUrl: URL + "/plugins/servlet/oauth/authorize",
			AccessTokenUrl:    URL + "/plugins/servlet/oauth/access-token",
			HttpMethod:        "POST",
		})
	c.HttpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return c
}

func NewClientWithToken(Consumer *oauth.Consumer, AccessToken string) *http.Client {
	var token oauth.AccessToken
	token.Token = AccessToken
	client, err := Consumer.MakeHttpClient(&token)
	if err != nil {
		log.Error(err)
	}
	return client
}
