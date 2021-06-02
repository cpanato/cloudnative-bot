package function

import (
	"encoding/base64"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

// Handle a serverless request
func Handle(req []byte) string {
	rand.Seed(time.Now().UnixNano())

	// log.Printf("Content Type: %s", os.Getenv("Http_Content_Type"))
	// if os.Getenv("Http_Content_Type") != "application/json" {
	// 	return "Invalid Content Type"
	// }

	err := loadConfig()
	if err != nil {
		log.Printf("failed to load the config: %v", err)
		return "Failed to load the configuration"
	}

	log.Println("***starting CloudNative Bot Next Kubecon function***")

	api := anaconda.NewTwitterApiWithCredentials(Config.TwitterAccessToken, Config.TwitterAccessSecret, Config.TwitterConsumerKey, Config.TwitterConsumerSecretKey)

	if _, err := api.VerifyCredentials(); err != nil {
		return fmt.Sprintf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens: %v", err)
	}

	layOut := "02/01/2006 15:04:05" // dd/mm/yyyy hh:mm:ss
	///"12/10/2021 8:00:00"
	nextKubecon, err := time.Parse(layOut, os.Getenv("NEXT_KUBECON"))
	if err != nil {
		return err.Error()
	}

	diff := time.Until(nextKubecon)
	inDays := RoundTime(diff.Seconds() / 86400)
	log.Printf("nextKubecon - %v - %v", nextKubecon, inDays)

	var msg string
	if inDays < 0 {
		return "it is in the past"
	} else if inDays == 0 {
		msg = "Kubecon LA is TODAY!! Enjoy the event! #cncf #KubeCon\nhttps://events.linuxfoundation.org/kubecon-cloudnativecon-north-america/"
	} else {
		msg = fmt.Sprintf("Kubecon LA will be in %d days #cncf #KubeCon\nhttps://events.linuxfoundation.org/kubecon-cloudnativecon-north-america/", inDays)
	}

	sendSimpleTweet(api, msg, nil)

	return "done"
}

func sendSimpleTweet(twitterAPI *anaconda.TwitterApi, message string, image []byte) {
	replyParams := url.Values{}
	replyParams.Set("auto_populate_reply_metadata", "true")
	replyParams.Set("display_coordinates", "false")

	if image != nil {
		mediaResponse, err := twitterAPI.UploadMedia(base64.StdEncoding.EncodeToString(image))
		if err != nil {
			log.Printf("Error uploading the image Err=%s\n", err.Error())
		} else {
			replyParams.Set("media_ids", mediaResponse.MediaIDString)
		}
	}

	result, err := twitterAPI.PostTweet(message, replyParams)
	if err != nil {
		log.Printf("Error while posting the tweet. Err=%s\n", err.Error())
		return
	}

	log.Printf("Tweet posted. TweetID = %s\n", result.IdStr)
}

func RoundTime(input float64) int {
	var result float64

	if input < 0 {
		result = math.Ceil(input - 0.5)
	} else {
		result = math.Floor(input + 0.5)
	}

	// only interested in integer, ignore fractional
	i, _ := math.Modf(result)

	return int(i)
}
