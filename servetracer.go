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
	DailyURI      = "/data/chart/day"
	MonthlyURI    = "/data/chart/month"
	AnnualURI     = "/data/chart/annual"
	LatestURI     = "/data/now"

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
		<div class="well" style="width: 17em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Battery charge</small></span>
			<div class="h1 text-center text-nowrap" id="bsoc"></div>		
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 17em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Solar panel</small></span>
			<div class="h1 text-center text-nowrap" id="pvp"></div>
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 17em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Load</small></span>
			<div class="h1 text-center text-nowrap" id="lp"></div>
		</div>
	</div>
	<div class="col-sm-6 col-md-3">
		<div class="well" style="width: 17em; margin: auto; margin-top: 2em;">
			<span class="h3 text-nowrap"><small>Total energy generated today</small></span>
			<div class="h1 text-center text-nowrap" id="egd"></div>
		</div>
	</div>
	<div class="center-block" style="text-align: center; width: 100%; margin-top: 4em; display: inline-block;">
		<div class="btn-group" role="group">
			<button type="button" class="btn btn-default btn-day">Day</button>
			<button type="button" class="btn btn-default btn-month">Month</button>
			<button type="button" class="btn btn-default btn-year">Year</button>
		</div>
	</div>
	<div style="display: inline-block; width: 100%;">
		<div style="margin-top: 1em; text-align: center;">
			<div id="chart"></div>
		</div>
	</div>

	<script>
		google.load('visualization', '1', {packages: ['corechart']});

		//var host = "http://10.0.1.199:8080"
		var host = "http://localhost:8080"
		//var host = ""
		var dailyData;
		var monthlyData;
		var annualData;
		var SECOND = 1000;
		var MINUTE = 60 * SECOND;
		var HOUR = 60 * MINUTE;
		var DAY = 24 * HOUR;

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

		var monthlyChartOptions = {
			subtitle: "Month",
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
				title: "kWh",
				viewWindow: {
					min: 0
				}
			}
		};

		var annualChartOptions = {
			subtitle: "Year",
			curveType: 'function',
			height: 400,
			legend: {
				position: "bottom"
			},
			hAxis: {
				title: "",
				format: "MMM",
				viewWindow: {
					min: moment().subtract(365, 'days').toDate(),
					max: moment().toDate()
				},
				gridlines: {
					count: 12
				},
			},
			vAxis: {
				title: "kWh",
				viewWindow: {
					min: 0
				}
			}
		};

		function init() {
			$(".btn-day").addClass("active");
			loadMonth();
			loadDaily();
			loadCurrent();
		    $("button.btn-day").click(function(event) {
		    	$(".btn").removeClass("active");
		    	$(".btn-day").addClass("active");
		    	if(dailyData == null) {
		    		loadDaily();
		    	} else {
		    		drawChart(dailyData, dailyChartOptions);
		    	}
		    });
		    $("button.btn-month").click(function() {
		    	$(".btn").removeClass("active");
		    	$(".btn-month").addClass("active");
		    	if(monthlyData == null) {
		    		loadMonth();
		    	} else {
		    		drawChart(monthlyData, monthlyChartOptions);
		    	}
		    });
		    $("button.btn-year").click(function() {
		    	$(".btn").removeClass("active");
		    	$(".btn-year").addClass("active");
		    	if(annualData == null) {
		    		loadAnnual();
		    	} else {
		    		drawChart(annualData, annualChartOptions);
		    	}
		    });

		    window.setInterval(loadCurrent, 5 * SECOND);
		    window.setInterval(loadDaily, 10 * MINUTE);
		    window.setInterval(loadMonth, DAY);
		}

		function loadCurrent() {
			$.ajax({
				url: host + "/data/now",
				cache: false
			}).done(function(data){
					var status = JSON.parse(data);
					$("#bsoc").html(status.bsoc + "%");
					$("#pvp").html(status.pvp.toFixed(2) + " W");
					$("#lp").html(status.lp.toFixed(2) + " W");
					$("#egd").html(status.egd.toFixed(2) + " kWh");
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
	    	$.ajax({
				url: host + "/data/chart/day",
				cache: false
			}).done(function(data) {
					if(data == null || data.length == 0) {
						dailyData = null;
					} else {
		    			dailyData = process(data);
						dailyData = new google.visualization.DataTable(dailyData);
						var formatter = new google.visualization.DateFormat({pattern: 'HH:mm'});
						formatter.format(dailyData, 0);
					}
	    			if($(".btn-day").hasClass("active")) {
	    				drawChart(dailyData, dailyChartOptions);
	    			} else {
	    				console.log("daily not selected");
	    			}
	    		}).
	    		fail(function(){
	    			console.log("daily failed");
	    		});
	    }

	    function loadMonth() {
	    	$.ajax({
				url: host + "/data/chart/month",
				cache: false
			}).done(function(data) {
					console.log("done");
					if(data == null || data.length == 0) {
						monthlyData = null;
					} else {
		    			monthlyData = process(data);
						monthlyData = new google.visualization.DataTable(monthlyData);
						var formatter = new google.visualization.DateFormat({pattern: 'yyyy-MM-dd'});
						formatter.format(monthlyData, 0);
					}
	    			if($(".btn-month").hasClass("active")) {
	    				drawChart(monthlyData, monthlyChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("monthly failed");
	    		});
	    }

	    function loadAnnual() {
	    	$.ajax({
				url: host + "/data/chart/annual",
				cache: false
			}).done(function(data) {
					console.log("done");
					if(data == null || data.length == 0) {
						annualData = null;
					} else {
		    			annualData = process(data);
						annualData = new google.visualization.DataTable(annualData);
						var formatter = new google.visualization.NumberFormat({fractionDigits: 2});
						formatter.format(annualData, 1);
						var formatter = new google.visualization.DateFormat({pattern: 'yyyy-MM-dd'});
						formatter.format(annualData, 0);
					}
	    			if($(".btn-year").hasClass("active")) {
	    				drawChart(annualData, annualChartOptions);
	    			}
	    		}).
	    		fail(function(){
	    			console.log("annual failed");
	    		});
	    }


		function drawChart(data, options) {
			$("#chart").html('<h3>The server is busy generating the chart, try again in a litte while</h3>');

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
	flag.StringVar(&port, "p", "8080", "HTTP service port")
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

func openDB() {
	var err error
	db, err = sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
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

func avg(end time.Time, minuteInterval int64) (ts []gotracer.TracerStatus, err error) {
	now := time.Now().UTC()

	// Minimum number of samples allowed to calculate an average. Below this
	// we consider values as missing.
	var minSamples int64 = int64(float32(minuteInterval) * float32(12) * 0.75)
	var i time.Time = end
	for i.Before(now) {
		var its []gotracer.TracerStatus
		its, err = load(i, i.Add(time.Minute*time.Duration(minuteInterval)))
		if err != nil {
			return
		}
		var t gotracer.TracerStatus

		// Set timestamp to middle of the sample interval
		t.Timestamp = i.Add(time.Minute * time.Duration(minuteInterval/2))

		// Convert to Unix timestamp
		t.Timestamp = time.Unix(t.Timestamp.Unix(), 0)

		// Only calculate average if there is a minimum of samples
		if int64(len(its)) > minSamples {
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
		} else {
			t.ArrayPower = -1
		}
		ts = append(ts, t)
		i = i.Add(time.Minute * time.Duration(minuteInterval))
	}
	return
}

func googleChart(ts []gotracer.TracerStatus) (chart string, err error) {
	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "datetime"}, GoogleChartsCol{Type: "number", Label: "Generated"}, GoogleChartsCol{Type: "number", Label: "Consumed"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		if tp.ArrayPower == -1 {
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: nil})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: nil})
		} else {
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.ArrayPower})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.LoadPower})
		}
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return
	}
	chart = string(b)
	return
}

