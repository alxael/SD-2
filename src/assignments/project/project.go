package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Global configurations

var logIntervalMs = 100
var logFileBasePath = "../src/assignments/project/logs"
var syncLogFilePath = "../src/assignments/project/logs/synchronization.csv"
var configurationPath = "../src/assignments/project/configuration.json"

// Helper for formatting time
func FormatTime(milliseconds int64) string {
	return time.Unix(milliseconds/1e3, (milliseconds%1e3)*1e6).Format("2006-01-02 15:04:05.000")
}

// Configuration structs

type SynchronizationConfiguration struct {
	RefreshTimeMs    int `json:"refreshTimeMs"`
	RequestTimeoutMs int `json:"requestTimeoutMs"`
	MaxDeviationMs   int `json:"maxDeviationsMs"`
}

type MasterConfiguration struct {
	Id             string `json:"id"`
	CurrentTimeMs  int64  `json:"currentTimeMs"`
	HttpServerPort int    `json:"httpServerPort"`
}

type SlaveConfiguration struct {
	Id             string `json:"id"`
	CurrentTimeMs  int64  `json:"currentTimeMs"`
	HttpServerPort int    `json:"httpServerPort"`
	SendDelayMs    int64  `json:"sendDelayMs"`
	ReceiveDelayMs int64  `json:"receiveDelayMs"`
}

type Configuration struct {
	Settings SynchronizationConfiguration `json:"settings"`
	Master   MasterConfiguration          `json:"master"`
	Slaves   []SlaveConfiguration         `json:"slaves"`
}

// Clock simulation logic

type Clock struct {
	CurrentTimeMs int64
	TargetTimeMs  int64
	TargetReached bool
	mutex         sync.RWMutex
}

func NewClock(initialTimeMs int64) *Clock {
	return &Clock{
		CurrentTimeMs: initialTimeMs,
		TargetTimeMs:  initialTimeMs,
		TargetReached: true,
	}
}

func (clock *Clock) SetTarget(targetTimeMs int64) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.TargetTimeMs = targetTimeMs
	clock.TargetReached = false
}

func (clock *Clock) GetTime() int64 {
	clock.mutex.RLock()
	defer clock.mutex.RUnlock()
	return clock.CurrentTimeMs
}

func (clock *Clock) Run() {
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		clock.mutex.Lock()

		if clock.TargetReached {
			clock.CurrentTimeMs += 1
			clock.mutex.Unlock()
		} else {
			if clock.TargetTimeMs > clock.CurrentTimeMs {
				difference := clock.TargetTimeMs - clock.CurrentTimeMs
				clock.CurrentTimeMs += difference
				clock.TargetReached = true
				clock.mutex.Unlock()
			} else {
				clock.mutex.Unlock()

				difference := clock.CurrentTimeMs - clock.TargetTimeMs
				if difference > 0 {
					time.Sleep(time.Duration(difference) * time.Millisecond)
				}

				clock.mutex.Lock()
				clock.TargetReached = true
				clock.mutex.Unlock()
			}
		}
	}
}

// Configuration loading

func LoadConfiguration(filePath string) (*Configuration, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open configuration file: %w", err)
	}
	defer file.Close()

	var configuration Configuration
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configuration); err != nil {
		return nil, fmt.Errorf("Failed to parse JSON content: %w", err)
	}

	return &configuration, nil
}

// File logger for node time

func StartTimeLogger(nodeId string, clock *Clock) {
	go func() {
		_ = os.MkdirAll(logFileBasePath, 0755) // Ensure directory exists
		fileName := fmt.Sprintf("%s/%s-time.log", logFileBasePath, nodeId)
		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			log.Printf("[%s] Failed to open log file: %v\n", nodeId, err)
			return
		}
		defer file.Close()

		refreshDuration := time.Duration(logIntervalMs) * time.Millisecond

		ticker := time.NewTicker(refreshDuration)
		defer ticker.Stop()

		initialTime := FormatTime(clock.GetTime())
		if _, err := fmt.Fprintf(file, "[%s] Initial time: %s\n", nodeId, initialTime); err != nil {
			log.Printf("[%s] Failed to write to log file: %v\n", nodeId, err)
		}

		for range ticker.C {
			currentTime := FormatTime(clock.GetTime())
			if _, err := fmt.Fprintf(file, "[%s] Current time: %s\n", nodeId, currentTime); err != nil {
				log.Printf("[%s] Failed to write to log file: %v\n", nodeId, err)
			}
		}
	}()
}

