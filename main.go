package main

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"image/color"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

var scheduler *gocron.Scheduler

var exchange string

var apiHandler ApiInterface

func init() {
	rand.Seed(time.Now().UnixNano())

	_ = godotenv.Load()
	exchange = os.Getenv("exchange")

	switch exchange {
	case "exmo":
		apiHandler = exmo.init()
	//case "binance":
	//	apiHandler = binance.init()
	default:
		log.Fatal("NO HANDLER")
	}

	apiHandler.showBalance()

	tgBot.init()
	CandleStorage = make(map[string]CandleData)
}

func main() {
	envParams := os.Getenv("params")
	if envParams != "" {
		params := strings.Split(envParams, "}{")
		params[0] = params[0][1:]
		params[len(params)-1] = params[len(params)-1][:len(params[len(params)-1])-1]
		var strategies []Strategy

		scheduler = gocron.NewScheduler(time.UTC)
		scheduler.StartAsync()

		for _, param := range params {
			strategy := getStrategy(param)
			strategies = append(strategies, strategy)
		}
		apiHandler.downloadHistoryCandlesForStrategies(getUniqueStrategies(strategies))
		apiHandler.listenCandles(strategies)

	}

	select {}
}

func (candleData *CandleData) drawBars(tp, sl float64) string {
	c := candleData.Candles

	red := color.NRGBA{R: 255, G: 108, B: 101, A: 255}
	green := color.NRGBA{R: 109, G: 195, B: 88, A: 255}
	gray := color.NRGBA{R: 22, G: 26, B: 37, A: 255}
	blue := color.NRGBA{G: 160, B: 240, A: 255}
	p := plot.New()
	p.BackgroundColor = gray
	p.X.Max = 62
	p.X.Min = -1
	p.X.Tick.Color = blue
	p.X.Tick.Label.Color = blue
	p.X.Tick.Label.Font.Size = 16

	p.Y.Tick.Color = blue
	p.Y.Tick.Label.Color = blue
	p.Y.Tick.Label.Font.Size = 16

	w := vg.Points(10)
	tw := vg.Points(1)
	whiskerStyle := draw.LineStyle{
		Width:  vg.Points(2),
		Dashes: []vg.Length{},
	}
	const cnt = 60
	startI := candleData.len() - cnt
	var xTicks []plot.Tick
	for i := 0; i < cnt; i++ {
		lo := c[L][startI+i]
		op := c[O][startI+i]
		cl := c[C][startI+i]
		hi := c[H][startI+i]
		bar, _ := plotter.NewBoxPlot(w, float64(i), plotter.Values{
			lo, op, op, op, cl, hi,
		})
		bar.AdjLow = lo
		bar.AdjHigh = hi
		bar.CapWidth = tw
		bar.WhiskerStyle = whiskerStyle
		bar.Outside = nil
		if cl >= op {
			bar.FillColor = green
			bar.WhiskerStyle.Color = green
			bar.BoxStyle.Color = green
			bar.MedianStyle.Color = green
		} else {
			bar.FillColor = red
			bar.WhiskerStyle.Color = red
			bar.BoxStyle.Color = red
			bar.MedianStyle.Color = red
		}
		if (i+1)%4 == 0 {
			xTicks = append(xTicks, plot.Tick{Value: float64(i), Label: candleData.Time[startI+i].Format("15:04")})
		}
		p.Add(bar)
	}
	p.X.Tick.Marker = plot.ConstantTicks(xTicks)
	p.Y.Label.TextStyle.Font.Size = 40
	p.X.Label.TextStyle.Font.Size = 40

	lineFn := func(level float64, clr color.RGBA) *plotter.Line {
		return &plotter.Line{
			XYs: []plotter.XY{{X: float64(cnt + 2), Y: level}, {X: float64(cnt), Y: level}},
			LineStyle: draw.LineStyle{
				Color:    clr,
				Width:    vg.Points(3),
				Dashes:   []vg.Length{vg.Points(4)},
				DashOffs: 0,
			}}
	}

	tpLevel := lineFn(tp, color.RGBA{R: 96, G: 255, A: 255})
	slLevel := lineFn(sl, color.RGBA{R: 255, B: 96, A: 255})
	p.Add(plotter.NewGrid(), tpLevel, slLevel)

	folder := fmt.Sprintf("./screens/%s", time.Now().Format("06/01/02"))
	_ = os.MkdirAll(folder, 0755)
	path := fmt.Sprintf("%s/%s_%s_%s.png", folder, candleData.Pair, resolution, time.Now().Format("1504"))
	err := p.Save(1200, 600, path)

	if err != nil {
		log.Panic(err)
	}

	return path
}
