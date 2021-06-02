package function

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

var (
	commandCNMatch = regexp.MustCompile(`(?mi)^*/(?:cloudnativetv)(?: +(.+?))?\s*$`)
)

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

	if receivedTweet.APIKey != Config.TokenAPIKey {
		return "Invalid API Key."
	}

	log.Println("***starting CloudNative Bot Slash command function***")

	api := anaconda.NewTwitterApiWithCredentials(Config.TwitterAccessToken, Config.TwitterAccessSecret, Config.TwitterConsumerKey, Config.TwitterConsumerSecretKey)

	if _, err := api.VerifyCredentials(); err != nil {
		return fmt.Sprintf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens: %v", err)
	}

	log.Printf("Checking Tweet from @%s ID = %s Text = %s\n", receivedTweet.UserScreenName, receivedTweet.IdStr, receivedTweet.Text)

	if commandCNMatch.MatchString(receivedTweet.Text) {
		go func() {
			n := randomInt(30, 120)
			time.Sleep(time.Duration(n) * time.Second)
			_, err := api.Favorite(receivedTweet.ID)
			if err != nil {
				log.Printf("Error while trying to favorite the tweet. Err=%s\n", err.Error())
			}
		}()
	}

	if checkCNReply(api, receivedTweet.UserScreenName, receivedTweet.IdStr) {
		return "Tweet already processed"
	}

	if commandCNMatch.MatchString(receivedTweet.Text) {
		log.Println("Tweet matched cloudnativetv")

		if strings.Contains(receivedTweet.Text, "RT ") {
			log.Println("This is a RT dont reply to not flood")
			return ""
		}

		list := []string{"cncf100Days", "cncfFaceOff", "cncfKat", "cncfLatinX", "cncfLGTM", "cncfSolidState",
			"cncfAwesomeCerts", "cncfFieldTested", "cncfSpotlightLive", "cncfThisWeekCN"}
		choose := list[rand.Intn(len(list))]

		msg := "Follow/Subscribe #100Days #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		choosedImage := cncf100DaysImage

		switch choose {
		case "cncf100Days":
			choosedImage = cncf100DaysImage
		case "cncfFaceOff":
			choosedImage = cncfFaceOffImage
			msg = "Follow/Subscribe #CNCFFaceOff #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfKat":
			choosedImage = cncfKatImage
			msg = "Follow/Subscribe #Kat #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfLatinX":
			choosedImage = cncfLatinXImage
			msg = "Follow/Subscribe #LatinX #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfLGTM":
			choosedImage = cncfLGTMImage
			msg = "Follow/Subscribe #LGTM #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfSolidState":
			choosedImage = cncfSolidStateImage
			msg = "Follow/Subscribe #SolidState #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfAwesomeCerts":
			choosedImage = cncfAwesomeCertsImage
			msg = "Follow/Subscribe #CNCFAwesomeCerts #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfFieldTested":
			choosedImage = cncfFieldTestedImage
			msg = "Follow/Subscribe #FieldTested #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfSpotlightLive":
			choosedImage = cncfSpotlightLiveImage
			msg = "Follow/Subscribe #SpotlightLive #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		case "cncfThisWeekCN":
			choosedImage = cncfThisWeekCNImage
			msg = "Follow/Subscribe #ThisWeekCNCF #CloudNativeTV #cloudnativebot\nhttps://www.twitch.tv/cloudnativefdn"
		default:
			choosedImage = cncf100DaysImage
		}

		sendTweet(api, receivedTweet.IdStr, msg, choosedImage)
		// cnTypeProcessed.WithLabelValues(choose).Add(1)
	}

	return "done"
}

func sendTweet(twitterAPI *anaconda.TwitterApi, originalTweetID, message string, image []byte) {
	n := randomInt(30, 120)
	log.Printf("Sleeping for %s", time.Duration(n)*time.Second)
	time.Sleep(time.Duration(n) * time.Second)

	replyParams := url.Values{}
	msg := message

	if image != nil {
		mediaResponse, err := twitterAPI.UploadMedia(base64.StdEncoding.EncodeToString(image))
		if err != nil {
			log.Printf("Error uploading the image Err=%s\n", err.Error())
		} else {
			replyParams.Set("media_ids", mediaResponse.MediaIDString)
		}
	}

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