func InitializeSyncLogger(slaves []SlaveConfiguration) {
	_ = os.MkdirAll(logFileBasePath, 0755)
	file, err := os.OpenFile(syncLogFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("Failed to create sync log file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"SyncID", "Average"}
	for _, slave := range slaves {
		header = append(header, fmt.Sprintf("Diff_%s", slave.Id))
	}
	for _, slave := range slaves {
		header = append(header, fmt.Sprintf("Corr_%s", slave.Id))
	}

	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header to sync log: %v", err)
	}
}

func LogSyncCycleToCSV(
	cycleId int,
	average int64,
	results map[string]int64,
	slaves []SlaveConfiguration,
) {
	file, err := os.OpenFile(syncLogFilePath, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Failed to open sync log file for appending: %v", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	row := []string{
		strconv.Itoa(cycleId),
		strconv.FormatInt(average, 10),
	}

	for _, slave := range slaves {
		if diff, ok := results[slave.Id]; ok {
			row = append(row, strconv.FormatInt(diff, 10))
		} else {
			row = append(row, "Err")
		}
	}

	for _, slave := range slaves {
		if diff, ok := results[slave.Id]; ok {
			correction := average - diff
			row = append(row, strconv.FormatInt(correction, 10))
		} else {
			row = append(row, "Err")
		}
	}

	if err := writer.Write(row); err != nil {
		log.Printf("Failed to write record to sync log: %v", err)
	}
}

// Master server methods
type SlaveResult struct {
	Slave      SlaveConfiguration
	Difference int64
	Success    bool
}

func GetTimeFromSlave(
	masterConfiguration *MasterConfiguration,
	slaveConfiguration *SlaveConfiguration,
	httpClient *http.Client,
) (int64, error) {
	url := fmt.Sprintf("http://localhost:%d/time", slaveConfiguration.HttpServerPort)
	response, err := httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	slaveTimeString := string(bodyBytes)
	slaveTime, err := strconv.ParseInt(slaveTimeString, 10, 64)
	if err != nil {
		return 0, err
	}

	return slaveTime, nil
}

func SendCorrectionToSlave(
	masterConfiguration *MasterConfiguration,
	slaveConfiguration *SlaveConfiguration,
	httpClient *http.Client,
	correction int64,
) error {
	url := fmt.Sprintf("http://localhost:%d/time", slaveConfiguration.HttpServerPort)
	body := strconv.FormatInt(correction, 10)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", response.StatusCode)
	}

	return nil
}

func GetSlaveResult(
	masterConfiguration *MasterConfiguration,
	slaveConfiguration *SlaveConfiguration,
	httpClient *http.Client,
	clock *Clock,
	maxDeviation int64,
) SlaveResult {
	var difference = int64(0)
	var success = false

	slaveTime, err := GetTimeFromSlave(masterConfiguration, slaveConfiguration, httpClient)
	masterTime := clock.GetTime()

	if err == nil {
		difference = slaveTime - masterTime
		success = true
	} else {
		difference = 0
		success = false
	}

	absoluteDifference := difference
	if absoluteDifference < 0 {
		absoluteDifference = -absoluteDifference
	}

	if absoluteDifference > maxDeviation {
		difference = 0
	}

	return SlaveResult{
		Slave:      *slaveConfiguration,
		Difference: difference,
		Success:    success,
	}
}

func SyncSlaves(
	configuration *Configuration,
	clock *Clock,
	cycleId int,
) {
	var waitGroup sync.WaitGroup
	var resultsMutex sync.Mutex
	var results []SlaveResult

	httpClient := http.Client{
		Timeout: time.Duration(configuration.Settings.RequestTimeoutMs) * time.Millisecond,
	}

	fmt.Printf("[%s] Starting synchronization cycle %d.\n", configuration.Master.Id, cycleId)

	// gather times from slaves
	for _, slave := range configuration.Slaves {
		waitGroup.Add(1)
		go func(slave SlaveConfiguration) {
			defer waitGroup.Done()

			slaveResult := GetSlaveResult(
				&configuration.Master,
				&slave,
				&httpClient,
				clock,
				int64(configuration.Settings.MaxDeviationMs),
			)

			resultsMutex.Lock()
			results = append(results, slaveResult)
			resultsMutex.Unlock()
		}(slave)
	}

	waitGroup.Wait()

	// compute average
	var sumDifferences int64 = 0
	var validCount int64 = 0
	resultsMap := make(map[string]int64)

	for _, res := range results {
		if res.Success {
			sumDifferences += res.Difference
			validCount++
			resultsMap[res.Slave.Id] = res.Difference
		}
	}

	average := int64(0)
	if validCount > 0 {
		average = sumDifferences / validCount
	}

	// log to csv
	LogSyncCycleToCSV(cycleId, average, resultsMap, configuration.Slaves)

	// apply correction to master
	masterTime := clock.GetTime()
	clock.SetTarget(masterTime + average)

	// apply correction to slaves
	for _, result := range results {
		if !result.Success {
			continue
		}
		waitGroup.Add(1)
		correction := average - result.Difference

		go func(slave SlaveConfiguration, correction int64) {
			defer waitGroup.Done()
			_ = SendCorrectionToSlave(&configuration.Master, &slave, &httpClient, correction)
		}(result.Slave, correction)
	}

	waitGroup.Wait()

	fmt.Printf("[%s] Synchronization cycle %d finished.\n", configuration.Master.Id, cycleId)
}

