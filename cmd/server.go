package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	timeout = 2 * time.Second
	port    = 8080
)

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")

	if slackToken == "" {
		log.Fatal("SLACK_TOKEN environment variable not set, exiting.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/invite", invite(slackToken))

	s := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	log.Info("Serving on :", port)

	log.Fatal(s.ListenAndServe())
}

func invite(slackToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		values := r.URL.Query()
		email := values.Get("email")

		if email == "" {
			log.Info("Received empty email")
			fmt.Fprintf(w, "email not found")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Infof("Email received: %s", email)

		slackURL := "https://cloud-native-nordics.slack.com/api/users.admin.invite"

		values = url.Values{
			"email":      {email},
			"token":      {slackToken},
			"set_active": {"true"},
		}

		res, err := http.PostForm(slackURL, values)
		if err != nil {
			log.Warning("HTTP Post to slack failed, this could be because the slack API is down.")
			fmt.Fprintf(w, "slack post failed: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var data map[string]interface{}
		err = json.NewDecoder(res.Body).Decode(&data)

		if err != nil {
			log.Warning("Failed to decode slack response.")
			fmt.Fprintf(w, "decode slack response failed")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if data["ok"] == false {
			if data["error"].(string) == "already_invited" || data["error"].(string) == "already_in_team" {
				log.Info("Invite successful: Already in team ")
				fmt.Fprintf(w, "Success! You were already invited.\n")
				w.WriteHeader(http.StatusOK)
				return
			} else if data["error"].(string) == "invalid_email" {
				log.Info("Invite failed: Email is invalid")
				fmt.Fprintf(w, "The email you entered is an invalid email.\n")
				w.WriteHeader(http.StatusBadRequest)
				return
			} else if data["error"].(string) == "invalid_auth" {
				log.Warning("Invite failed: Invalid auth, is the SLACK_TOKEN environment variable correct?")
				fmt.Fprintf(w, "Invalid auth: Something has gone wrong. Please contact a system administrator.\n")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Warning("Invite failed: Catch all: something has gone wrong.")
			fmt.Fprintf(w, "Catch all: Something has gone wrong. Please contact a system administrator.\n")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Info("Invite successful: Invited to team.")
		fmt.Fprintf(w, "Success! Check “%s“ for an invite from Slack.\n", email)
		w.WriteHeader(http.StatusOK)
	}
}
