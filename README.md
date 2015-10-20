# Data manager
This is a source repository for [docker](http://docker.io) image to download data for
[dictyBase](http://dictybase.org) data migration tasks. The docker container setup is based on [radial](https://github.com/radial/docs)
topology. 

## Configuration parameters

### Environment variables

`API_KEY`

[Bioportal](http://bioportal.bioontology.org/) api key, required. Open an
account and the copy the key from account information page.

`ONTOLOGY`

Comma separated list of ontologies to download from bioportal. The name of the
ontology should be ontology acronym(two or more letter capital words) as given
in bioportal page, for example as given [here](http://bioportal.bioontology.org/ontologies?filter=OBO_Foundry).

### Data volume

Optionally, it could given an extra data container with a data volume set to
```/data``` folder. In that case, the ontologies will be saved
in ```/data/ontology```.

## Usage

`Build`

```docker build --rm=true -t dictybase/data-manager .```

`Run`

```
docker run -v /data --name ontodata progrium/busybox
docker run -e API_KEY=d8535830jerekcei -e ONTOLOGY=SO,OBOREL,ECO --volumes-from ontodata dictybase/data-manager
```

