package main

import (
	fcolor "github.com/fatih/color"
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

func init() {
	_ = godotenv.Load()
	rand.Seed(time.Now().UnixNano())

	exmo.init()
	fcolor.HiYellow("Balance %+v", exmo.Balance)

	tgBot.init()
	CandleStorage = make(map[string]CandleData)
	//drawBars()
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
			operation := getStrategy(param)
			strategies = append(strategies, operation)
		}
		exmo.downloadHistoryCandlesForStrategies(getUniqueStrategies(strategies))
		exmo.listenCandles(strategies)

	}

	select {}
}

func drawBars() {
	candleData := getCandleData("ETC_USDT.hour")
	candleData.restore()
	//exmo.downloadHistoryCandles(candleData)
	//candleData.backup()
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
	var ind1 []plotter.XY
	startI := candleData.index() - cnt
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
		ind1 = append(ind1, plotter.XY{X: float64(i), Y: (lo + hi) * .5})
		if (i+1)%4 == 0 {
			xTicks = append(xTicks, plot.Tick{Value: float64(i), Label: candleData.Time[startI+i].Format("15:04")})
		}
		p.Add(bar)
	}
	p.X.Tick.Marker = plot.ConstantTicks(xTicks)

	line := &plotter.Line{
		XYs: ind1,
		LineStyle: draw.LineStyle{
			Color:    color.RGBA{46, 113, 173, 255},
			Width:    vg.Points(2),
			Dashes:   []vg.Length{},
			DashOffs: 0,
		},
	}
	p.Y.Label.TextStyle.Font.Size = 40
	p.X.Label.TextStyle.Font.Size = 40
	p.Add(line, plotter.NewGrid())

	err := p.Save(1200, 600, "verticalBarChart.png")
	err = p.Save(200, 600, "verticalBarChart.pdf")

	tgBot.newOrderOpened("TEST", 10, 20, 30)

	if err != nil {
		log.Panic(err)
	}
}
