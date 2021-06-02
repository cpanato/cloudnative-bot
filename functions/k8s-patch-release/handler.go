package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	getter "github.com/hashicorp/go-getter"
	"gopkg.in/yaml.v2"
)

var (
	commandPatchMatch = regexp.MustCompile(`(?mi)^*/(?:k8s-patch-schedule)(?: +(.+?))?\s*$`)
)

// PatchSchedule main struct to hold the schedules
type PatchSchedule struct {
	Schedules []Schedule `yaml:"schedules"`
}

// PreviousPatches struct to define the old pacth schedules
type PreviousPatches struct {
	Release            string `yaml:"release"`
	CherryPickDeadline string `yaml:"cherryPickDeadline"`
	TargetDate         string `yaml:"targetDate"`
}

// Schedule struct to define the release schedule for a specific version
type Schedule struct {
	Release            string            `yaml:"release"`
	Next               string            `yaml:"next"`
	CherryPickDeadline string            `yaml:"cherryPickDeadline"`
	TargetDate         string            `yaml:"targetDate"`
	EndOfLifeDate      string            `yaml:"endOfLifeDate"`
	PreviousPatches    []PreviousPatches `yaml:"previousPatches"`
}

type Tweet struct {
	ID             int64  `json:"id"`
	Text           string `json:"text"`
	IdStr          string `json:"id_str"`
	UserScreenName string `json:"user_screen_name"`
	APIKey         string `json:"api_key"`
}

// Handle a serverless request
func Handle(req []byte) string {
	rand.Seed(time.Now().UnixNano())

	// log.Printf("Content Type: %s", os.Getenv("Http_Content_Type"))
	// if os.Getenv("Http_Content_Type") != "application/json" {
	// 	return "Invalid Content Type"
	// }

	log.Println(string(req))
	if req == nil {
		log.Println("nothing to process")
		return "nothing to process"
	}

	var receivedTweet Tweet
	err := json.Unmarshal(req, &receivedTweet)
	if err != nil {
		return fmt.Sprintf("failed to decode the incomming tweet: %v", err)
	}

	err = loadConfig()
	if err != nil {
		log.Printf("failed to load the config: %v", err)
		return "Failed to load the configuration"
	}

	log.Println(receivedTweet.APIKey)
	log.Println(Config.TokenAPIKey)
	if receivedTweet.APIKey != Config.TokenAPIKey {
		return "Invalid API Key."
	}

	log.Println("***starting CloudNative Bot Slash command function***")

	api := anaconda.NewTwitterApiWithCredentials(Config.TwitterAccessToken, Config.TwitterAccessSecret, Config.TwitterConsumerKey, Config.TwitterConsumerSecretKey)

	if _, err := api.VerifyCredentials(); err != nil {
		return fmt.Sprintf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens: %v", err)
	}

	log.Printf("Checking Tweet from @%s ID = %s Text = %s\n", receivedTweet.UserScreenName, receivedTweet.IdStr, receivedTweet.Text)

	if commandPatchMatch.MatchString(receivedTweet.Text) {
		go func() {
			time.Sleep(5 * time.Second)
			_, err := api.Favorite(receivedTweet.ID)
			if err != nil {
				log.Printf("Error while trying to favorite the tweet. Err=%s\n", err.Error())
			}
		}()
	}

	if checkCNReply(api, receivedTweet.UserScreenName, receivedTweet.IdStr) {
		return "Tweet already processed"
	}

	if commandPatchMatch.MatchString(receivedTweet.Text) {
		log.Println("Tweet matched k8s-schedule-patch")

		if strings.Contains(receivedTweet.Text, "RT ") {
			log.Println("This is a RT dont reply to not flood")
			return "This is a RT no reply to not flood"
		}

		client := &getter.Client{
			Ctx:  context.Background(),
			Dst:  "/tmp/schedule",
			Dir:  true,
			Src:  "github.com/kubernetes/website.git/data/releases",
			Mode: getter.ClientModeDir,
			Detectors: []getter.Detector{
				&getter.GitHubDetector{},
			},
			Getters: map[string]getter.Getter{
				"git": &getter.GitGetter{},
			},
		}

		if err := client.Get(); err != nil {
			log.Printf("failed getting path %s: %v", client.Src, err)
			return "failed to get the k8s patch schedule"
		}

		schedule, err := ioutil.ReadFile("/tmp/schedule/schedule.yaml")
		if err != nil {
			log.Printf("failed reading %s: %v", client.Dst, err)
			return "failed to read the schedule file"
		}
		defer os.Remove("/tmp/schedule")

		var patchSchedule PatchSchedule
		err = yaml.Unmarshal(schedule, &patchSchedule)
		if err != nil {
			log.Printf("failed decode the schedules: %v", err)
			return "failed decode the schedule"
		}

		var msg strings.Builder
		fmt.Fprint(&msg, "K8s Patch Release Schedule\n\n")
		for _, release := range patchSchedule.Schedules {
			fmt.Fprintf(&msg, "Release %s\n- CP Deadline: %s\n- Target Date: %s\n\n", release.Next, release.CherryPickDeadline, release.TargetDate)
		}

		fmt.Fprintf(&msg, "source: https://github.com/kubernetes/website/blob/master/data/releases/schedule.yaml\n#K8sRelease")

		log.Println(msg.String())
		sendTweet(api, receivedTweet.IdStr, msg.String())
	}

	return "done"
}

func sendTweet(twitterAPI *anaconda.TwitterApi, originalTweetID, message string) {
	n := randomInt(30, 120)
	log.Printf("Sleeping for %s", time.Duration(n)*time.Second)
	time.Sleep(time.Duration(n) * time.Second)

	replyParams := url.Values{}
	msg := message

	replyParams.Set("in_reply_to_status_id", originalTweetID)
	replyParams.Set("auto_populate_reply_metadata", "true")
	replyParams.Set("display_coordinates", "false")
	result, err := twitterAPI.PostTweet(msg, replyParams)
	if err != nil {
		log.Printf("failed to post the tweet: %v\n", err)
	}
	log.Printf("Tweet posted. TweetID = %s\n", result.IdStr)
}

func checkCNReply(twitterAPI *anaconda.TwitterApi, userScreenName, idStr string) bool {
	searchReplyParams := url.Values{}
	searchReplyParams.Set("to", fmt.Sprintf("@%s", userScreenName))
	searchReplyParams.Set("count", Config.TwitterSearchCounts)
	searchResultReply, err := twitterAPI.GetSearch("", searchReplyParams)
	if err != nil {
		log.Printf("Error getting the search: %v", err.Error())
		return false
	}
	for _, tweetReply := range searchResultReply.Statuses {
		if tweetReply.InReplyToStatusIdStr == idStr {
			if tweetReply.User.ScreenName == "honk_bot" {
				log.Printf("already replied to this tweet ID = %s, skipping...\n", tweetReply.IdStr)
				return true
			}
		}
	}
	return false
}

// Returns an int >= min, < max
func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}
