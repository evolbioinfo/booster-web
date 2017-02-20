TBS is a new way of computing bootstrap supports in large phylogenies.

## Computing TBS via the web interface

You can submit TBS jobs on the [run](/run) page of this web interface. Only two inputs are required:

1. A reference tree file in newick format (may be gzipped)
2. A bootstrap tree file containing all the bootstrap trees (may be gzipped)

Clicking on the "run" button will launch the analysis and take you to the result page, with the following steps:

1. The analysis will be first pending, waiting for available resources;
2. Then, as soon as the analysis is running, you will see the number of bootstrap trees analyzed so far;
3. After 1 hour, the analysis is "timedout". It does not mean that it is canceled, but no more bootstrap trees will be taken into account in the support. If you want to analyze a large number of bootstrap trees, we advise to do it through [command-line](#commandline);
4. Once the analysis done, result page shows the following panels:
    1. Informations about the run (identifier, start/end time, number of bootstrap trees analyzed, output message);
    2. Links to export resulting tree to iTOL and to download resulting tree;
    3. Tree visualizer that allows to collapse branches with a support lower than the cutoff given by the slider.

## Installing a local version of the web interface

The web interface has been developped in Go, and in thus executable on any platform (Linux, MacOS, and Windows).
The only thing to do is downloading the latest release of tbs-web on [Github](https://github.com/fredericlemoine/tbs-web/releases), and run it by clinking the executable.

Then open a web browser to the url [http://localhost:8080](http://localhost:8080).

## <a name="commandline"></a>Computing TBS via command line
TBS has initially been implemented in C, and is available on [Github](https://github.com/nameoftheteam/nameofthetool).

    Usage: ./mast_like_supports -i <tree file> -b <bootstrap prefix or file> [-r <# rand shufling> -n <normalization> -@ <cpus> -s <seed> -S <stat file> -o <output tree> -v]
    Options:
      -i : Input tree file
      -b : Bootstrap prefix (e.g. boot_) or file containing several bootstrap trees
      -o : Output file (optional), default : stdout
      -@ : Number of threads (default 1)
      -s : Seed (optional)
      -S : Prints output statistics for each branch in the given output file
      -r : Number of random shuffling (for empirical norm only). Default: 10
      -n : Sets the normalization strategy to "auto" (default), "empirical" or "theoretical"
          - empirical      : Normalizes the support by the expected mast distance computed
                             using random trees (shuffled from the reference tree)
          - theoretical    : Normalizes the support by the expected mast distance computed as
                             p-1, p=the number of taxa on the lightest side of the bipartition
          - auto (default) : Will choose automatically : empirical if < 1000 taxa, theoretical
                             otherwise
      -v : Prints version (optional)
      -h : Prints this help

All information for installation is available on the github page.


## Supplementary materials
All data and workflows described in the article are available on [Github](https://github.com/fredericlemoine/tbs-workflows) as [Nextflow](https://www.nextflow.io/) workflows.
