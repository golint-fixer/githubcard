package main

import (
	"encoding/json"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	pb "github.com/brotherlogic/cardserver/card"
)

func visitURL(urlv string) string {
	resp, err := http.Get(urlv)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	return string(body)
}

// GetIssues Gets github issues for a given project
func GetIssues(project string) pb.CardList {
	cardlist := pb.CardList{}
	urlv := "https://api.github.com/repos/" + project + "/issues?state=open"
	body := visitURL(urlv)

	var data []interface{}
	err := json.Unmarshal([]byte(body), &data)
	if err != nil {
		panic(err)
	}

	for _, issue := range data {
		issueMap := issue.(map[string]interface{})

		issueSource := issueMap["url"].(string)
		issueTitle := issueMap["title"].(string)
		issueText := issueMap["body"].(string)

		card := &pb.Card{}
		card.Text = issueTitle + "\n" + issueText + "\n\n" + issueSource
		cardlist.Cards = append(cardlist.Cards, card)
	}

	return cardlist
}

func main() {
	issues := GetIssues("brotherlogic/cardserver")

	log.Printf("Got issues: %v", issues)
	os.Exit(1)

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	client := pb.NewCardServiceClient(conn)
	_, err = client.AddCards(context.Background(), &issues)
	if err != nil {
		log.Printf("Problem adding cards %v", err)
	}
}