func googleKWHChart(ts []gotracer.TracerStatus) (chart string, err error) {
	var table GoogleChartsDataTable
	table.Cols = []GoogleChartsCol{GoogleChartsCol{Type: "date"}, GoogleChartsCol{Type: "number", Label: "Generated"}, GoogleChartsCol{Type: "number", Label: "Consumed"}}

	for _, tp := range ts {
		var row GoogleChartsRow
		if tp.ArrayPower == -1 {
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: nil})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: nil})
		} else {
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.Timestamp})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.EnergyGeneratedDaily})
			row.Cells = append(row.Cells, GoogleChartsCell{Value: tp.EnergyConsumedDaily})
		}
		table.Rows = append(table.Rows, row)
	}

	var b []byte
	b, err = json.Marshal(table)
	if err != nil {
		return
	}
	chart = string(b)
	return
}

func updateDailyCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24)

	ts, err := avg(end, 10)
	if err != nil {
		return err
	}

	dailyCache, err = googleChart(ts)
	if err != nil {
		return err
	}

	log.Printf("updated daily cache in %v", time.Since(start))

	return nil
}

func updateWeeklyCache() error {
	start := time.Now()
	end := time.Now().UTC().Add(time.Hour * -24 * 7)

	ts, err := avg(end, 30)
	if err != nil {
		return err
	}

	weeklyCache, err = googleChart(ts)
	if err != nil {
		return err
	}

	log.Printf("updated weekly cache in %v", time.Since(start))

	return nil
}

