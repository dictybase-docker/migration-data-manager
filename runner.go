package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var folder string = "/data/ontology"
var baseURL string = "http://data.bioontology.org/ontologies"

func saveObo(name string, folder string, resp *http.Response) error {
	output := filepath.Join(folder, name+".obo")
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func DownloadObo(apiKey string, acronym string) *http.Response {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", baseURL, acronym, "download"), nil)
	if err != nil {
		log.Fatalf("Unable to make new request error: %s\n", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("apikey token=%s", apiKey))
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Problem in http request error: %s", err)
	}
	return resp
}

func validateEnv() error {
	if len(os.Getenv("API_KEY")) == 0 {
		return fmt.Errorf("%s env is not set", "API_KEY")
	}
	if len(os.Getenv("ONTOLOGY")) == 0 {
		return fmt.Errorf("%s env is not set", "ONTOLOGY")
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(name)
}

func ontologies() []string {
	all := os.Getenv("ONTOLOGY")
	if strings.ContainsAny(all, ",") {
		return strings.Split(all, ",")
	}
	return []string{all}
}

func main() {
	// name of output folder
	if len(os.Args) == 2 {
		folder = os.Args[1]
	}
	// create if the folder does not exist
	_, err := os.Stat(folder)
	if os.IsNotExist(err) {
		os.MkdirAll(folder, 0744)
	}

	err = validateEnv()
	if err != nil {
		log.Fatal(err)
	}
	for _, onto := range ontologies() {
		resp := DownloadObo(os.Getenv("API_KEY"), onto)
		err = saveObo(normalizeName(onto), folder, resp)
		if err != nil {
			log.Fatalf("Unable to save %s obo error: %s", onto, err)
		}
		log.Printf("Downloaded ontology %s\n", onto)
	}
}
