# Data manager
This is a source repository for [docker](http://docker.io) image to download data for
[dictyBase](http://dictybase.org) data migration tasks. The docker container setup is based on [radial](https://github.com/radial/docs)
topology. 

## Usage

`Build`

```docker build --rm=true -t dictybase/data-manager .```

`Run`

### Command line

```
docker run --rm dictybase/data-manager app -h

NAME:
   downloader - Download obo ontology from bioportal and github

USAGE:
   downloader [global options] command [command options] [arguments...]

VERSION:
   1.0.0

COMMANDS:
   help, h	Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   --folder, -f '/data/ontology'				Download folder
   --bioportal, --bp '--bioportal option --bioportal option'	Name of bioportal ontologies
   --github, --gh '--github option --github option'		Name of github ontologies
   --api-key 							Bioportal api key
   --help, -h							show help
   --version, -v						print the version
```

By default, it expects an extra data container with a data volume set to
`/data` folder. In that case, the ontologies will be saved
in `/data/ontology`.

### Example

```
docker run -v /data --name ontodata progrium/busybox
docker run --rm dictybase/data-manager --volumes from ontodata \
    --api-key d8535830jerekcei --bp so --bp eco 
```




