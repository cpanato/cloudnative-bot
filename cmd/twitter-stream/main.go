package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"github.com/openfaas/connector-sdk/types"
	"github.com/openfaas/faas-provider/auth"
)

func main() {
	var (
		flagConfigFile string
		api            *anaconda.TwitterApi

		commandCNMatch      = regexp.MustCompile(`(?mi)^*/(?:cloudnativetv)(?: +(.+?))?\s*$`)
		commandPatchMatch   = regexp.MustCompile(`(?mi)^*/(?:k8s-patch-schedule)(?: +(.+?))?\s*$`)
		commandKubeconMatch = regexp.MustCompile(`(?mi)^*/(?:kubecon-random-video)(?: +(.+?))?\s*$`)
	)

	flag.StringVar(&flagConfigFile, "config", "config.json", "Configuration file for the honk bot.")
	flag.Parse()

	err := loadConfig(flagConfigFile)
	if err != nil {
		log.Fatalf("Error starting the job: %v", err)
	}

	creds := &auth.BasicAuthCredentials{
		User:     Config.OpenFaasUsername,
		Password: Config.OpenFaasPassword,
	}

	config := &types.ControllerConfig{
		RebuildInterval:         time.Millisecond * 1000,
		GatewayURL:              Config.OpenFaasGateway,
		PrintResponse:           true,
		PrintResponseBody:       true,
		AsyncFunctionInvocation: false,
	}

	log.Println("***Starting CloudNative Bot Stream***")

	api = anaconda.NewTwitterApiWithCredentials(Config.TwitterAccessToken, Config.TwitterAccessSecret, Config.TwitterConsumerKey, Config.TwitterConsumerSecretKey)
	if _, err := api.VerifyCredentials(); err != nil {
		log.Fatalf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens: %s", err)
	}

	streamValues := url.Values{}
	streamValues.Set("track", "/cloudnativetv,/k8s-patch-schedule,/kubecon-random-video")
	streamValues.Set("stall_warnings", "true")
	log.Println("Starting CloudNative Stream...")
	s := api.PublicStreamFilter(streamValues)
	defer s.Stop()

	controller := types.NewController(creds, config)

	receiver := ResponseReceiver{}
	controller.Subscribe(&receiver)

	controller.BeginMapBuilder()

	go func() {
		for t := range s.C {
			switch v := t.(type) {
			case anaconda.Tweet:
				data := []byte(fmt.Sprintf(`{"text": %q,"id": %d, "id_str": %q, "user_screen_name": %q, "api_key": %q}`, v.Text, v.Id, v.IdStr, v.User.ScreenName, Config.TokenAPIKey))

				var topic string
				if commandCNMatch.MatchString(v.Text) {
					topic = "cloudnative.twitter.stream"
				} else if commandPatchMatch.MatchString(v.Text) {
					topic = "cloudnative.twitter.schedule"
				} else if commandKubeconMatch.MatchString(v.Text) {
					topic = "cloudnative.twitter.video"
				} else {
					log.Printf("Did not match any command. Nothing to do - %s - https://twitter.com/%s/status/%d ", v.Text, v.User.ScreenName, v.Id)
					continue
				}

				log.Printf("Got one message - https://twitter.com/%s/status/%d - Invoking on topic %s\n", v.User.ScreenName, v.Id, topic)
				controller.Invoke(topic, &data)
			default:
				log.Printf("Got something else %v", v)
			}
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-sig
}

// ResponseReceiver enables connector to receive results from the
// function invocation
type ResponseReceiver struct {
}

// Response is triggered by the controller when a message is
// received from the function invocation
func (ResponseReceiver) Response(res types.InvokerResponse) {
	if res.Error != nil {
		log.Printf("twitter-stream got error: %s", res.Error.Error())
	} else {
		log.Printf("twitter-stream: [%d] %s => %s (%d) bytes", res.Status, res.Topic, res.Function, len(*res.Body))
	}
}
