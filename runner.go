package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/codegangsta/cli.v1"
)

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
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", baseURL, strings.ToUpper(acronym), "download"), nil)
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

func validateArgs(c *cli.Context) error {
	if len(c.StringSlice("bioportal")) > 0 {
		if !c.IsSet("api-key") {
			return fmt.Errorf("bioportal api-key is not set")
		}
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(name)
}

func main() {
	app := cli.NewApp()
	app.Name = "downloader"
	app.Usage = "Download obo ontology from bioportal and github"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "folder, f",
			Usage: "Download folder",
			Value: "/data/ontology",
		},
		cli.StringSliceFlag{
			Name:  "bioportal, bp",
			Usage: "Name of bioportal ontologies",
			Value: &cli.StringSlice{""},
		},
		cli.StringSliceFlag{
			Name:  "github, gh",
			Usage: "Name of github ontologies",
			Value: &cli.StringSlice{""},
		},
		cli.StringFlag{
			Name:  "api-key",
			Usage: "Bioportal api key",
		},
	}
	app.Action = DownloadAction
	app.Run(os.Args)
}

func DownloadAction(c *cli.Context) {
	if err := validateArgs(c); err != nil {
		log.Fatal(err)
	}
	// create if the folder does not exist
	_, err := os.Stat(c.String("folder"))
	if os.IsNotExist(err) {
		fmt.Printf("creating output folder %s", c.String("folder"))
		os.MkdirAll(c.String("folder"), 0744)
	}
	for _, onto := range c.StringSlice("bioportal") {
		resp := DownloadObo(c.String("api-key"), onto)
		err = saveObo(normalizeName(onto), c.String("folder"), resp)
		if err != nil {
			log.Fatalf("Unable to save %s obo error: %s", onto, err)
		}
		log.Printf("Downloaded ontology %s\n", onto)
	}
}
