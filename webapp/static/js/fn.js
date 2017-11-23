function updateNbootInput(val) {
    document.getElementById('nboottext').innerHTML=val; 
}

$( document ).ready(function() {
    // When sequence file is selected,
    // Tree files are cleared
    $("#refalign").change(function (){
	$("#reftree").val('');
	$("#boottrees").val('');
    });
    $("#reftree").change(function (){
	$("#refalign").val('');
    });
    $("#boottrees").change(function (){
	$("#refalign").val('');
    });
    
    $("#nboottext").html($("#nboot").val());
});
