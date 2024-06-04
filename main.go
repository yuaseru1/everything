package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed index.html
var assetIndex []byte

func main() {
	port := or(os.Getenv("PORT"), "8888")
	log.Println("starting on port", port)
	log.Fatalln(http.ListenAndServe("0.0.0.0:"+port,
		http.HandlerFunc(handler)))
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Now().Sub(start))
		if err := recover(); err != nil {
			log.Printf("panic: %s %s: %v", r.Method, r.URL.Path, err)
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("server error: %v", err)))
		}
	}()
	if r.Method == "POST" {
		user := r.URL.Query().Get("user")
		body, err := ioutil.ReadAll(r.Body)
		check(err)
		r.Body.Close()
		checkpoint, err := strconv.Atoi(r.URL.Query().Get("checkpoint"))
		check(err)
		data := string(awsGet(user)) + string(body)
		awsPut(user, []byte(data))
		datums := strings.Split(data, "\n")
		fmt.Println("sync", user, checkpoint, string(body), len(datums))
		w.Header().Set("Content-Type", "application/json")
		check(json.NewEncoder(w).Encode(datums[checkpoint:]))
		return
	}
	if r.URL.Path == "/app.webmanifest" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"Everything","background_color":"white"}`))
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(assetIndex)
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func awsGet(key string) []byte {
	target := fmt.Sprintf("https://%s.s3.amazonaws.com/%s",
		os.Getenv("AWS_BUCKET"), key)
	return aws("GET", target, nil, nil)
}
func awsPut(key string, value []byte) {
	aws(
		"PUT",
		fmt.Sprintf("https://%s.s3.amazonaws.com/%s",
			os.Getenv("AWS_BUCKET"), key),
		map[string]string{"content-type": or(http.DetectContentType(value), "application/octet-stream")},
		value,
	)
}

func aws(method, target string, headers map[string]string, body []byte) []byte {
	var awsRegion = "us-east-1"
	var awsService = "s3"
	var awsAccessKey = os.Getenv("AWS_ACCESS_KEY")
	var awsSecretKey = os.Getenv("AWS_SECRET_KEY")
	req, err := http.NewRequest(method, target, bytes.NewReader(body))
	check(err)

	// Headers
	payloadHash := hashAndEncode(body)
	now := time.Now().UTC()
	date := now.Format("20060102T150405Z")
	canonicalHeaders := ""
	headerNames := []string{"host", "x-amz-date", "x-amz-content-sha256"}
	headers2 := map[string]string{}
	headers2["host"] = req.URL.Host
	headers2["x-amz-date"] = date
	headers2["x-amz-content-sha256"] = payloadHash
	if headers != nil {
		for k, v := range headers {
			headerNames = append(headerNames, strings.ToLower(k))
			headers2[strings.ToLower(k)] = strings.TrimSpace(v)
		}
	}
	sort.Strings(headerNames)
	signedHeaders := strings.Join(headerNames, ";")
	for _, k := range headerNames {
		req.Header.Set(k, headers2[k])
		canonicalHeaders += fmt.Sprintf("%s:%s\n", k, headers2[k])
	}

	// Cannonical
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method, req.URL.Path, req.URL.Query().Encode(), canonicalHeaders, signedHeaders, payloadHash)
	canonicalRequestHash := hashAndEncode([]byte(canonicalRequest))
	algorithm := "AWS4-HMAC-SHA256"
	credentialDate := now.Format("20060102")
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", credentialDate, awsRegion, awsService)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s", algorithm, date, credentialScope, canonicalRequestHash)

	// Sign
	signingKey := getSignatureKey(awsSecretKey, credentialDate, awsRegion, awsService)
	signatureSHA := hmac.New(sha256.New, signingKey)
	signatureSHA.Write([]byte(stringToSign))
	signatureString := hex.EncodeToString(signatureSHA.Sum(nil))

	// Authorization Header
	authorizationHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s", algorithm, awsAccessKey, credentialScope, signedHeaders, signatureString)
	req.Header.Set("Authorization", authorizationHeader)

	res, err := http.DefaultClient.Do(req)
	check(err)

	resBody, err := ioutil.ReadAll(res.Body)
	check(err)
	res.Body.Close()

	if res.StatusCode != 200 {
		panic(fmt.Errorf("aws error: %d: %s", res.StatusCode, string(resBody)))
	}
	return resBody
}

func hashAndEncode(s []byte) string {
	h := sha256.New()
	h.Write(s)
	return hex.EncodeToString(h.Sum(nil))
}

func sign(key []byte, message string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return h.Sum(nil)
}

func getSignatureKey(key, dateStamp, region, service string) []byte {
	kDate := sign([]byte("AWS4"+key), dateStamp)
	kRegion := sign(kDate, region)
	kService := sign(kRegion, service)
	kSigning := sign(kService, "aws4_request")
	return kSigning
}
