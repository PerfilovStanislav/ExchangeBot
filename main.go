package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/go-co-op/gocron"
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

var scheduler *gocron.Scheduler

func init() {
	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())
	exmo.init()
}

func main() {
	CandleStorage = make(map[string]CandleData)

	//exmo.test()

	envParams := os.Getenv("params")
	if envParams != "" {
		params := strings.Split(envParams, "}{")
		params[0] = params[0][1:]
		params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]
		var operations []OperationParameter

		scheduler = gocron.NewScheduler(time.UTC)
		scheduler.StartAsync()

		for _, param := range params {
			operation := getOperationParameter(param)
			operations = append(operations, operation)
		}
		exmo.asyncDownloadHistoryCandles(getUniqueOperations(operations))
		exmo.listenCandles(operations)
	}

	select {}

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

func getFigiAndInterval(str string) (string, string) {
	param := strings.Split(str, ".")
	return param[0], "hour"
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

func f2s(x float64) string {
	return fmt.Sprintf("%v", x)
}

func s2f(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func i2s(i int64) string {
	return strconv.FormatInt(i, 10)
}

func getCurrencies(pair string) (Currency, Currency) {
	split := strings.Split(pair, "_")
	return Currency(split[0]), Currency(split[1])
}

func getLeftCurrency(pair string) Currency {
	currency, _ := getCurrencies(pair)
	return currency
}

func getRightCurrency(pair string) Currency {
	_, currency := getCurrencies(pair)
	return currency
}

func getUniqueOperations(operations []OperationParameter) []OperationParameter {
	var uniqueOperations []OperationParameter
	var symbols []string
	for _, operation := range operations {
		pair := operation.getPairName()
		if sliceIndex(symbols, pair) == -1 {
			symbols = append(symbols, pair)
			uniqueOperations = append(uniqueOperations, operation)
		}
	}
	return uniqueOperations
}

func sliceIndex[E comparable](s []E, v E) int {
	for i, vs := range s {
		if v == vs {
			return i
		}
	}
	return -1
}