func updateAnnualCache() error {
	start := time.Now()

	rows, err := db.Query("SELECT timestamp, MAX(generated_day), MAX(consumed_day) FROM log GROUP BY DATE(timestamp) ORDER BY timestamp")
	if err != nil {
		return err
	}
	defer rows.Close()

	var ts []gotracer.TracerStatus
	for rows.Next() {
		var t gotracer.TracerStatus
		if err = rows.Scan(&t.Timestamp, &t.EnergyGeneratedDaily, &t.EnergyConsumedDaily); err != nil {
			return err
		}
		t.Timestamp = time.Unix(t.Timestamp.Unix(), 0)
		ts = append(ts, t)
	}

	var c int32 = 0
	var gen float32 = 0
	var con float32 = 0
	var ts2 []gotracer.TracerStatus
	for _, t := range ts {
		if t.Timestamp.Weekday() == time.Sunday {
			gen = gen / float32(c)
			con = con / float32(c)
			t.EnergyGeneratedDaily = gen
			t.EnergyConsumedDaily = con
			ts2 = append(ts2, t)
			c = 0
			gen = 0
			con = 0
		} else {
			gen += t.EnergyGeneratedDaily
			con += t.EnergyConsumedDaily
			c++
		}
	}

	annualCache, err = googleKWHChart(ts2)
	if err != nil {
		return err
	}

	log.Printf("updated annual cache in %v", time.Since(start))

	return err
}

func updateMonthlyCache() error {
	start := time.Now()

	rows, err := db.Query("SELECT timestamp, MAX(generated_day), MAX(consumed_day) FROM log GROUP BY DATE(timestamp) ORDER BY timestamp")
	if err != nil {
		return err
	}
	defer rows.Close()

	var ts []gotracer.TracerStatus
	for rows.Next() {
		var t gotracer.TracerStatus
		if err = rows.Scan(&t.Timestamp, &t.EnergyGeneratedDaily, &t.EnergyConsumedDaily); err != nil {
			return err
		}
		t.Timestamp = time.Unix(t.Timestamp.Unix(), 0)
		ts = append(ts, t)
	}

	monthlyCache, err = googleKWHChart(ts)
	if err != nil {
		return err
	}

	log.Printf("updated monthly cache in %v", time.Since(start))

	return err
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

func monthlyDaemon() {
	err := updateMonthlyCache()
	if err != nil {
		log.Printf("updateMonthlyCache failed: %v", err)
	}
	c := time.Tick(24 * time.Hour)
	for _ = range c {
		err = updateMonthlyCache()
		if err != nil {
			log.Printf("updateMonthlyCache failed: %v", err)
		}
	}
}

func annualDaemon() {
	err := updateAnnualCache()
	if err != nil {
		log.Printf("updateAnnualCache failed: %v", err)
	}
	c := time.Tick(168 * time.Hour)
	for _ = range c {
		err = updateAnnualCache()
		if err != nil {
			log.Printf("updateAnnualCache failed: %v", err)
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
	go monthlyDaemon()
	go annualDaemon()
	http.HandleFunc("/", IndexHandler)
	http.HandleFunc(LatestURI, LatestHandler)
	http.HandleFunc(DailyURI, DailyHandler)
	http.HandleFunc(MonthlyURI, MonthlyHandler)
	http.HandleFunc(AnnualURI, AnnualHandler)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}
