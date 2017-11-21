var newick="";
var tree;
var layout="radial";
var algorithm="tbe";
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
    updateTreeCanvas();

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
	    layout=$(this).val();
	});
	updateTreeCanvas();
    });
    $ ("#algorithm").change(function(){
	$( "#algorithm option:selected" ).each(function() {
	    algorithm=$(this).val();
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
	    url: "/api/image/"+id+"/"+collapse+"/"+layout+"/"+algorithm+"/svg",
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

function downloadTBENormTree(id){
    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.tbenormtree, "boosterweb_tbe_norm.nh", "text/plain");
	    }
	}
    });
}

function downloadTBERawTree(id){
    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.tberawtree, "boosterweb_tbe_raw.nh", "text/plain");
	    }
	}
    });
}

function downloadFBPTree(id){
    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.fbptree, "boosterweb_fbp.nh", "text/plain");
	    }
	}
    });
}

function downloadAlignment(id){
    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.align, "alignment.fa", "text/plain");
	    }
	}
    });
}

function downloadLogs(id){
    $.ajax({
	url: "/api/analysis/"+id,
	dataType: 'json',
 	async: true,
 	success: function(data) {
	    var analysis = data;
	    if(analysis.status == 2 || analysis.status == 5){
		download(analysis.tbelogs, "boosterweb_tbe_logs.txt", "text/plain");
	    }
	}
    });
}
