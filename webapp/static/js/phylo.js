var newick="";
var tree;
var layout="radial";
var collapse=80;

$( document ).ready(function() {
    // $( "#phylocanvas" ).each(function() {
    //tree = Phylocanvas.createTree('phylocanvas',{alignLabels: true,
    //					     showLabels : false,
    //					     showBootstrap: true,
    //					     history: {
    //						 parent: $(this).get(0),
    //						 zIndex: 1,
    //					     }});
    //});
    updateTreeCanvas(80)

    $( "#slider" ).slider({
	min:0,
	max:100,
	step: 1,
	value:collapse,
	change: function( event, ui ) {
	    collapse=ui.value;
	    updateTreeCanvas();
	},
	slide: function( event, ui ) {
	    $("#collapse").html(ui.value);
	}
    });

    $ ("#layout").change(function(){
	$( "#layout option:selected" ).each(function() {
	    layout=$(this).text();
	});
	updateTreeCanvas();
    });
});

/* Collapse: from 0 to 100 : collapse branches with support < collapse */
function updateTreeCanvas(){
    $( "#phylocanvas" ).each(function() {
	var id = $(this).data("id");
	var elt = $(this);
	$.ajax({
	    url: "/api/image/"+id+"/"+collapse+"/"+layout+"/svg",
	    dataType: 'text',
	    async: true,
	    success: function(data) {
		elt.find("img").attr('src','data:image/svg+xml,' + data);
	    },
	    error: function(resultat, statut, erreur){
		console.log(erreur);
	    }
	});
    });
}

function downloadTree(id){
    $.ajax({
	url: "/api/analysis/"+id+"/0",
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.result, "bootstrap.nh", "text/plain");
	    }
	}
    });
}
