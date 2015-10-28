package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/google/go-github/github"
	"gopkg.in/codegangsta/cli.v1"
)

var baseURL string = "http://data.bioontology.org/ontologies"

func getEtcdAPIHandler(c *cli.Context) (client.KeysAPI, error) {
	url := "http://" + c.String("etcd-host") + ":" + c.String("etcd-port")
	cfg := client.Config{
		Endpoints: []string{url},
		Transport: client.DefaultTransport,
	}
	cl, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	return client.NewKeysAPI(cl), nil
}

func downloadFromURL(url string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func saveFileFromResp(output string, resp *http.Response) error {
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

func saveObo(name string, folder string, resp *http.Response) error {
	output := filepath.Join(folder, name+".obo")
	return saveFileFromResp(output, resp)
}

func DownloadObo(apiKey string, acronym string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s/%s", baseURL, strings.ToUpper(acronym), "download"), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("apikey token=%s", apiKey))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func validateArgs(c *cli.Context) error {
	if len(c.StringSlice("bioportal")) > 1 {
		if len(c.String("api-key")) < 1 {
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
	app.Usage = "An ontology downloader from bioportal and github"
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
		cli.BoolFlag{
			Name:  "github, gh",
			Usage: "Flag to download all ontologies from dictybase github repo",
		},
		cli.StringFlag{
			Name:   "api-key",
			EnvVar: "BIOPORTAL_API_KEY",
			Usage:  "Bioportal api key",
		},
		cli.StringFlag{
			Name:  "log-level, ll",
			Usage: "Logging level",
			Value: "info",
		},
		cli.StringFlag{
			Name:   "etcd-host",
			EnvVar: "ETCD_CLIENT_SERVICE_HOST",
			Usage:  "ip address of etcd instance",
		},
		cli.StringFlag{
			Name:   "etcd-port",
			EnvVar: "ETCD_CLIENT_SERVICE_PORT",
			Usage:  "port number of etcd instance",
		},
	}
	app.Action = DownloadAction
	app.Run(os.Args)
}

func DownloadAction(c *cli.Context) {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	l := c.String("log-level")
	switch l {
	default:
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "panic":
		log.SetLevel(log.PanicLevel)
	}

	if err := validateArgs(c); err != nil {
		log.WithFields(log.Fields{
			"error": "download",
		}).Fatal(err)
	}
	// create if the folder does not exist
	_, err := os.Stat(c.String("folder"))
	if os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"folder": c.String("folder"),
		}).Info("creating output folder")
		os.MkdirAll(c.String("folder"), 0744)
	}
	for _, onto := range c.StringSlice("bioportal") {
		resp, err := DownloadObo(c.String("api-key"), onto)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  "download",
				"source": "bioportal",
			}).Fatal(err)
		}
		if err := saveObo(normalizeName(onto), c.String("folder"), resp); err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"source": "bioportal",
				"file":   onto,
			}).Fatal("Unable to download")
		}
		log.WithFields(log.Fields{
			"source": "bioportal",
			"file":   onto,
		}).Info("Downloaded file")
	}

	if c.IsSet("github") {
		client := github.NewClient(nil)
		_, ghdir, _, err := client.Repositories.GetContents("dictyBase", "migration-data", "ontologies", nil)
		if err != nil {
			log.WithFields(log.Fields{
				"source": "github",
			}).Fatal(err)
		}
		for _, cont := range ghdir {
			resp, err := downloadFromURL(*cont.DownloadURL)
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"source": "github",
					"file":   *cont.Name,
				}).Fatal("Unable to download")
			}
			output := filepath.Join(c.String("folder"), filepath.Base(*cont.DownloadURL))
			if err := saveFileFromResp(output, resp); err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"source": "github",
					"file":   output,
				}).Fatal("Unable to save")
			}
			log.WithFields(log.Fields{
				"source": "github",
				"file":   output,
			}).Info("Downloaded file")
		}
	}

	// check if etcd host is given
	if len(c.String("etcd-host")) > 1 && len(c.String("etcd-port")) > 1 {
		api, err := getEtcdAPIHandler(c)
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		_, err = api.Create(context.Background(), "/migration/download", "complete")
		if err != nil {
			log.WithFields(log.Fields{
				"type": "etcd-client",
			}).Fatal(err)
		}
		log.WithFields(log.Fields{
			"type": "etcd-client",
		}).Info("added download completion in etcd")
	}
}
