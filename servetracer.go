package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spagettikod/gotracer"
)

const (
	SDBDomainName   = "tracerlogger"
	DailyPVPowerURI = "/day/pv/power"
	LatestURI       = "/now"

	SelectSQL string = `SELECT ts, array_voltage, array_current, array_power, battery_voltage, battery_current, battery_soc, battery_temp, battery_max_volt, battery_min_volt, device_temp, load_voltage, load_current, load_power, load, consumed_day, consumed_month, consumed_year, consumed_total, generated_day, generated_month, generated_year, generated_total FROM log `
	LatestSQL string = SelectSQL + `WHERE timestamp ORDER BY timestamp DESC LIMIT 1;`

	IndexPage string = `
<!DOCTYPE html>
<html>
<head>
	<link href="http://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.2.0/css/bootstrap.min.css" rel="stylesheet">
	<link href="http://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.2.0/css/font-awesome.min.css" rel="stylesheet">

	<script src="http://cdnjs.cloudflare.com/ajax/libs/jquery/2.1.1/jquery.min.js"></script>
	<script src="http://cdnjs.cloudflare.com/ajax/libs/twitter-bootstrap/3.2.0/js/bootstrap.min.js"></script>
	<script src="http://cdnjs.cloudflare.com/ajax/libs/moment.js/2.7.0/moment.min.js"></script>
	<script type="text/javascript" src="https://www.google.com/jsapi"></script>
</head>
<body>
	<!--div style="margin-bottom: 2em;">&nbsp;</div-->
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Batteriladdning</small></span>
			<div class="h1 text-center text-nowrap" id="bsoc"></div>		
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Effekt från solpanel</small></span>
			<div class="h1 text-center text-nowrap" id="pvp"></div>
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Förbrukning</small></span>
			<div class="h1 text-center text-nowrap" id="lp"></div>
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Total effekt idag</small></span>
			<div class="h1 text-center text-nowrap" id="ecd"></div>
		</div>
	</div>
	<!--div class="col-md-12 center-block" id="chart" style="text-align: center; margin-top: 4em;">
		<div class="btn-group" role="group">
			<button type="button" class="btn btn-default btn-day" id="btn-day">Dag</button>
			<button type="button" class="btn btn-default btn-week">Vecka</button>
			<button type="button" class="btn btn-default btn-month">Månad</button>
			<button type="button" class="btn btn-default btn-year">År</button>
		</div>
	</div>
	<div class="col-md-12" style="margin-top: 1em; text-align: center;">
		<div id="chart"></div>
		<div id="loading"><i class="fa fa-spinner fa-spin fa-3x" style="margin-top: 2em;"></i></div>
	</div-->

	<script>
		function init() {
		    /*$("#btn-day").click(function() {
		    	loadChart("day");
		    });

		    $("#btn-day").click();*/
		    loadCurrent();
		    window.setInterval(loadCurrent, 5000);
		}

		function loadCurrent() {
			$.get("/now").
				done(function(data){
					var status = JSON.parse(data);
					$("#bsoc").html(status.bsoc + "%");
					$("#pvp").html(status.pvp + " W");
					$("#lp").html(status.lp + " W");
					$("#ecd").html(status.ecd + " kWh");
				}).
				fail(function() {
					console.log("Failed");
				});
		}

		// chart is either day, week, month or year
		function loadChart(chart) {
			$("#loading").show();
	    	$("#btn").removeClass("active");
	    	$("#btn-" + chart).addClass("active");
		}


		google.load('visualization', '1', {packages: ['corechart']});

	    function loadData() {
	    	$.get("//day/pv/power").
	    		done(function(data) {
					var aDate;
					parsedData = JSON.parse(data, function(k,v){
						var reISO = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:Z|(\+|-)([\d|:]*))?$/;
				        if (typeof v === 'string') {
				            var a = reISO.exec(v);
				            if (a) {
				            	aDate = new Date(v);
				            	return aDate;
				            }
				        }
				        return v;
					});
					drawChart(parsedData, aDate);
	    		}).
	    		fail(function(){
	    			console.log("failed");
	    		});
	    }

		function drawChart(input, aDate) {
			var data = new google.visualization.DataTable(input);

			var formatter = new google.visualization.DateFormat({pattern: 'HH:mm'});
			formatter.format(data, 0);
			
			var chart = new google.visualization.LineChart(document.getElementById('chart'));

			var options = {
				subtitle: "Idag",
				curveType: 'function',
				height: 400,
				legend: {
					position: "bottom"
				},
				hAxis: {
					title: "",
					format: "HH:mm",
					viewWindow: {
						min: moment().subtract(1, 'days').toDate(),
						max: moment().toDate()
					},
					gridlines: {
						count: 12,
						units: {
							days: {format: ["MMM dd"]},
							hours: {format: ["HH:mm", "ha"]},
						}
					},
				},
				vAxis: {
					title: "Watt",
					viewWindow: {
						min: 0,
						max: 80
					},
					gridlines: {
						count: 10,
					}
				}
			};
			chart.draw(data, options);
	    }

    $(function() {
		google.setOnLoadCallback(init);
    });
  </script>
</body>
</html>
	`
)