func StartMasterServer(
	configuration *Configuration,
	readyChannel chan<- string,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	clock := NewClock(configuration.Master.CurrentTimeMs)
	go clock.Run()

	StartTimeLogger(configuration.Master.Id, clock)
	InitializeSyncLogger(configuration.Slaves)

	mux := http.NewServeMux()

	address := fmt.Sprintf(":%d", configuration.Master.HttpServerPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[%s] Failed to bind to port %d: %v", configuration.Master.Id, configuration.Master.HttpServerPort, err)
	}

	server := &http.Server{
		Handler: mux,
	}

	fmt.Printf("[%s] Server listening on port %d\n",
		configuration.Master.Id,
		configuration.Master.HttpServerPort,
	)
	readyChannel <- configuration.Master.Id

	go func() {
		ticker := time.NewTicker(time.Duration(configuration.Settings.RefreshTimeMs) * time.Millisecond)
		defer ticker.Stop()

		cycleId := 1
		for range ticker.C {
			SyncSlaves(configuration, clock, cycleId)
			cycleId++
		}
	}()

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[%s] Server error: %v", configuration.Master.Id, err)
	}
}

// Slave server methods

func HandleGetTimeRequest(
	slaveConfiguration *SlaveConfiguration,
	clock *Clock,
	writer http.ResponseWriter,
) {
	if slaveConfiguration.SendDelayMs > 0 {
		time.Sleep(time.Duration(slaveConfiguration.SendDelayMs) * time.Millisecond)
	}

	currentTime := clock.GetTime()
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte(fmt.Sprintf("%d", currentTime)))
}

func HandlePostTimeRequest(
	slaveConfiguration *SlaveConfiguration,
	clock *Clock,
	request *http.Request,
	writer http.ResponseWriter,
) {
	if slaveConfiguration.ReceiveDelayMs > 0 {
		time.Sleep(time.Duration(slaveConfiguration.ReceiveDelayMs) * time.Millisecond)
	}

	bodyBytes, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	correctionStr := string(bodyBytes)
	correction, err := strconv.ParseInt(correctionStr, 10, 64)
	if err != nil {
		http.Error(writer, "Invalid integer format", http.StatusBadRequest)
		return
	}

	current := clock.GetTime()
	newTarget := current + correction
	clock.SetTarget(newTarget)

	writer.WriteHeader(http.StatusOK)
}

func StartSlaveServer(
	slaveConfiguration SlaveConfiguration,
	readyChannel chan<- string,
	waitGroup *sync.WaitGroup,
) {
	defer waitGroup.Done()

	clock := NewClock(slaveConfiguration.CurrentTimeMs)
	go clock.Run()

	StartTimeLogger(slaveConfiguration.Id, clock)

	mux := http.NewServeMux()

	mux.HandleFunc("/time", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			HandleGetTimeRequest(&slaveConfiguration, clock, writer)
		case http.MethodPost:
			HandlePostTimeRequest(&slaveConfiguration, clock, request, writer)
		default:
			http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	address := fmt.Sprintf(":%d", slaveConfiguration.HttpServerPort)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("[%s] Failed to bind port %d: %v", slaveConfiguration.Id, slaveConfiguration.HttpServerPort, err)
	}

	server := &http.Server{
		Handler: mux,
	}

	fmt.Printf("[%s] Server listening on port %d (Delay: Send=%dms, Receive=%dms)\n",
		slaveConfiguration.Id,
		slaveConfiguration.HttpServerPort,
		slaveConfiguration.SendDelayMs,
		slaveConfiguration.ReceiveDelayMs,
	)
	readyChannel <- slaveConfiguration.Id

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[%s] Server error: %v", slaveConfiguration.Id, err)
	}
}

func main() {
	if _, err := os.Stat(configurationPath); os.IsNotExist(err) {
		if _, err := os.Stat("configuration.json"); err == nil {
			configurationPath = "configuration.json"
		}
	}

	configuration, err := LoadConfiguration(configurationPath)
	if err != nil {
		log.Fatal(err)
	}

	var waitGroup sync.WaitGroup

	masterReady := make(chan string)

	waitGroup.Add(1)
	go StartMasterServer(configuration, masterReady, &waitGroup)
	<-masterReady

	slaveCount := len(configuration.Slaves)
	slaveReady := make(chan string, slaveCount)
	for _, slaveConfig := range configuration.Slaves {
		waitGroup.Add(1)
		go StartSlaveServer(slaveConfig, slaveReady, &waitGroup)
	}

	for i := 0; i < slaveCount; i++ {
		<-slaveReady
	}

	waitGroup.Wait()
}
