package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	tf "github.com/TinkoffCreditSystems/invest-openapi-go-sdk"
	"github.com/go-co-op/gocron"
	_ "github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var apiHandler ApiInterface
var scheduler *gocron.Scheduler

func main() {
	scheduler = gocron.NewScheduler(time.UTC)
	scheduler.StartAsync()

	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())

	//c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt, os.Kill)
	//go func() {
	//	for sig := range c {
	//		log.Printf("Stopped %+v", sig)
	//		pprof.StopCPUProfile()
	//		os.Exit(1)
	//	}
	//}()

	envParams := os.Getenv("params")

	CandleStorage = make(map[string]CandleData)

	if envParams != "" {
		params := strings.Split(envParams, "}{")
		params[0] = params[0][1:]
		params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]
		var operations []OperationParameter
		for _, param := range params {
			operation := getOperationParameter(param)
			candleData := getCandleData(operation.FigiInterval)
			apiHandler = getApiHandler(operation.FigiInterval)
			apiHandler.downloadCandles(candleData, operation)
			operations = append(operations, operation)
		}
		apiHandler.listenCandles(operations)
	}

	////tinkoff.Open("BBG000B9XRY4", 2)
	////tinkoff.Close("BBG000B9XRY4", 2)
	////ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	////defer cancel()
	////p, _ := tinkoff.ApiClient.Portfolio(ctx, tinkoff.Account.ID)
	////fmt.Printf("%+v", p)
	//
	////listenCandles(tinkoff)
	//
	select {}

}

func getApiHandler(figi string) ApiInterface {
	handler := func(figi string) ApiInterface {
		if strings.Contains(figi, "_") {
			return &exmo
		}
		return &tinkoff
	}(figi)
	handler.init()
	return handler
}

func getOperationParameter(str string) OperationParameter {
	var operationParameter OperationParameter

	params := strings.Split(str, "|")
	figis := strings.Split(params[0], " ")
	operationParameter.FigiInterval = figis[0] + ".hour"
	operationParameter.Op = toInt(figis[1])
	operationParameter.Cl = toInt(figis[2])

	operationParameter.Ind1 = getIndicatorParameter(params[1])
	operationParameter.Ind2 = getIndicatorParameter(params[2])

	return operationParameter
}

func getIndicatorParameter(str string) IndicatorParameter {
	var indicatorParameter IndicatorParameter

	split := strings.Split(str, " ")
	indicatorParameter.IndicatorType = IndicatorType(toInt(split[0]))
	indicatorParameter.BarType = BarType(split[1])
	indicatorParameter.Coef = toInt(split[2])

	return indicatorParameter
}

func toInt(str string) int {
	i, err := strconv.Atoi(str)
	if err != nil {
		fmt.Printf("%+v", err)
		return -100
	}
	return i
}

func getFigiAndInterval(str string) (string, tf.CandleInterval) {
	param := strings.Split(str, ".")
	return param[0], tf.CandleInterval(param[1])
}

func EncodeToBytes(p interface{}) []byte {
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(p)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("uncompressed size (bytes): ", len(buf.Bytes()))
	return buf.Bytes()
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ReadFromFile(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

// parallel processes the data in separate goroutines.
func parallel(start, stop int, fn func(<-chan int)) {
	count := stop - start
	if count < 1 {
		return
	}

	procs := runtime.GOMAXPROCS(0)
	if procs > count {
		procs = count
	}

	c := make(chan int, count)
	for i := start; i < stop; i++ {
		c <- i
	}
	close(c)

	var wg sync.WaitGroup
	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(c)
		}()
	}
	wg.Wait()
}

func figiInterval(figi string, interval tf.CandleInterval) string {
	return fmt.Sprintf("%s_%s", figi, interval)
}
