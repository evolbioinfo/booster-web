var stats;

$( document ).ready(function() {
    $.ajax({
	url: "/api/stat/json",
	dataType: 'json',
	async: true,
	success: function(data) {
	    stats = data;
	    initDayDistHistos();
	    initMonthDistHistos();
	    initYearDistHistos();
	},
	error : function(resultat, statut, erreur){
	    console.log(erreur);
       }
    });
});


function initDayDistHistos(){
    var days=stats["days"];
    var distdays=stats["distdays"];

    for (i = 0; i < distdays.length; i++) {
	distdays[i] = distdays[i]/1000;
    }

    distdays.unshift("Distance");

    chart = c3.generate({
	size: {
            height: 180,
	},
	bindto: '#statdistday',
	data: {
            columns: [
		distdays,
            ],
            type : 'bar',
	},
	axis: {
            x: {
		type: 'category',
		categories: days,
            }
	}
    });    
}

function initMonthDistHistos(){
    var months=stats["months"];
    var distmonths=stats["distmonths"];

    for (i = 0; i < distmonths.length; i++) {
	distmonths[i] = distmonths[i]/1000;
    }

    distmonths.unshift("Distance");

    chart = c3.generate({
	size: {
            height: 180,
	},
	bindto: '#statdistmonth',
	data: {
            columns: [
		distmonths,
            ],
            type : 'bar',
	},
	axis: {
            x: {
		type: 'category',
		categories: months,
            }
	}
    });    
}

function initYearDistHistos(){
    var years=stats["years"];
    var distyears=stats["distyears"];

    for (i = 0; i < distyears.length; i++) {
	distyears[i] = distyears[i]/1000;
    }

    distyears.unshift("Distance");

    chart = c3.generate({
	size: {
            height: 180,
	},
	bindto: '#statdistyear',
	data: {
            columns: [
		distyears,
            ],
            type : 'bar',
	},
	axis: {
            x: {
		type: 'category',
		categories: years,
            }
	}
    });    
}
