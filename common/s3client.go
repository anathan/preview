package common

import (
	"bytes"
	"fmt"
	"github.com/ngerakines/preview/util"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type S3Client interface {
	Put(s3object S3Object, content []byte) error
	Get(bucket, file string) (S3Object, error)
	Delete(bucket, file string) error
	NewObject(name, bucket, contentType string) (S3Object, error)
}

type S3Object interface {
	FileName() string
	Bucket() string
	Url() string
	ContentType() string
	Payload() []byte
}

type AmazonS3Client struct {
	config *AmazonS3ClientConfig
}

type AmazonS3Object struct {
	name, bucket string
	payload      []byte
	contentType  string
}

type AmazonS3ClientConfig struct {
	key, secret, host string
}

func NewBasicS3Config(key, secret, host string) *AmazonS3ClientConfig {
	return &AmazonS3ClientConfig{key, secret, host}
}

func NewAmazonS3Client(config *AmazonS3ClientConfig) S3Client {
	return &AmazonS3Client{config}
}

func NewAmazonS3Object(name, bucket, contentType string) S3Object {
	return &AmazonS3Object{name, bucket, nil, contentType}
}

func NewAmazonS3ResponseObject(name, bucket, contentType string, content []byte) S3Object {
	return &AmazonS3Object{name, bucket, content, contentType}
}

func (client *AmazonS3Client) Put(s3object S3Object, content []byte) error {
	resource := fmt.Sprintf("/%s/%s", s3object.Bucket(), s3object.FileName())
	date, signature := client.createSignature("PUT", s3object.ContentType(), resource)
	headers := make(map[string]string)
	headers["Host"] = fmt.Sprintf("%s.s3.amazonaws.com", s3object.Bucket())
	headers["Date"] = date
	headers["Content-Type"] = s3object.ContentType()
	headers["Authorization"] = fmt.Sprintf("AWS %s:%s", client.config.key, signature)
	url := fmt.Sprintf("%s/%s/%s", client.config.host, s3object.Bucket(), s3object.FileName())
	_, err := client.submitPutRequest(url, content, headers)
	if err != nil {
		return err
	}
	return nil
}

func (client *AmazonS3Client) Get(bucket, file string) (S3Object, error) {
	resource := fmt.Sprintf("/%s/%s", bucket, file)
	date, signature := client.createSignature("GET", "", resource)
	headers := make(map[string]string)
	headers["Host"] = fmt.Sprintf("%s.s3.amazonaws.com", bucket)
	headers["Date"] = date
	headers["Authorization"] = fmt.Sprintf("AWS %s:%s", client.config.key, signature)
	url := fmt.Sprintf("%s/%s/%s", client.config.host, bucket, file)
	body, contentType, err := client.submitGetRequest(url, headers)
	if err != nil {
		return nil, err
	}
	return NewAmazonS3ResponseObject(file, bucket, contentType, body), nil
}

func (client *AmazonS3Client) createSignature(method, contentType, resource string) (string, string) {
	date := time.Now().UTC().Format(time.RFC1123Z)
	stringToSign := fmt.Sprintf("%s\n\n%s\n%s\n%s", method, contentType, date, resource)
	signature := util.ComputeHmac256(stringToSign, client.config.secret)
	return date, signature
}

func (client *AmazonS3Client) submitGetRequest(url string, headers map[string]string) ([]byte, string, error) {
	log.Println("url", url)
	response, err := client.executeRequest("GET", url, nil, headers)
	if err != nil {
		log.Println("response", response)
		return nil, "", err
	}
	if response.StatusCode == 404 {
		return nil, "", ErrorS3FileNotFound
	}
	defer response.Body.Close()
	log.Println("status", response.Status)
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err.Error())
		return nil, "", err
	}
	return body, response.Header.Get("Content-Type"), nil
}

func (client *AmazonS3Client) submitPutRequest(url string, payload []byte, headers map[string]string) ([]byte, error) {
	response, err := client.executeRequest("PUT", url, bytes.NewReader(payload), headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (client *AmazonS3Client) submitDeleteRequest(url string, headers map[string]string) ([]byte, error) {
	response, err := client.executeRequest("DELETE", url, nil, headers)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (client *AmazonS3Client) executeRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	for header, headerValue := range headers {
		request.Header.Set(header, headerValue)
	}
	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (client *AmazonS3Client) Delete(bucket, file string) error {
	resource := fmt.Sprintf("/%s/%s", bucket, file)
	date, signature := client.createSignature("DELETE", "", resource)
	headers := make(map[string]string)
	headers["Host"] = fmt.Sprintf("%s.s3.amazonaws.com", bucket)
	headers["Date"] = date
	headers["Authorization"] = fmt.Sprintf("AWS %s:%s", client.config.key, signature)
	url := fmt.Sprintf("%s/%s/%s", client.config.host, bucket, file)
	_, err := client.submitDeleteRequest(url, headers)
	if err != nil {
		return err
	}
	return nil
}

func (client *AmazonS3Client) NewObject(name, bucket, contentType string) (S3Object, error) {
	return NewAmazonS3Object(name, bucket, contentType), nil
}

func (s3obj *AmazonS3Object) FileName() string {
	return s3obj.name
}

func (s3obj *AmazonS3Object) Bucket() string {
	return s3obj.bucket
}

func (s3obj *AmazonS3Object) ContentType() string {
	return s3obj.contentType
}

func (s3obj *AmazonS3Object) Payload() []byte {
	return s3obj.payload
}

func (s3obj *AmazonS3Object) Url() string {
	return "http://s3.amazonaws.com/" + s3obj.name
}