var (
	port, dbFile   string
	db             *sql.DB
	ErrNoRowsFound error = errors.New("No rows found")
)

type GoogleChartsCol struct {
	Id    string `json:"id,omitempty"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type GoogleChartsRow struct {
	Cells []GoogleChartsCell `json:"c"`
}

type GoogleChartsCell struct {
	Value     interface{} `json:"v"`
	Formatted string      `json:"f,omitempty"`
}

type GoogleChartsDataTable struct {
	Cols []GoogleChartsCol `json:"cols"`
	Rows []GoogleChartsRow `json:"rows"`
}

func init() {
	flag.StringVar(&port, "p", "", "HTTP service port")
	flag.StringVar(&dbFile, "db", "", "path and filename of SQLite database")
}

func corsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-1), t.Location())
}

func LatestHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)
	t, err := Latest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var b []byte
	b, err = json.Marshal(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, string(b))
}

func uriDate(uriRoot string, req *http.Request) (t time.Time, err error) {
	id := req.URL.RequestURI()[len(uriRoot):]
	if id == "" {
		t = time.Now().UTC()
	} else {
		t, err = time.Parse("2006-01-02", id[1:]) // Remove the leading slash
	}
	return
}

/*
func DayPowerHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	start := time.Now()
	//t, err := uriDate(DailyPVPowerURI, req)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	log.Println(err)
	//	return
	//}
	t := time.Now().UTC()

	var tps []TracerPowerStatus
	var b []byte
	err := fetch(t.Add((time.Hour*24)*-1), t, &tps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	log.Printf("query: %v", time.Since(start).String())
	start = time.Now()

	// Limit to one every 5 minute average
	//var pvTotal float32
	var j int64
	var tmp []TracerPowerStatus
	for _, tp := range tps {
		//timeTotal += tp.Timestamp.Unix()
		//pvTotal += tp.ArrayPower
		if j == 120 || len(tps)-int(j) == 0 {
			//avgTime := time.Unix(timeTotal/j, 0)
			//tmp = append(tmp, TracerPowerStatus{ArrayPower: pvTotal / float32(j), Timestamp: avgTime})
			tp.Timestamp = time.Unix(tp.Timestamp.Unix(), 0)
			tmp = append(tmp, tp)
			j = 0
			//timeTotal = 0
			//pvTotal = 0
		}
		j++
	}

	log.Printf("limit: %v", time.Since(start).String())
	start = time.Now()

	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Solpanel"}, GoogleChartsCol{Type: "number", Label: "Förbrukning"}}

	for _, tp := range tmp {
		var row GoogleChartsRow
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		table.Rows = append(table.Rows, row)
	}

	log.Printf("table: %v", time.Since(start).String())
	start = time.Now()

	b, err = json.Marshal(table)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err)
		return
	}

	log.Printf("marshal: %v", time.Since(start).String())

	w.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(w, string(b))
}*/

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	fmt.Fprint(w, IndexPage)
}

func logAccess(req *http.Request) {
	log.Printf("%v %v %v %v %v", req.RemoteAddr, req.RequestURI, req.Method, req.URL, req.UserAgent())
}

func Latest() (t gotracer.TracerStatus, err error) {
	err = db.QueryRow(LatestSQL).Scan(&t.Timestamp, &t.ArrayVoltage, &t.ArrayCurrent, &t.ArrayPower, &t.BatteryVoltage, &t.BatteryCurrent, &t.BatterySOC, &t.BatteryTemp, &t.BatteryMaxVoltage, &t.BatteryMinVoltage, &t.DeviceTemp, &t.LoadVoltage, &t.LoadCurrent, &t.LoadPower, &t.Load, &t.EnergyConsumedDaily, &t.EnergyConsumedMonthly, &t.EnergyConsumedAnnual, &t.EnergyConsumedTotal, &t.EnergyGeneratedDaily, &t.EnergyGeneratedMonthly, &t.EnergyGeneratedAnnual, &t.EnergyGeneratedTotal)
	return
}

func openDB() {
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func main() {
	flag.Parse()
	if port == "" || dbFile == "" {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	openDB()
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc(LatestURI, LatestHandler)
	//http.HandleFunc(DailyPVPowerURI, DayPowerHandler)
	//http.HandleFunc(DailyPVPowerURI+"/", DayPowerHandler)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
