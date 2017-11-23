BOOSTER is a new way of computing bootstrap supports in large phylogenies.

## Computing transfer supports via the web interface

Two kinds of jobs may be submited to BOOSTER-WEB:

1. Whole phylogenetic analysis;
2. Bootstrap support computation.

### Whole phylogenetic analysis

If you do not have reference and bootstrap trees already computed, you may just submit an alignment file to the [run](/new) input form (may be gzipped), in the "input sequence" field. 
The alignment file may be in Fasta, Phylip or Nexus format.

In that case, you may choose between two workflows to run:
1. PhyML-SMS;
2. FastTree.

	These two workflows are installed and launche on the Instut Pasteur [Galaxy](https://galaxy.pasteur.fr/) server. 

They constist of the following steps:

1. Tree inference, two possibilities:
    a. Model selection + tree inference using [Phyml-SMS](http://www.atgc-montpellier.fr/phyml-sms/) or
	b. Reference + Bootstrap Tree reconstructions using [FastTree](http://www.microbesonline.org/fasttree/)
2. Bootstrap support computation using [booster](https://github.com/evolbioinfo/booster/).

### Bootrap support computation

You may also directly submit BOOSTER jobs on the [run](/new) page. In that case, only two inputs are required:

1. A reference tree file in newick format (may be gzipped)
2. A bootstrap tree file containing all the bootstrap trees (may be gzipped)

### Workflow

Please note that if an alignment  file (Fasta, Phylip or Nexus) is provided, no tree file will be taken into account.

Clicking the "run" button will launch one of the analyses, and will take you to the result page, with the following steps:

1. The analysis will be first pending, waiting for available resources;
2. Then, as soon as the analysis is running, you will be redirected to a waiting page;
3. After 5 days, the analysis is "timedout". If you want to analyze a large number of large bootstrap trees, we advise to do it through [command-line](#commandline);
4. Once the analysis done, result page shows the following panels:
        1. Informations about the run (identifier, start/end time, number of bootstrap trees analyzed, output message);
    2. Links to download results:
	   1. Alignment file (only for whole phylogenetic analysis);
	   2. Tree with FBP (classical) supports;
	   2. Tree with TBE (transfer bootstrap) normalized supports (download newick format or upload to iTOL);
	   3. Tree with branch labels formatted as following: "Branch ID|Average transfer Distance|Depth" (download newick format or upload to iTOL);
	   4. Booster log file with 2 parts:
		  1. Transfer score per taxa (2 columns, "Taxon : Transfer Score");
		  2. Most transfered taxa per branch (4 columns: Branch Id\tp\tAverage distance\tsemicolumn separated list of most transfered taxa with their respective transfer index)
    3. Tree visualizer that highlights branches with a support greater than the cutoff given by the slider.

## Generating reference and bootstrap trees

Ig you want to generate reference and bootstrap trees with other means, you may do it using the following commands:

* PhyML: Input file: alignment, Phylip format
```bash
phyml -i align.phy -d nt -b 100 -m GTR -f e -t e -c 6 -a e -s SPR -o tlr 
# Output Reference tree: align.phy_phyml_tree.txt
# Output Bootstrap trees: align.phy_phyml_boot_trees.txt
```

* RAxML: Standard bootstrap. Input file: alignment, Phylip format
```bash
# Infer reference tree
raxmlHPC -m GTRGAMMA -p $RANDOM -s align.phy -n REF
# Infer bootstrap trees
raxmlHPC -m GTRGAMMA -p $RANDOM -b $RANDOM -# 100 -s align.phy -n BOOT
# Output Reference tree: RAxML_bestTree.REF
# Output Bootstrap trees: RAxML_bootstrap.BOOT
```

* RAxML: Rapid bootstrap. Input file: alignment, Phylip format
```bash
# Infer reference tree + bootstrap trees
raxmlHPC -f a -m GTRGAMMA -c 4 -s align.phy -n align -T 4 -p $RANDOM -x $RANDOM -# 100
# Output Reference tree: RAxML_bestTree.align
# Output Bootstrap trees: RAxML_bootstrap.align
```

* FastTree: You will need to generate bootstrap alignments (Phylip format), with [goalign](https://github.com/fredericlemoine/goalign) for example. Input file: alignment (Phylip or Fasta format)
```bash
# Build bootstrap alignments
goalign build seqboot -i align.phy -p -n 100 -o boot -S
# Infer reference tree
FastTree -nt -gtr align.phy > ref.nhx
# Infer bootstrap trees
cat boot*.ph | FastTree -nt -n 100 -gtr > boot.nhx
# Output Reference tree: ref.nhx
# Output Bootstrap trees: boot.nhx
```

* IQ-TREE : Booster supports and ultrafast bootstrap. Input file: alignment (Phylip format)
```
# Infer Reference tree + ultrafast bootstrap trees
iqtree-omp -wbt -s align.phy -m GTR -bb 100 -nt 5
# Output Reference tree: align.phy.treefile
# Output Bootstrap trees: align.phy.ufboot
```

## Example dataset

You may try BOOSTER-WEB with the following trees inferred from primate nt alignment taken from ["The Phylogenetic Handbook"](http://www.cambridge.org/catalogue/catalogue.asp?isbn=9780521877107):

* Reference tree: [.nw.gz](/static/files/primates/ref.nw.gz)
* 1000 Bootstrap trees: [.nw.gz](/static/files/primates/boot.nw.gz)

If you want to also infer reference and bootstrap trees prior to using BOOSTER-WEB:

* Original alignment: [.phy.gz](/static/files/primates/DNA_primates.phy)
* Nextflow workflow: [.nf](/static/files/primates/primates.nf) (`nextflow run primates.nf` to run it)


After computing TBE and FBP using BOOSTER-WEB, you should obtain trees like the followings:

* TBE
![TBE](/static/files/primates/TBE.png)

* FBP
![FBP](/static/files/primates/FBP.png)


## Installing a local version of the web interface

The web interface has been developped in Go, and in thus executable on any platform (Linux, MacOS, and Windows).
The only thing to do is downloading the latest release of BoosterWeb on [Github](https://github.com/fredericlemoine/booster-web/releases), and run it by clicking the executable.

Then open a web browser to the url [http://localhost:8080](http://localhost:8080).

## <a name="commandline"></a>Computing transfer supports via command line
BOOSTER is also available as a standalone executable (implemented in C). Sources and binaries are available on [Github](https://github.com/evolbioinfo/booster).

```
Usage: ./booster -i <ref tree file (newick)> -b <bootstrap tree file (newick)> [-d <dist_cutoff> -r <raw distance output tree file> -@ <cpus>  -S <stat file> -o <output tree> -v]
Options:
      -i : Input tree file
      -b : Bootstrap tree file (1 file containing all bootstrap trees)
      -a, --algo  : bootstrap algorithm, tbe (transfer bootstrap) or fbp (Felsenstein bootstrap) (default tbe)
      -o : Output file (optional), default : stdout
      -r, --out-raw : Output file (only with tbe, optional) with raw transfer distance as support values in the form of
                       id|avgdist|depth, default : none
      -@ : Number of threads (default 1)
      -S : Prints output logs in the given output file (average raw min transfer distance per branches, and average
      	   transfer index per taxa)
      -c, --count-per-branch : Prints individual taxa moves for each branches in the log file (only with -S and -a tbe)
      -d, --dist-cutoff: Distance cutoff to consider a branch for moving taxa computation (tbe only, default 0.3)
      -q, --quiet : Does not print progress messages during analysis
      -v : Prints version (optional)
      -h : Prints this help
```

All information for installation and usage is available on the [GitHub page](https://github.com/evolbioinfo/booster).

## <a name="note"></a>Note
PhyML can also directly compute TBE supports (beta). To do so, you will need to download and install PhyML from its [github repository](https://github.com/stephaneguindon/phyml/):

```bash
phyml -i align.phy -d nt -b 100 --tbe -m GTR -f e -t e -c 6 -a e -s SPR -o tlr 
# Output tree with supports: align.phy_phyml_tree.txt
```


## <a name="suppmat"></a>Supplementary materials
Workflows described in the article are available on [Github](https://github.com/evolbioinfo/booster-workflows) as [Nextflow](https://www.nextflow.io/) workflows, and data are located on the [release](https://github.com/evolbioinfo/booster-workflows/releases/latest) page.
