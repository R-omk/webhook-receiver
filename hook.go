package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

func hookHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r)
	params := mux.Vars(r)
	id := params["id"]
	clientIP := getClientIP(r)
	if clientIP != strings.Split(r.RemoteAddr, ":")[0] {
		log.Printf("Received hook for id '%s' from %s on %s\n", id, clientIP, r.RemoteAddr)
	} else {
		log.Printf("Received hook for id '%s' from %s\n", id, r.RemoteAddr)
	}
	rb, err := NewRunBook(id)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	remoteIP := net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	if !rb.AddrIsAllowed(remoteIP) {
		log.Printf("Hook id '%s' is not allowed from %v\n", id, r.RemoteAddr)
		http.Error(w, "Not authorized.", http.StatusUnauthorized)
		return
	}
	interoplatePOSTData(rb, r)
	response, err := rb.execute()
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), 500)
		return
	}
	log.Println(response)
	if echo {
		data, err := json.MarshalIndent(response, "", "  ")

		if err != nil {
			log.Println(err.Error())
		}

		w.Write(data)
	}
}

func interoplatePOSTData(rb *runBook, r *http.Request) {
	if r.ContentLength == 0 {
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil && err != io.EOF {
		log.Fatal(err)
		return
	}
	defer r.Body.Close()
	stringData := string(data[:r.ContentLength])
	log.Println("POST DATA", stringData)
	for i := range rb.Scripts {
		for j := range rb.Scripts[i].Args {
			rb.Scripts[i].Args[j] = strings.Replace(rb.Scripts[i].Args[j], "{{POST}}", stringData, -1)

			rb.Scripts[i].Args[j], err = replaceTokens(stringData, rb.Scripts[i].Args[j])
			if err != nil {
				log.Fatal(err)
				return
			}
		}
	}
}

func getClientIP(r *http.Request) string {
	remoteIP := strings.Split(r.RemoteAddr, ":")[0]
	if !proxy {
		return remoteIP
	}
	headerVal := r.Header.Get(proxyHeader)
	// proxies can chain upstream client addresses- take only the closest (last) address
	// http://en.wikipedia.org/wiki/X-Forwarded-For
	upstreams := strings.Split(headerVal, ", ")
	return upstreams[len(upstreams)-1]
}

func replaceTokensRecur(m map[string]interface{}, keybase string, str string) (res string, err error) {
	res = str
	for key, value := range m {

		key = keybase + key
		switch t := value.(type) {
		default:
		case string:
			res = strings.Replace(res, "{{"+key+"}}", t, -1)
			switch key {
			case "ref":
				re := regexp.MustCompile("refs/heads/(.*)")
				matches := re.FindStringSubmatch(t)
				if len(matches) > 0 {
					res = strings.Replace(res, "{{branch_name}}", matches[1], -1)
				}

				re = regexp.MustCompile("refs/tags/(.*)")
				matches = re.FindStringSubmatch(t)
				if len(matches) > 0 {
					res = strings.Replace(res, "{{tag_name}}", matches[1], -1)
				}

			}
		case map[string]interface{}:
			res, err = replaceTokensRecur(t, key+".", res)
			if err != nil {
				return
			}
		}

	}

	return
}

func replaceTokens(js string, target string) (res string, err error) {
	err = nil
	res = target
	var x map[string]interface{}
	err = json.Unmarshal([]byte(js), &x)

	if err != nil {
		return
	}

	prefix := ""
	res, err = replaceTokensRecur(x, prefix, res)

	return
}
