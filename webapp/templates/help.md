BOOSTER is a new way of computing bootstrap supports in large phylogenies.

## Computing transfer supports via the web interface

You can submit BOOSTER jobs on the [run](/run) page of this web interface. Only two inputs are required:

1. A reference tree file in newick format (may be gzipped)
2. A bootstrap tree file containing all the bootstrap trees (may be gzipped)

Clicking on the "run" button will launch the analysis and take you to the result page, with the following steps:

1. The analysis will be first pending, waiting for available resources;
2. Then, as soon as the analysis is running, you will be redirected to a waiting page;
3. After 1 hour, the analysis is "timedout". It does not mean that it is canceled, but no more bootstrap trees will be taken into account in the support. If you want to analyze a large number of bootstrap trees, we advise to do it through [command-line](#commandline);
4. Once the analysis done, result page shows the following panels:
    1. Informations about the run (identifier, start/end time, number of bootstrap trees analyzed, output message);
    2. Links to export resulting tree to iTOL and to download resulting tree;
    3. Tree visualizer that allows to highlight branches with a support greater than the cutoff given by the slider.

## Installing a local version of the web interface

The web interface has been developped in Go, and in thus executable on any platform (Linux, MacOS, and Windows).
The only thing to do is downloading the latest release of BoosterWeb on [Github](https://github.com/fredericlemoine/booster-web/releases), and run it by clinking the executable.

Then open a web browser to the url [http://localhost:8080](http://localhost:8080).

## <a name="commandline"></a>Computing transfer supports via command line
BOOSTER has initially been implemented in C, and is available on [Github](https://github.com/nameoftheteam/nameofthetool).

```
Usage: booster -i <tree file> -b <bootstrap prefix or file> [-@ <cpus>  -S <stat file> -o <output tree> -v]
Options:
	-i, --input      : Input tree file
	-b, --boot       : Bootstrap tree file (1 file containing all bootstrap trees)
	-o, --out        : Output file (optional), default : stdout
	-@, --num-threads: Number of threads (default 1)
	-S, --stat-file  : Prints output statistics for each branch in the given output file
	-a, --algo       : tbe or fbp (default tbe
	-q, --quiet      : Does not print progress messages during analysis
	-v, --version    : Prints version (optional)
	-h, --help       : Prints this help
```

All information for installation is available on the github page.


## Supplementary materials
All data and workflows described in the article are available on [Github](https://github.com/evolbioinfo/booster-workflows) as [Nextflow](https://www.nextflow.io/) workflows.
