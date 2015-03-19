package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spagettikod/gotracer"
)

const (
	SDBDomainName = "tracerlogger"
	DailyURI      = "/day"
	WeeklyURI     = "/week"
	MonthlyURI    = "/month"
	AnnualURI     = "/annual"
	LatestURI     = "/now"

	SelectSQL   string = `SELECT timestamp, array_voltage, array_current, array_power, battery_voltage, battery_current, battery_soc, battery_temp, battery_max_volt, battery_min_volt, device_temp, load_voltage, load_current, load_power, load, consumed_day, consumed_month, consumed_year, consumed_total, generated_day, generated_month, generated_year, generated_total FROM log `
	IntervalSQL string = SelectSQL + `WHERE timestamp BETWEEN ? AND ? ORDER BY timestamp ASC;`
	LatestSQL   string = SelectSQL + `WHERE timestamp ORDER BY timestamp DESC LIMIT 1;`

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
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Batteri</small></span>
			<div class="h1 text-center text-nowrap" id="bsoc"></div>		
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 15em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Solpanel</small></span>
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
	<div class="center-block" style="text-align: center; width: 100%; margin-top: 4em; display: inline-block;">
		<div class="btn-group" role="group">
			<button type="button" class="btn btn-default btn-day">Dag</button>
			<button type="button" class="btn btn-default btn-week">Vecka</button>
			<button type="button" class="btn btn-default btn-month">Månad</button>
			<button type="button" class="btn btn-default btn-year">År</button>
		</div>
	</div>
	<div style="display: inline-block; width: 100%;">
		<div style="margin-top: 1em; text-align: center;">
			<div id="chart"></div>
			<!--div id="loading"><i class="fa fa-spinner fa-spin fa-3x" style="margin-top: 2em;"></i></div-->
		</div>
	</div>

	<script>
		google.load('visualization', '1', {packages: ['corechart']});

		var host = "";
		var dailyData;
		var weeklyData;
		var monthlyData;
		var annualData;

		var dailyChartOptions = {
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
					count: 12
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

		var weeklyChartOptions = {
			subtitle: "Vecka",
			curveType: 'function',
			height: 400,
			legend: {
				position: "bottom"
			},
			hAxis: {
				title: "",
				format: "cccc",
				viewWindow: {
					min: moment().subtract(7, 'days').toDate(),
					max: moment().toDate()
				},
				gridlines: {
					count: 7
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

		var monthlyChartOptions = {
			subtitle: "Månad",
			curveType: 'function',
			height: 400,
			legend: {
				position: "bottom"
			},
			hAxis: {
				title: "",
				format: "d",
				viewWindow: {
					min: moment().subtract(30, 'days').toDate(),
					max: moment().toDate()
				},
				gridlines: {
					count: 30
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

		var annualChartOptions = {
			subtitle: "År",
			curveType: 'function',
			height: 400,
			legend: {
				position: "bottom"
			},
			hAxis: {
				title: "",
				format: "MMMM",
				viewWindow: {
					min: moment().subtract(365, 'days').toDate(),
					max: moment().toDate()
				},
				gridlines: {
					count: 12
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

		function init() {
			$(".btn-day").addClass("active");
			loadDaily();
			loadWeekly();
			loadMonthly();
			loadAnnual();
			loadCurrent();
		    $("button.btn-day").click(function(event) {
		    	$(".btn").removeClass("active");
		    	$(".btn-day").addClass("active");
		    	drawChart(dailyData, dailyChartOptions);
		    });
		    $("button.btn-week").click(function() {
		    	$(".btn").removeClass("active");
		    	$(".btn-week").addClass("active");
		    	drawChart(weeklyData, weeklyChartOptions);
		    });
		    $("button.btn-month").click(function() {
		    	$(".btn").removeClass("active");
		    	$(".btn-month").addClass("active");
		    	drawChart(monthlyData, monthlyChartOptions);
		    });
		    $("button.btn-year").click(function() {
		    	$(".btn").removeClass("active");
		    	$(".btn-year").addClass("active");
		    	drawChart(annualData, annualChartOptions);
		    });

		    window.setInterval(loadCurrent, 5000);
		    window.setInterval(loadDaily, 600000);
		    window.setInterval(loadMonthly, 21600000);
		    window.setInterval(loadAnnual, 86400000);
		}

		function loadCurrent() {
			$.get(host + "/now").
				done(function(data){
					var status = JSON.parse(data);
					$("#bsoc").html(status.bsoc + "%");
					$("#pvp").html(status.pvp.toFixed(2) + " W");
					$("#lp").html(status.lp.toFixed(2) + " W");
					$("#ecd").html(status.ecd.toFixed(2) + " kWh");
				}).
				fail(function() {
					console.log("Failed");
				});
		}

		function process(data) {
			var aDate;
			return JSON.parse(data, function(k,v){
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
		}

	    function loadDaily() {
	    	$.get(host + "/day").
	    		done(function(data) {
	    			dailyData = process(data);
	    			if($(".btn-day").hasClass("active")) {
	    				drawChart(dailyData, dailyChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("daily failed");
	    		});
	    }

	    function loadWeekly() {
	    	$.get(host + "/week").
	    		done(function(data) {
	    			weeklyData = process(data);
	    			if($(".btn-week").hasClass("active")) {
	    				drawChart(weeklyData, weeklyChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("weekly failed");
	    		});
	    }

	    function loadMonthly() {
	    	$.get(host + "/month").
	    		done(function(data) {
	    			monthlyData = process(data);
	    			if($(".btn-month").hasClass("active")) {
	    				drawChart(monthlyData, monthlyChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("monthly failed");
	    		});
	    }

	    function loadAnnual() {
	    	$.get(host + "/annual").
	    		done(function(data) {
	    			annualData = process(data);
	    			if($(".btn-annual").hasClass("active")) {
	    				drawChart(annualData, annualChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("annual failed");
	    		});
	    }

		function drawChart(input, options) {
			$("#chart").html('<i class="fa fa-spinner fa-spin fa-3x" style="margin-top: 2em;"></i>');
			var data = new google.visualization.DataTable(input);

			var formatter = new google.visualization.DateFormat({pattern: 'HH:mm'});
			formatter.format(data, 0);
			
			var chart = new google.visualization.LineChart(document.getElementById('chart'));


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
	port, dbFile, dailyCache, weeklyCache, monthlyCache, annualCache string
	db                                                               *sql.DB
	ErrNoRowsFound                                                   error = errors.New("No rows found")
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

func openDB() {
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func logAccess(req *http.Request) {
	log.Printf("%v %v %v %v %v", req.RemoteAddr, req.RequestURI, req.Method, req.URL, req.UserAgent())
}

func latest() (t gotracer.TracerStatus, err error) {
	err = db.QueryRow(LatestSQL).Scan(&t.Timestamp, &t.ArrayVoltage, &t.ArrayCurrent, &t.ArrayPower, &t.BatteryVoltage, &t.BatteryCurrent, &t.BatterySOC, &t.BatteryTemp, &t.BatteryMaxVoltage, &t.BatteryMinVoltage, &t.DeviceTemp, &t.LoadVoltage, &t.LoadCurrent, &t.LoadPower, &t.Load, &t.EnergyConsumedDaily, &t.EnergyConsumedMonthly, &t.EnergyConsumedAnnual, &t.EnergyConsumedTotal, &t.EnergyGeneratedDaily, &t.EnergyGeneratedMonthly, &t.EnergyGeneratedAnnual, &t.EnergyGeneratedTotal)
	return
}

func load(begin, end time.Time) (ts []gotracer.TracerStatus, err error) {
	rows, err := db.Query(IntervalSQL, begin, end)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t gotracer.TracerStatus
		if err = rows.Scan(&t.Timestamp, &t.ArrayVoltage, &t.ArrayCurrent, &t.ArrayPower, &t.BatteryVoltage, &t.BatteryCurrent, &t.BatterySOC, &t.BatteryTemp, &t.BatteryMaxVoltage, &t.BatteryMinVoltage, &t.DeviceTemp, &t.LoadVoltage, &t.LoadCurrent, &t.LoadPower, &t.Load, &t.EnergyConsumedDaily, &t.EnergyConsumedMonthly, &t.EnergyConsumedAnnual, &t.EnergyConsumedTotal, &t.EnergyGeneratedDaily, &t.EnergyGeneratedMonthly, &t.EnergyGeneratedAnnual, &t.EnergyGeneratedTotal); err != nil {
			return
		}
		ts = append(ts, t)
	}
	return
}

func round(f float64) int32 {
	return int32(f + math.Copysign(0.5, f))
}

func avg(end time.Time, interval int64) (ts []gotracer.TracerStatus, err error) {
	now := time.Now().UTC()
	var i time.Time = end
	for i.Before(now) {
		var its []gotracer.TracerStatus
		its, err = load(i, i.Add(time.Minute*time.Duration(interval)))
		if err != nil {
			return
		}
		var t gotracer.TracerStatus
		for _, s := range its {
			t.ArrayCurrent += s.ArrayCurrent
			t.ArrayPower += s.ArrayPower
			t.ArrayVoltage += s.ArrayVoltage
			t.BatteryCurrent += s.BatteryCurrent
			t.BatterySOC += s.BatterySOC
			t.BatteryTemp += s.BatteryTemp
			t.BatteryVoltage += s.BatteryVoltage
			t.LoadCurrent += s.LoadCurrent
			t.LoadPower += s.LoadPower
			t.LoadVoltage += s.LoadVoltage
			t.DeviceTemp += s.DeviceTemp
		}
		t.Timestamp = i.Add(time.Minute * time.Duration(interval/2))
		t.Timestamp = time.Unix(t.Timestamp.Unix(), 0)
		if len(its) > 0 {
			t.ArrayCurrent = t.ArrayCurrent / float32(len(its))
			t.ArrayPower = t.ArrayPower / float32(len(its))
			t.ArrayVoltage = t.ArrayVoltage / float32(len(its))
			t.BatteryCurrent = t.BatteryCurrent / float32(len(its))
			t.BatterySOC = round(float64(float32(t.BatterySOC) / float32(len(its))))
			t.BatteryTemp = t.BatteryTemp / float32(len(its))
			t.BatteryVoltage = t.BatteryVoltage / float32(len(its))
			t.LoadCurrent = t.LoadCurrent / float32(len(its))
			t.LoadPower = t.LoadPower / float32(len(its))
			t.LoadVoltage = t.LoadVoltage / float32(len(its))
			t.DeviceTemp = t.DeviceTemp / float32(len(its))
		}
		ts = append(ts, t)
		i = i.Add(time.Minute * time.Duration(interval))
	}
	return
}

func updateDailyCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24)

	ts, err := avg(end, 10)
	if err != nil {
		return err
	}

	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Solpanel"}, GoogleChartsCol{Type: "number", Label: "Förbrukning"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return err
	}
	dailyCache = string(b)

	log.Printf("updated daily cache in %v", time.Since(start))

	return nil
}

func updateWeeklyCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24 * 7)

	ts, err := avg(end, 60)
	if err != nil {
		return err
	}

	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Solpanel"}, GoogleChartsCol{Type: "number", Label: "Förbrukning"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return err
	}
	weeklyCache = string(b)

	log.Printf("updated weekly cache in %v", time.Since(start))

	return nil
}

func updateMonthlyCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24 * 30)

	ts, err := avg(end, 360)
	if err != nil {
		return err
	}

	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Solpanel"}, GoogleChartsCol{Type: "number", Label: "Förbrukning"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return err
	}
	monthlyCache = string(b)

	log.Printf("updated monthly cache in %v", time.Since(start))

	return nil
}

func updateAnnualCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24 * 365)

	ts, err := avg(end, 1440)
	if err != nil {
		return err
	}

	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Solpanel"}, GoogleChartsCol{Type: "number", Label: "Förbrukning"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
		row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return err
	}
	annualCache = string(b)

	log.Printf("updated annual cache in %v", time.Since(start))

	return nil
}

func dailyDaemon() {
	err := updateDailyCache()
	if err != nil {
		log.Printf("daily failed: %v", err)
	}
	c := time.Tick(10 * time.Minute)
	for _ = range c {
		err = updateDailyCache()
		if err != nil {
			log.Printf("daily failed: %v", err)
		}
	}
}

func weeklyDaemon() {
	err := updateWeeklyCache()
	if err != nil {
		log.Printf("weekly failed: %v", err)
	}
	c := time.Tick(1 * time.Hour)
	for _ = range c {
		err = updateWeeklyCache()
		if err != nil {
			log.Printf("weekly failed: %v", err)
		}
	}
}

func monthlyDaemon() {
	err := updateMonthlyCache()
	if err != nil {
		log.Printf("monthly failed: %v", err)
	}
	c := time.Tick(6 * time.Hour)
	for _ = range c {
		err = updateMonthlyCache()
		if err != nil {
			log.Printf("monthly failed: %v", err)
		}
	}
}

func annualDaemon() {
	err := updateAnnualCache()
	if err != nil {
		log.Printf("annual failed: %v", err)
	}
	c := time.Tick(24 * time.Hour)
	for _ = range c {
		err = updateAnnualCache()
		if err != nil {
			log.Printf("annual failed: %v", err)
		}
	}
}

func LatestHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)
	t, err := latest()
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

func DailyHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)

	fmt.Fprint(w, dailyCache)
}

func WeeklyHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)

	fmt.Fprint(w, weeklyCache)
}

func MonthlyHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)

	fmt.Fprint(w, monthlyCache)
}

func AnnualHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	corsHeaders(w)

	fmt.Fprint(w, annualCache)
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	logAccess(req)
	fmt.Fprint(w, IndexPage)
}

func main() {
	flag.Parse()
	if port == "" || dbFile == "" {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	openDB()
	go dailyDaemon()
	go weeklyDaemon()
	go monthlyDaemon()
	go annualDaemon()
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc(LatestURI, LatestHandler)
	http.HandleFunc(DailyURI, DailyHandler)
	http.HandleFunc(WeeklyURI, WeeklyHandler)
	http.HandleFunc(MonthlyURI, MonthlyHandler)
	http.HandleFunc(AnnualURI, AnnualHandler)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
