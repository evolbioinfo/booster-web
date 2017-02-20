var newick="";
var tree;

$( document ).ready(function() {
    $( "#phylocanvas" ).each(function() {
	tree = Phylocanvas.createTree('phylocanvas',{alignLabels: true,
						     showLabels : false,
						     showBootstrap: true,
						     history: {
							 parent: $(this).get(0),
							 zIndex: 1,
						     }});
    });
    updateTreeCanvas(80)
			     
    $( "#slider" ).slider({
	min:0,
	max:100,
	step: 1,
	value:80,
	change: function( event, ui ) {
	    updateTreeCanvas(ui.value);
	},
	slide: function( event, ui ) {
	    $("#collapse").html(ui.value);
	}
    });
});

/* Collapse: from 0 to 100 : collapse branches with support < collapse */
function updateTreeCanvas(collapse){
    $( "#phylocanvas" ).each(function() {
	var id = $("#phylocanvas").data("id");
	$.ajax({
	    url: "/api/analysis/"+id+"/"+collapse,
	    dataType: 'json',
	    async: true,
	    success: function(data) {
		var analysis = data;
		if(analysis.status == 2 || analysis.status == 5){
		    tree.load(analysis.collapsed);
		}
		newick=analysis.newick;
	    },
	    error: function(resultat, statut, erreur){
		console.log(erreur);
	    }
	});
    });
}

function downloadTree(id){
    if(newick!=""){
	download(newick, "bootstrap.nh", "text/plain");
    }
}
