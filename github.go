package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/cardserver/card"
	pbdi "github.com/brotherlogic/discovery/proto"
)

// GithubBridge the bridge to the github API
type GithubBridge struct {
	accessCode string
}

func (b *GithubBridge) visitURL(urlv string) string {

	url := urlv
	if len(b.accessCode) > 0 && strings.Contains(urlv, "?") {
		url = url + "&access_token=" + b.accessCode
	} else {
		url = url + "?access_token=" + b.accessCode
	}

	log.Printf("Visiting: %v", url)
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	return string(body)
}

// Project is a project in the github world
type Project struct {
	Name string
}

// GetProjects from github
func (b *GithubBridge) GetProjects() []Project {
	list := b.visitURL("https://api.github.com/user/repos?per_page=100")
	var projects []Project
	json.Unmarshal([]byte(list), &projects)
	return projects
}

// GetIssues Gets github issues for a given project
func (b *GithubBridge) GetIssues(project string) pb.CardList {
	cardlist := pb.CardList{}
	urlv := "https://api.github.com/repos/" + project + "/issues?state=open"
	log.Printf("VISITING %v", urlv)
	body := b.visitURL(urlv)

	log.Printf("BODY = %v", body)

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
		card.Hash = issueSource
		card.Channel = pb.Card_ISSUES
		cardlist.Cards = append(cardlist.Cards, card)
	}

	return cardlist
}

func getIP(servername string, ip string, port int) (string, int) {
	conn, _ := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	entry := pbdi.RegistryEntry{Name: servername}
	r, _ := registry.Discover(context.Background(), &entry)
	return r.Ip, int(r.Port)
}

func main() {
	var token = flag.String("token", "", "Token for auth")
	var dryRun = flag.Bool("dryrun", false, "Whether to run in dry run mode")
	var quiet = flag.Bool("quiet", true, "Suppress logging output")
	flag.Parse()

	if *quiet {
		log.SetOutput(ioutil.Discard)
	}

	bridge := GithubBridge{accessCode: *token}

	projects := bridge.GetProjects()

	issues := pb.CardList{}
	for _, project := range projects {
		tempIssues := bridge.GetIssues("brotherlogic/" + project.Name)
		issues.Cards = append(issues.Cards, tempIssues.Cards...)
	}

	if !*dryRun {
		ip, port := getIP("cardserver", "10.0.1.17", 50055)
		conn, err := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		client := pb.NewCardServiceClient(conn)
		_, err = client.AddCards(context.Background(), &issues)
		if err != nil {
			log.Printf("Problem adding cards %v", err)
		}
	} else {
		log.Printf("Would write: %v", issues)
	}
}
