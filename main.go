package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func main() {
	clientURL := flag.String("rpc-url", "", "Solana RPC client URL (must be specified)")
	flag.Parse()

	envClientURL := os.Getenv("RPC_URL")

	var rpcURL string
	if envClientURL != "" {
		rpcURL = envClientURL
	} else {
		rpcURL = *clientURL
	}

	if rpcURL == "" {
		log.Fatalf("Error: The RPC client URL must be specified either with the -rpc-url parameter or the RPC_URL environment variable.")
		return
	}

	if err := termui.Init(); err != nil {
		fmt.Printf("Error initializing termui: %v\n", err)
		return
	}
	defer termui.Close()

	slotGauge := widgets.NewGauge()
	slotGauge.Title = "Solana Slot Progress"
	slotGauge.Percent = 0
	slotGauge.BarColor = termui.ColorWhite
	slotGauge.BorderStyle.Fg = termui.ColorWhite
	slotGauge.LabelStyle = termui.NewStyle(termui.ColorWhite, termui.ColorClear, termui.ModifierBold)
	slotGauge.TitleStyle = termui.NewStyle(termui.ColorWhite)
	slotGauge.SetRect(0, 6, 80, 10)

	etaStatus := widgets.NewParagraph()
	etaStatus.Border = true
	etaStatus.Title = "ETA"
	etaStatus.Text = "Calculating..."
	etaStatus.TextStyle.Fg = termui.ColorWhite
	etaStatus.BorderStyle.Fg = termui.ColorWhite
	etaStatus.SetRect(0, 0, 40, 3)

	nodeStatus := widgets.NewParagraph()
	nodeStatus.Border = true
	nodeStatus.Title = "Node Status"
	nodeStatus.Text = "Initializing..."
	nodeStatus.TextStyle.Fg = termui.ColorWhite
	nodeStatus.BorderStyle.Fg = termui.ColorWhite
	nodeStatus.SetRect(40, 0, 80, 3)

	grid := termui.NewGrid()
	termWidth, termHeight := termui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		termui.NewRow(0.2,
			termui.NewCol(0.5, etaStatus),
			termui.NewCol(0.5, nodeStatus),
		),
		termui.NewRow(0.3, slotGauge),
	)

	rpcMain := rpc.New(rpc.MainNetBeta_RPC)
	rpcClient := rpc.New(rpcURL)

	var previousSlotClient uint64
	var previousTime time.Time

	uiEvents := termui.PollEvents()
	ticker := time.NewTicker(time.Second).C

	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			}
		case <-ticker:
			slotMain, errMain := rpcMain.GetSlot(context.Background(), rpc.CommitmentFinalized)
			if errMain != nil {
				log.Printf("Error fetching mainnet slot: %v", errMain)
				nodeStatus.Text = "Error: Mainnet Slot"
				nodeStatus.TextStyle.Fg = termui.ColorRed
				termui.Render(nodeStatus)
				continue
			}

			slotClient, errClient := rpcClient.GetSlot(context.Background(), rpc.CommitmentFinalized)
			if errClient != nil {
				log.Printf("Error fetching client slot: %v", errClient)
				nodeStatus.Text = "Error: Client Slot"
				nodeStatus.TextStyle.Fg = termui.ColorRed
				termui.Render(nodeStatus)
				continue
			}

			var percent int
			if slotMain > 0 {
				percent = int((slotClient * 100) / slotMain)
			} else {
				percent = 0
			}

			var eta string
			if previousSlotClient > 0 {
				slotDifference := int64(slotMain - slotClient)
				timeDifference := time.Since(previousTime).Seconds()
				slotProgress := int64(slotClient - previousSlotClient)

				if slotProgress > 0 {
					estimatedTime := float64(slotDifference) * (timeDifference / float64(slotProgress))
					etaDuration := time.Duration(estimatedTime) * time.Second

					days := int(etaDuration.Hours()) / 24
					hours := int(etaDuration.Hours()) % 24
					minutes := int(etaDuration.Minutes()) % 60

					eta = fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
				} else {
					eta = "Calculating..."
				}
			} else {
				eta = "Calculating..."
			}

			previousSlotClient = slotClient
			previousTime = time.Now()

			slotGauge.Title = fmt.Sprintf("Solana Slot %d / %d", slotClient, slotMain)
			slotGauge.Percent = percent
			if percent <= 25 {
				slotGauge.BarColor = termui.ColorRed
			} else if percent > 25 && percent <= 99 {
				slotGauge.BarColor = termui.ColorYellow
			} else {
				slotGauge.BarColor = termui.ColorGreen
			}

			nodeHealth, errHealth := rpcClient.GetHealth(context.Background())
			if errHealth != nil || nodeHealth != "ok" {
				nodeStatus.Text = fmt.Sprintf("Node Status: Unhealthy")
				nodeStatus.TextStyle.Fg = termui.ColorRed
			} else {
				nodeStatus.Text = fmt.Sprintf("Node Status: Healthy")
				nodeStatus.TextStyle.Fg = termui.ColorGreen
			}

			etaStatus.Text = eta
			termui.Render(grid)
		}
	}
}
