package api

import (
	"fmt"
	"github.com/ngerakines/preview/util"
	"log"
	neturl "net/url"
	"strconv"
	"strings"
	"time"
)

type SignatureManager interface {
	Sign(url string) (string, int64, error)
	IsValid(url string) bool
}

// TODO: The key should be set by configuration and should support multiple keys.
type defaultSignatureManager struct {
	key string
}

func NewSignatureManager() SignatureManager {
	signatureManager := new(defaultSignatureManager)
	signatureManager.key = "foo"
	return signatureManager
}

func (signatureManager *defaultSignatureManager) Sign(url string) (string, int64, error) {
	parseUrl, err := signatureManager.parseUrl(url)
	if err != nil {
		log.Println("Could not parse url", err)
		return "", 0, err
	}
	expiresValue := time.Now().Add(5 * time.Minute).UnixNano()
	expires := strconv.FormatInt(expiresValue, 10)
	signature := signatureManager.createSignature(parseUrl.Path, expires)
	q := parseUrl.Query()
	q.Set("signature", signature)
	q.Set("expires", expires)

	parseUrl.RawQuery = q.Encode()

	log.Println("parsed url", parseUrl)
	return parseUrl.String(), expiresValue, nil

}

func (signatureManager *defaultSignatureManager) IsValid(url string) bool {
	parseUrl, err := signatureManager.parseUrl(url)
	if err != nil {
		log.Println("Could not parse url", err)
		return false
	}

	expires := parseUrl.Query().Get("expires")
	signature := parseUrl.Query().Get("signature")
	log.Println("query string expires", expires, "signature", signature)

	checkSignature := signatureManager.createSignature(parseUrl.Path, expires)
	log.Println("check signature", checkSignature)

	return signature == checkSignature
}

func (signatureManager *defaultSignatureManager) parseUrl(uri string) (*neturl.URL, error) {
	if strings.HasPrefix(uri, "/") {
		return neturl.ParseRequestURI(uri)
	}
	return neturl.Parse(uri)
}

func (signatureManager *defaultSignatureManager) createSignature(path, expires string) string {
	stringToSign := fmt.Sprintf("%s\n%s", path, expires)
	signature := util.ComputeHmac256(stringToSign, signatureManager.key)
	return signature
}
