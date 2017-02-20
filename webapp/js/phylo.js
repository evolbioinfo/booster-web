$( document ).ready(function() {
    var tree = Phylocanvas.createTree('phylocanvas',{alignLabels: true,showLabels : false, showBootstrap: true});
    var id = $("#phylocanvas").data("id");

    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
	async: true,
	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		tree.load(analysis.newick);
	    }
	},
	error: function(resultat, statut, erreur){
	    console.log(erreur);
	}
    });
});
