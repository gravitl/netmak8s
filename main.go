package main

import (
	"bytes"
	"strconv"
	"log"
	"io/ioutil"
	"errors"
	"os"
	"net/http"
	"os/exec"
	"golang.org/x/build/kubernetes/api"
	"encoding/json"
)

//Start MongoDB Connection and start API Request Handler
func main() {

	serverurl, apiappend, secret, network := getConnDetails()

	nmdns, err := getNetmakerDNS(serverurl, apiappend, secret)
	if err != nil {
		log.Fatal(err)
	}
	k8sdns, err := getK8SDNS()
	if err != nil {
		log.Fatal(err)
	}
	err = compareAndPush(nmdns, k8sdns, serverurl, network, secret, apiappend)
	if err != nil {
		log.Println(err)
	}
	log.Println("success")
}

func getConnDetails() (string, string, string, string) {
        serverurl := "http://localhost:8081"
        if os.Getenv("SERVER_API_URL") != "" {
                serverurl = os.Getenv("SERVER_API_URL")
        }
        network := ""
        if os.Getenv("NETWORK") != "" {
                network = os.Getenv("NETWORK")
        }
        secret := "secretkey"
        if os.Getenv("SECRET") != "" {
                secret = os.Getenv("SECRET")
        }
	apiappend := ""
	if network == "" {
		apiappend = "/api/dns"
	} else {
		apiappend = "/api/dns/adm/" + network
	}
	return serverurl, apiappend, secret, network
}

func getNetmakerDNS(serverurl string, apiappend string, secret string) ([]DNSEntry, error) {
	var dnsentries []DNSEntry
	req, err := http.NewRequest("GET", serverurl + apiappend, nil)
	if err != nil {
		return dnsentries, err
	}
	req.Header.Set("Authorization", "Bearer " + secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return dnsentries, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
	        err = json.Unmarshal(bodyBytes,&dnsentries)
	} else {
		return dnsentries, errors.New("Bad response from server "+strconv.Itoa(resp.StatusCode))
	}
	return dnsentries, err
}

func getK8SDNS() ([]api.Service, error) {
	var services []api.Service
	out, err := exec.Command("kubectl","get","svc","-A","-o","jsonpath={.items}").Output()
	err = json.Unmarshal(out,&services)
	return services, err
}

func compareAndPush(nmdns []DNSEntry, k8sdns []api.Service, serverurl string, network string, secret string, apiappend string) error {
	var err error
	for _, service := range k8sdns {
		dnsname := service.ObjectMeta.Name +"."+service.ObjectMeta.Namespace
		log.Println(dnsname)
		found := false
		for _, entry := range nmdns {
			if entry.Name == dnsname {
				if entry.Address != service.Spec.ClusterIP {
					entry.Address = service.Spec.ClusterIP
					err = createUpdateDNS("update",
								serverurl,
								"/api/dns/"+network+"/"+entry.Name,
								secret,
								entry)
					if err != nil {
						log.Println(err)
					}
				}
				found = true
				break
			}
		}
		if found == false {
			newentry := DNSEntry{
					Address: service.Spec.ClusterIP,
					Name: dnsname,
					Network: network,
				}
			err = createUpdateDNS("create",
						serverurl,
						"/api/dns/"+network,
						secret,
						newentry)
		}
		if err != nil {
			log.Println(err)
		}
	}
	return err
}

func createUpdateDNS(crudop string, serverurl string, apiappend string, secret string, entry DNSEntry) error {
	entrybytes, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", serverurl + apiappend,  bytes.NewBuffer(entrybytes))
	if crudop == "create" {
		req, err = http.NewRequest("POST", serverurl + apiappend,  bytes.NewBuffer(entrybytes))
	}
        if err != nil {
                return err
        }
        req.Header.Set("Authorization", "Bearer " + secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
                err = errors.New("Bad response from server "+strconv.Itoa(resp.StatusCode))
        }
        return err
}

type DNSEntry struct {
	Address string `json:"address" bson:"address" validate:"required,ip"`
	Name    string `json:"name" bson:"name" validate:"required,name_unique,min=1,max=192"`
	Network string `json:"network" bson:"network" validate:"network_exists"`
}


