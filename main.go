package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Status struct {
	UpTime           int64  `json:"upTime"`
	DeviceName       string `json:"deviceName"`
	Paired           bool   `json:"paired"`
	FirmwareVersion  string `json:"firmwareVersion"`
	AccessoryID      string `json:"accessoryID"`
	LocalIP          string `json:"localIP"`
	SubnetMask       string `json:"subnetMask"`
	GatewayIP        string `json:"gatewayIP"`
	MacAddress       string `json:"macAddress"`
	WifiSSID         string `json:"wifiSSID"`
	GDOSecurityType  string `json:"GDOSecurityType"`
	GarageDoorState  string `json:"garageDoorState"`
	GarageLockState  string `json:"garageLockState"`
	GarageLightOn    bool   `json:"garageLightOn"`
	GarageMotion     bool   `json:"garageMotion"`
	GarageObstructed bool   `json:"garageObstructed"`
	PasswordRequired bool   `json:"passwordRequired"`
	RebootSeconds    int    `json:"rebootSeconds"`
	FreeHeap         int    `json:"freeHeap"`
	MinHeap          int    `json:"minHeap"`
	MinStack         int    `json:"minStack"`
	CrashCount       int    `json:"crashCount"`
	WifiPhyMode      int    `json:"wifiPhyMode"`
	WifiPower        int    `json:"wifiPower"`
	TTCseconds       int    `json:"TTCseconds"`
	MotionTriggers   int    `json:"motionTriggers"`
	LEDidle          int    `json:"LEDidle"`
	LastDoorUpdateAt int    `json:"lastDoorUpdateAt"`
	CheckFlashCRC    bool   `json:"checkFlashCRC"`
}

var (
	upTime           *prometheus.GaugeVec
	paired           *prometheus.GaugeVec
	garageLightOn    *prometheus.GaugeVec
	garageMotion     *prometheus.GaugeVec
	garageObstructed *prometheus.GaugeVec
	passwordRequired *prometheus.GaugeVec
	freeHeap         *prometheus.GaugeVec
	minHeap          *prometheus.GaugeVec
	minStack         *prometheus.GaugeVec
	crashCount       *prometheus.GaugeVec
	garageDoorState  *prometheus.GaugeVec
	deviceInfo       *prometheus.GaugeVec

	requestCount *prometheus.CounterVec // new counter metric

	jsonAddress string
	port        string
	location    string
	mutex       sync.Mutex
)

func init() {
	upTime = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_up_time_seconds",
		Help: "Uptime of the garage door in seconds.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	paired = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_paired",
		Help: "Indicates if the garage door is paired.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	garageLightOn = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_light_on",
		Help: "Indicates if the garage light is on.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	garageMotion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_motion",
		Help: "Indicates if there is motion detected in the garage.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	garageObstructed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_obstructed",
		Help: "Indicates if the garage door is obstructed.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	passwordRequired = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_password_required",
		Help: "Indicates if a password is required.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	freeHeap = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_free_heap_bytes",
		Help: "Free heap memory in bytes.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	minHeap = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_min_heap_bytes",
		Help: "Minimum heap memory in bytes.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	minStack = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_min_stack_bytes",
		Help: "Minimum stack memory in bytes.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	crashCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_crash_count",
		Help: "Number of crashes.",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	garageDoorState = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_door_state",
		Help: "The state of the garage door (0 = Closed, 1 = Open).",
	}, []string{"location", "accessoryID", "deviceName", "localIP", "macAddress"})

	deviceInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "homekit_ratgdo_info",
		Help: "Garage door device info.",
	}, []string{"location", "firmwareVersion", "subnetMask", "gatewayIP", "wifiSSID", "garageLockState", "GDOSecurityType"})

	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "homekit_ratgdo_request_count",
			Help: "Count of HTTP requests to the JSON endpoint, labeled by status code class.",
		},
		[]string{"status_code_class"},
	)

	flag.StringVar(&jsonAddress, "json-address", "http://ratgdo/status.json", "The address of the JSON endpoint")
	flag.StringVar(&port, "port", "8080", "The port to expose metrics on")
	flag.StringVar(&location, "location", "home", "The location label for the metrics")
	flag.Parse()

	// Register all metrics including new requestCount
	prometheus.MustRegister(upTime)
	prometheus.MustRegister(paired)
	prometheus.MustRegister(garageLightOn)
	prometheus.MustRegister(garageMotion)
	prometheus.MustRegister(garageObstructed)
	prometheus.MustRegister(passwordRequired)
	prometheus.MustRegister(freeHeap)
	prometheus.MustRegister(minHeap)
	prometheus.MustRegister(minStack)
	prometheus.MustRegister(crashCount)
	prometheus.MustRegister(garageDoorState)
	prometheus.MustRegister(deviceInfo)
	prometheus.MustRegister(requestCount)

	// Pre-allocate request count labels
	requestCount.WithLabelValues("2xx")
	requestCount.WithLabelValues("3xx")
	requestCount.WithLabelValues("4xx")
	requestCount.WithLabelValues("5xx")
}

func fetchData() (int, error) {
	mutex.Lock()
	defer mutex.Unlock()

	resp, err := http.Get(jsonAddress)
	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return 0, err
	}

	var status Status
	err = json.Unmarshal(body, &status)
	if err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		return 0, err
	}

	upTime.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(float64(status.UpTime))
	paired.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(boolToFloat(status.Paired))
	garageLightOn.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(boolToFloat(status.GarageLightOn))
	garageMotion.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(boolToFloat(status.GarageMotion))
	garageObstructed.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(boolToFloat(status.GarageObstructed))
	passwordRequired.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(boolToFloat(status.PasswordRequired))
	freeHeap.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(float64(status.FreeHeap))
	minHeap.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(float64(status.MinHeap))
	minStack.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(float64(status.MinStack))
	crashCount.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(float64(status.CrashCount))

	if status.GarageDoorState == "Closed" {
		garageDoorState.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(0)
	} else if status.GarageDoorState == "Open" {
		garageDoorState.WithLabelValues(location, status.AccessoryID, status.DeviceName, status.LocalIP, status.MacAddress).Set(1)
	}

	deviceInfo.With(prometheus.Labels{
		"location":        location,
		"firmwareVersion": status.FirmwareVersion,
		"subnetMask":      status.SubnetMask,
		"gatewayIP":       status.GatewayIP,
		"wifiSSID":        status.WifiSSID,
		"garageLockState": status.GarageLockState,
		"GDOSecurityType": status.GDOSecurityType,
	}).Set(1)

	return resp.StatusCode, nil
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	statusCode, err := fetchData()
	if err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
	}

	switch {
	case statusCode >= 200 && statusCode < 300:
		requestCount.WithLabelValues("2xx").Inc()
	case statusCode >= 300 && statusCode < 400:
		requestCount.WithLabelValues("3xx").Inc()
	case statusCode >= 400 && statusCode < 500:
		requestCount.WithLabelValues("4xx").Inc()
	case statusCode >= 500:
		requestCount.WithLabelValues("5xx").Inc()
	}

	promhttp.Handler().ServeHTTP(w, r)
}

func main() {
	http.HandleFunc("/metrics", metricsHandler)
	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
