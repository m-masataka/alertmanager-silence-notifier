package main

import (
	"fmt"
	"os"
	"time"
	"net/url"
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/prometheus/alertmanager/api/v2/client/silence"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"
	"github.com/ashwanthkumar/slack-go-webhook"
)

func main() {
	os.Exit(run())
}

type IdAndState struct {
	id string
	state string
}

func run() int {
	var (
		host         = kingpin.Flag("alertmanager.host", "Alertmanager host.").Default("localhost").String()
		port         = kingpin.Flag("alertmanager.port", "Alertmanager port.").Default("9093").String()
		username     = kingpin.Flag("slack.username", "username of slack bot.").Default("Bot").String()
		channel      = kingpin.Flag("slack.channel", "post channel.").Default("general").String()
		token        = kingpin.Flag("slack.token", "slack api token.").Default("xxx").String()
		interval     = kingpin.Flag("interval", "api polling interval.").Default("5s").Duration()
		timerange    = kingpin.Flag("timerange", "api polling time range.").Default("5m").Duration()
	)
	kingpin.CommandLine.GetFlag("help").Short('h')
	kingpin.Parse()

	u, _ := url.Parse("http://" + *host + ":" + *port)
	silenceParams := silence.NewGetSilencesParams()
	amclient := cli.NewAlertmanagerClient(u)

	prev := make([]IdAndState, 0)
	getOk, err := amclient.Silence.GetSilences(silenceParams)
	if err != nil {
		fmt.Println(err)
		return 1
	}
	for _, silence := range getOk.Payload {
		if time.Time(*silence.EndsAt).After(time.Now().UTC().Add(- *timerange)) {
			prev = append(prev, IdAndState{*silence.ID, *silence.Status.State})
		}
	}

	for {
		tmp := make([]IdAndState, 0)
		getOk, err = amclient.Silence.GetSilences(silenceParams)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, silence := range getOk.Payload {
				if time.Time(*silence.EndsAt).After(time.Now().UTC().Add(-5 * time.Minute)) {
					if CompareSilences(prev, *silence.ID, *silence.Status.State) {
						PostSlack(*silence,*username,*channel,*token, *host, *port)
					}	
					tmp = append(tmp, IdAndState{*silence.ID, *silence.Status.State})
				}
			}
			prev = tmp
		}
		time.Sleep(*interval)
	}
	return 0
}

func CompareSilences(list []IdAndState, id string, state string) bool {
	for _, v := range list {
		if v.id == id && v.state == state {
			return false
		}
	}
	return true
}

func PostSlack(s models.GettableSilence, username string, channel string, token string, host string, port string) error {
	titleStr := *s.Status.State + " : " + *s.ID
	valueStr := "Starts at: " + s.Silence.StartsAt.String() + "\n" +
		"Ends at     : " + s.Silence.EndsAt.String() + "\n" +
		"Updated at  : " + s.UpdatedAt.String() + "\n" +
		"Created by  : " + *s.Silence.CreatedBy + "\n" +
		"Comment     : " + *s.Silence.Comment + "\n" +
		"Matchers:\n"
	for _, matcher := range s.Silence.Matchers {
		var operator string
		if *matcher.IsRegex {
			operator = "~="
		} else {
			operator = "="
		}
		valueStr += *matcher.Name + operator + *matcher.Value + "\n"
	}
	silenceUrl := "http://" + host + ":" + port + "/#/silences/" + *s.ID
	attachment := slack.Attachment{}
	attachment.AddField(slack.Field{ Title: titleStr, Value: valueStr })
	attachment.AddAction(slack.Action { Type: "button", Text: "View", Url: silenceUrl })
	var color string
	var msg string
	if *s.Status.State == "active" {
		color = "good"
		msg = "New Silence!"
	} else if *s.Status.State == "pending" {
		color = "warning"
		msg = "New Silence!"
	} else {
		color = "danger"
		msg = "Expired Silence!"
	}
	attachment.Color = &color
	payload := slack.Payload {
		Username: username,
		Channel: channel,
		Text: msg,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send("https://hooks.slack.com/services/" + token, "", payload)
	if err != nil {
		return err[0]
	}
	return nil
}
