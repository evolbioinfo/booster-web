function updateNbootInput(val) {
    document.getElementById('nboottext').innerHTML=val; 
}

var timer=0;

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

    timer=$('meta[http-equiv=refresh]').attr("content");
    timerfunc();
    setInterval(timerfunc, 1000);

    $("#runnamebutton").click(function(event) {
	event.preventDefault();
	$.ajax({
	    url: "/api/randrunname",
	    dataType: 'text',
 	    async: true,
	    success: function(data) {
		var runname = data;
		$("#runname").val(runname);
	    }
	});
    });
});

function timerfunc(){
    $("#timeout").text(timer);
    timer=timer-1;
}
