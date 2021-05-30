package function

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ChimeraCoder/anaconda"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var (
	commandKubeconMatch = regexp.MustCompile(`(?mi)^*/(?:kubecon-random-video)(?: +(.+?))?\s*$`)
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

	if receivedTweet.APIKey != Config.TokenAPIKey {
		return "Invalid API Key."
	}

	log.Println("***starting CloudNative Bot Slash command function***")

	api := anaconda.NewTwitterApiWithCredentials(Config.TwitterAccessToken, Config.TwitterAccessSecret, Config.TwitterConsumerKey, Config.TwitterConsumerSecretKey)

	if _, err := api.VerifyCredentials(); err != nil {
		return fmt.Sprintf("Bad Authorization Tokens. Please refer to https://apps.twitter.com/ for your Access Tokens: %v", err)
	}

	log.Printf("Checking Tweet from @%s ID = %s Text = %s\n", receivedTweet.UserScreenName, receivedTweet.IdStr, receivedTweet.Text)

	if commandKubeconMatch.MatchString(receivedTweet.Text) {
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

	if commandKubeconMatch.MatchString(receivedTweet.Text) {
		log.Println("Tweet matched kubecon-random-video")

		if strings.Contains(receivedTweet.Text, "RT ") {
			log.Println("This is a RT dont reply to not flood")
			return "This is a RT no reply to not flood"
		}

		playlistIDs := []string{"PLj6h78yzYM2MqBm19mRz9SYLsw4kfQBrC", "PLj6h78yzYM2NDs-iu8WU5fMxINxHXlien", "PLj6h78yzYM2Pn8RxfLh2qrXBDftr6Qjut", "PLj6h78yzYM2O1wlsM-Ma-RYhfT5LKq0XC"}

		choosePlayList := playlistIDs[rand.Intn(len(playlistIDs))]
		log.Printf("trying to get a video from %s playlist", choosePlayList)
		randomVideo := getRandomYoutubeVideoFromPlayList("UCvqbFHwN-nwalWPjPUKpvTA", choosePlayList)
		log.Printf("got video %s", randomVideo)
		if randomVideo == "" {
			randomVideo = "ZT1PXt87qSs"
		}

		msg := fmt.Sprintf("Hope you enjoy this talk! #cloudnative\nhttps://www.youtube.com/watch?v=%s", randomVideo)

		sendTweet(api, receivedTweet.IdStr, msg)
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

func getRandomYoutubeVideoFromPlayList(channelID, playListID string) string {
	log.Printf("yutbua api: %s", Config.YouTubeAPIKey)
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(Config.YouTubeAPIKey))
	if err != nil {
		log.Printf("Error creating new YouTube client: %v", err)
		return ""
	}

	response, err := service.Playlists.List([]string{}).ChannelId(channelID).MaxResults(50).Do()
	if err != nil {
		log.Printf("Error creating getting the youtube video search: %s\n", err.Error())
		return ""
	}

	for _, playList := range response.Items {
		if playList.Id == playListID {
			videos, err := service.PlaylistItems.List([]string{"contentDetails"}).MaxResults(500).PlaylistId(playList.Id).Do()
			if err != nil {
				log.Printf("Error creating getting the youtube video search: %s\n", err.Error())
				return ""
			}

			if len(videos.Items) != 0 {
				items := videos.Items
				log.Printf("Got youtube videos %v", items)
				return items[rand.Intn(len(items))].ContentDetails.VideoId
			}
		}
	}

	return ""
}
