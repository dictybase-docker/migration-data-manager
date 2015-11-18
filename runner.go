package main

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/google/go-github/github"
	"gopkg.in/codegangsta/cli.v1"
)

var baseURL string = "http://data.bioontology.org/ontologies"
var mURL string = "https://northwestern.box.com/shared/static/t35zifjta5l8nk3mxminfaff1dlhfitz.bz2"
var gpadURL string = "http://www.ebi.ac.uk/QuickGO/GAnnotation?format=gpa&limit=-1&db=dictyBase"
var purl string = "http://purl.obolibrary.org/obo"

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

func DownloadObo(name string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s.obo", purl, name), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func validateArgs(c *cli.Context) error {
	if len(c.String("api-key")) < 1 {
		return fmt.Errorf("bioportal api-key is not set")
	}
	return nil
}

func normalizeName(name string) string {
	return strings.ToLower(name)
}

func main() {
	app := cli.NewApp()
	app.Name = "downloader"
	app.Usage = "A downloader for migration data"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "download-folder, df",
			Usage: "Download folder for migration data",
			Value: "/data",
		},
		cli.BoolFlag{
			Name:  "migration-data, md",
			Usage: "Flag to download and extract migration data from box",
		},
		cli.StringSliceFlag{
			Name:  "obo",
			Usage: "Name of ontologies to download using purl url",
			Value: &cli.StringSlice{""},
		},
		cli.BoolFlag{
			Name:  "github, gh",
			Usage: "Flag to download all ontologies from dictybase github repo",
		},
		cli.BoolFlag{
			Name:  "gpad",
			Usage: "Flag to download dictybase gpad annotations",
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

	CreateDownloadFolder(c)
	wg := new(sync.WaitGroup)
	if len(c.String("obo")) > 2 {
		wg.Add(1)
		go OboAction(c, wg)
	}

	if c.IsSet("github") {
		wg.Add(1)
		go GithubAction(c, wg)
	}

	if c.IsSet("migration-data") {
		wg.Add(1)
		go MigrationAction(c, wg)
		log.WithFields(log.Fields{
			"source": "box",
			"file":   "migration-data.tar.bz2",
		}).Info("extracted migration file")
	}
	if c.IsSet("gpad") {
		wg.Add(1)
		go DownloadGAF(c, wg)
	}

	wg.Wait()

	// check if etcd host is given
	if len(c.String("etcd-host")) > 1 && len(c.String("etcd-port")) > 1 {
		WriteToEtcd(c)
	}
}

func WriteToEtcd(c *cli.Context) {
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

func untarGunzip(c *cli.Context, file string) {
	folder := c.String("download-folder")
	reader, err := os.Open(file)
	defer reader.Close()
	if err != nil {
		log.WithFields(log.Fields{
			"file":  file,
			"error": "open file",
		}).Fatal(err)
	}
	archive := bzip2.NewReader(reader)
	tarReader := tar.NewReader(archive)
	if err != nil {
		log.WithFields(log.Fields{
			"error": "open tar file",
		}).Fatal(err)
	}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			log.WithFields(log.Fields{
				"error": "reading tar member",
			}).Fatal(err)
		}
		path := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(folder, path), os.FileMode(header.Mode)); err != nil {
				log.WithFields(log.Fields{
					"error": "creating directory",
				}).Fatal(err)
			}
		case tar.TypeReg:
			writer, err := os.Create(filepath.Join(folder, path))
			defer writer.Close()
			if err != nil {
				log.WithFields(log.Fields{
					"error": "opening file for writing",
				}).Fatal(err)
			}
			_, err = io.Copy(writer, tarReader)
			if err != nil {
				log.WithFields(log.Fields{
					"error": "writing member file",
				}).Fatal(err)
			}
		default:
			log.WithFields(log.Fields{
				"error": "Unknown action",
				"file":  path,
				"type":  header.Typeflag,
			}).Info("No action taken")
		}
	}
}

func OboAction(c *cli.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	CreateOntologyFolder(c)
	for _, onto := range c.StringSlice("obo") {
		resp, err := DownloadObo(onto)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  "download",
				"source": "purl",
			}).Fatal(err)
		}
		if err := saveObo(onto, filepath.Join(c.String("download-folder"), "ontology"), resp); err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"source": "purl",
				"file":   onto,
			}).Fatal("Unable to download")
		}
		log.WithFields(log.Fields{
			"source": "purl",
			"file":   onto,
		}).Info("Downloaded file")
	}

}

func CreateOntologyFolder(c *cli.Context) {
	// create if the folder does not exist
	_, err := os.Stat(filepath.Join(c.String("download-folder"), "ontology"))
	if os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"folder": c.String("download-folder"),
		}).Info("creating output folder")
		os.MkdirAll(filepath.Join(c.String("download-folder"), "ontology"), 0744)
	}
}

func CreateFolder(folder string) {
	// create if the folder does not exist
	_, err := os.Stat(folder)
	if os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"folder": folder,
		}).Info("creating output folder")
		os.MkdirAll(folder, 0744)
	}
}

func CreateDownloadFolder(c *cli.Context) {
	// create if the folder does not exist
	_, err := os.Stat(c.String("download-folder"))
	if os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"folder": c.String("download-folder"),
		}).Info("creating output folder")
		os.MkdirAll(c.String("folder"), 0744)
	}
}

func GithubAction(c *cli.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	CreateOntologyFolder(c)
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
		output := filepath.Join(c.String("download-folder"), "ontology", filepath.Base(*cont.DownloadURL))
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

func MigrationAction(c *cli.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	resp, err := downloadFromURL(mURL)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"source": "box",
			"url":    mURL,
		}).Fatal("Unable to download")
	}
	output := filepath.Join(c.String("download-folder"), "migration-data.tar.bz2")
	if err := saveFileFromResp(output, resp); err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"source": "box",
			"file":   "migration-data.tar.bz2",
		}).Fatal("Unable to save")
	}
	log.WithFields(log.Fields{
		"source": "box",
		"file":   "migration-data.tar.bz2",
	}).Info("Downloaded file")
	untarGunzip(c, output)
}

func DownloadGAF(c *cli.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	CreateFolder(filepath.Join(c.String("download-folder"), "gpad"))
	resp, err := downloadFromURL(gpadURL)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"source": "ebi",
			"url":    gpadURL,
		}).Fatal("Unable to download")
	}
	output := filepath.Join(c.String("download-folder"), "gpad", "dicty.gpad")
	if err := saveFileFromResp(output, resp); err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"source": "ebi",
			"file":   "dicty.gpad",
		}).Fatal("Unable to save")
	}
	log.WithFields(log.Fields{
		"source": "ebi",
		"file":   "dicty.gpad",
	}).Info("Downloaded file")
}
