package main

import (
	"adsense_report_sync/auth"
	"adsense_report_sync/db"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/adsense/v2"
	"google.golang.org/api/option"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	task := flag.String("task", "", "task name")
	flag.Parse()

	loadEnv()

	switch *task {
	case "start":
		start()
	case "pm2":
		pm2()
	case "test":
		runTask()
	default:
		log.Fatalf("task not support: %s. Available: start, pm2, test", *task)
	}

}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
}

func start() {
	c := cron.New()

	cronTime := "0 3 * * *"

	_, err := c.AddFunc(cronTime, func() {
		log.Printf("Running cron at %s", time.Now().Format("2006-01-02 15:04:05"))
		runTask()
	})

	if err != nil {
		panic(err)
	}
	log.Printf("Started cron task with opt: %s", cronTime)
	c.Start()
	select {}
}

func pm2() {
	obj := map[string]interface{}{
		"apps": []interface{}{
			map[string]interface{}{
				"name":       "adsense report sync",
				"script":     "npm",
				"args":       "run start",
				"cwd":        func() string { dir, _ := os.Getwd(); return dir }(),
				"out_file":   os.Getenv("LOG_FILE"),
				"error_file": os.Getenv("LOG_FILE"),
			},
		},
	}
	f, err := os.OpenFile("pm2_generated.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
		return
	}
	defer f.Close()
	str, _ := json.MarshalIndent(obj, "", "  ")
	_, err = f.WriteString(string(str))
	if err != nil {
		log.Fatalf("error writing to file: %v", err)
	}
	log.Printf("wrote to file: %s", f.Name())
}

func runTask() {
	ctx := context.Background()

	creds, err := os.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("could not read credentials: %v", err)
	}

	config, err := google.ConfigFromJSON(creds, adsense.AdsenseReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := auth.GetClient(config)
	service, err := adsense.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Adsense client: %v", err)
	}

	account := os.Getenv("ADSENSE_ACCOUNT")

	startYear, startMonth, startDay := getDMY(-3)
	endYear, endMonth, endDay := getDMY(-1)

	reportCall := service.Accounts.Reports.Generate(account).
		StartDateYear(startYear).
		StartDateMonth(startMonth).
		StartDateDay(startDay).
		EndDateYear(endYear).
		EndDateMonth(endMonth).
		EndDateDay(endDay).
		Dimensions("CUSTOM_CHANNEL_ID", "DATE").
		Metrics("ESTIMATED_EARNINGS")
	report, err := reportCall.Do()
	if err != nil {
		log.Fatalf("Unable to retrieve report: %v", err)
	}

	rdb := db.GetRedisClient()

	for _, row := range report.Rows {
		tmp := strings.Split(row.Cells[0].Value, ":")
		channelID := tmp[len(tmp)-1]
		date := row.Cells[1].Value
		earnings := row.Cells[2].Value

		str, _ := json.Marshal(map[string]interface{}{
			"timestamp":  time.Now().Unix(),
			"channel_id": channelID,
			"date":       date,
			"earnings":   earnings,
			"id":         fmt.Sprintf("%s:%s", channelID, date),
		})
		rdb.LPush(ctx, os.Getenv("REDIS_QUEUE"), string(str))
		log.Printf("Channel ID: %s, Date: %s, Estimated Earnings: %s\n", channelID, date, earnings)
	}

}

func getDMY(offsetDay int) (int64, int64, int64) {
	date := time.Now().AddDate(0, 0, offsetDay)
	return int64(date.Year()), int64(date.Month()), int64(date.Day())
}
