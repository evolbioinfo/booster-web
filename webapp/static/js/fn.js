function updateNbootInput(val) {
    document.getElementById('nboottext').innerHTML=val; 
}

$( document ).ready(function() {
    // When sequence file is selected,
    // Tree files are cleared
    $("#refseqs").change(function (){
	$("#reftree").val('');
	$("#boottrees").val('');
    });
    $("#reftree").change(function (){
	$("#refseqs").val('');
    });
    $("#boottrees").change(function (){
	$("#refseqs").val('');
    });
});
