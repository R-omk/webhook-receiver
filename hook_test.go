package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/gorilla/mux"
)

var hookHandlerScript = `
{
  "scripts": [
    {
      "command": "echo",
      "args": [
        "foo"
      ]
    }
  ]
}`

var hookHandlerScriptDenied = `
{
  "scripts": [
    {
      "command": "echo",
      "args": [
        "foo"
      ]
    }
  ],
  "allowedNetworks": [
    "10.0.0.0/8"
  ]
}`

var hookResponseBody = `{
  "results": [
    {
      "stdout": "foo\n",
      "stderr": "",
      "status_code": 0
    }
  ]
}`

var data = []byte(`{"test": "test"}`)

var exposePostHandlerScript = `
{
  "scripts": [
    {
      "command": "echo",
      "args": [
        "{{POST}}"
      ]
    }
  ]
}
`

var exposePostResponseBody = `{
  "results": [
    {
      "stdout": "{\"test\": \"test\"}\n",
      "stderr": "",
      "status_code": 0
    }
  ]
}`

var hookHanderTests = []struct {
	body       string
	echo       bool
	script     string
	statusCode int
	postBody   io.Reader
}{
	{"", false, hookHandlerScript, 200, nil},
	{"Not authorized.\n", false, hookHandlerScriptDenied, 401, nil},
	{hookResponseBody, true, hookHandlerScript, 200, nil},
	{exposePostResponseBody, true, exposePostHandlerScript, 200, bytes.NewBuffer(data)},
}

func TestHookHandler(t *testing.T) {
	// Start a test server so we can test using the gorilla mux.
	r := mux.NewRouter()
	r.HandleFunc("/{id}", hookHandler).Methods("POST")
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Set configdir option
	tempdir := os.TempDir()
	configdir = tempdir

	for _, tt := range hookHanderTests {
		// Set the echo config option.
		echo = tt.echo

		f, err := os.Create(path.Join(tempdir, "test.json"))
		if err != nil {
			t.Errorf(err.Error())
		}
		defer os.Remove(f.Name())
		defer f.Close()

		_, err = f.WriteString(tt.script)
		if err != nil {
			t.Errorf(err.Error())
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", ts.URL, "test"), tt.postBody)
		if err != nil {
			t.Errorf(err.Error())
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf(err.Error())
		}
		if resp.StatusCode != tt.statusCode {
			t.Errorf("wanted %d, got %d", tt.statusCode, resp.StatusCode)
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf(err.Error())
		}
		if string(data) != tt.body {
			t.Errorf("wanted %s, got %s", tt.body, string(data))
		}
	}
}

var clientIPTests = []struct {
	proxy        bool
	proxyHeader  string
	clientIP     string
	headerString string
}{
	{false, "", "127.0.0.1", ""},
	{true, "X-Forwarded-For", "10.0.0.1", "10.0.0.1"},
	{true, "X-Real-Ip", "10.0.0.1", "10.0.0.1"},
	{true, "X-Forwarded-For", "172.16.0.1", "10.0.0.1, 172.16.0.1"},
}

func TestClientIP(t *testing.T) {
	for _, ct := range clientIPTests {
		proxy = ct.proxy
		proxyHeader = ct.proxyHeader
		r := &http.Request{
			RemoteAddr: "127.0.0.1:55555",
			Header:     map[string][]string{},
		}
		r.Header.Add(ct.proxyHeader, ct.headerString)
		clientIP := getClientIP(r)
		if clientIP != ct.clientIP {
			t.Errorf("getClientIP failed with %s: %v. Expected %s, got %s", ct.proxyHeader, ct.headerString, ct.clientIP, clientIP)
		}
	}
}

var hookBuildBody = `
{
    "object_kind": "build",
    "ref": "refs/heads/master",
    "tag": false,
    "before_sha": "0000000000000000000000000000000000000000",
    "sha": "655796684732a6205b4e8d329595991e8944ac8d",
    "build_id": 1,
    "build_name": "build_job",
    "build_stage": "build",
    "build_status": "success",
    "build_started_at": "2016-06-23 10:53:57 +0300",
    "build_finished_at": "2016-06-23 10:54:06 +0300",
    "build_duration": 8.648080666,
    "build_allow_failure": false,
    "project_id": 78,
    "project_name": "namespace / tetshook",
    "user": {
        "id": null,
        "name": null,
        "email": null
    },
    "commit": {
        "id": 506,
        "sha": "655796684732a6205b4e8d329595991e8944ac8d",
        "message": "Add license",
        "author_name": "author",
        "author_email": "author@author",
        "status": "success",
        "duration": 28,
        "started_at": "2016-06-22 18:52:13 +0300",
        "finished_at": "2016-06-23 10:54:06 +0300"
    },
    "repository": {
        "name": "tetshook",
        "url": "ssh://git@test/author/tetshook.git",
        "description": "",
        "homepage": "https://test/author/tetshook",
        "git_http_url": "https://test/author/tetshook.git",
        "git_ssh_url": "ssh://git@test/author/tetshook.git",
        "visibility_level": 0
    }
}
`

var hookReplacedTests = []struct {
	template string
	result   string
}{
	{"{{ref}}", "refs/heads/master"},
	{"{{branch_name}}", "master"},
	{"{{build_status}}", "success"},
	{"{{repository.git_ssh_url}}", "ssh://git@test/author/tetshook.git"},
	{"--hash {{sha}} --authorcommit {{commit.author_name}}", "--hash 655796684732a6205b4e8d329595991e8944ac8d --authorcommit author"},
}

func TestReplaceTokens(t *testing.T) {

	for _, ct := range hookReplacedTests {

		str, err := replaceTokens(hookBuildBody, ct.template)

		if err != nil {
			t.Errorf("replaceTokens failed with error %v:%s. Expected %s, got %s", err, ct.template, ct.result, str)
			continue
		}

		if str != ct.result {
			t.Errorf("replaceTokens failed with %s. Expected %s, got %s", ct.template, ct.result, str)
		}
	}

}
