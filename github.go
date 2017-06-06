package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/brotherlogic/cardserver/card"
	pbdi "github.com/brotherlogic/discovery/proto"
)

// GithubBridge the bridge to the github API
type GithubBridge struct {
	accessCode string
}

func (b *GithubBridge) postURL(urlv string, data string) string {
	url := urlv
	if len(b.accessCode) > 0 && strings.Contains(urlv, "?") {
		url = url + "&access_token=" + b.accessCode
	} else {
		url = url + "?access_token=" + b.accessCode
	}

	log.Printf("Posting: %v and %v", url, data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(data)))

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	log.Printf("RETURN: %v", string(body))
	return string(body)
}

func (b *GithubBridge) visitURL(urlv string) (string, error) {

	url := urlv
	if len(b.accessCode) > 0 && strings.Contains(urlv, "?") {
		url = url + "&access_token=" + b.accessCode
	} else {
		url = url + "?access_token=" + b.accessCode
	}

	log.Printf("Visiting: %v", url)
	resp, err := http.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Project is a project in the github world
type Project struct {
	Name string
}

// GetProjects from github
func (b *GithubBridge) GetProjects() []Project {
	list, err := b.visitURL("https://api.github.com/user/repos?per_page=100")
	var projects []Project
	if err != nil {
		return projects
	}
	json.Unmarshal([]byte(list), &projects)
	return projects
}

// AddIssue adds an issue
func (b *GithubBridge) AddIssue(owner, repo, title, body string) {
	ip, port := getIP("cardserver", "192.168.86.64", 50055)
	conn, err := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	data := "{\"title\": \"" + title + "\", \"body\": \"" + body + "\", \"assignee\": \"" + owner + "\"}"
	urlv := "https://api.github.com/repos/" + owner + "/" + repo + "/issues"
	b.postURL(urlv, data)
}

func hash(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}

// GetIssues Gets github issues for a given project
func (b *GithubBridge) GetIssues(project string) pb.CardList {
	cardlist := pb.CardList{}
	urlv := "https://api.github.com/repos/" + project + "/issues?state=open"
	body, err := b.visitURL(urlv)

	if err != nil {
		return cardlist
	}

	var data []interface{}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		panic(err)
	}

	for _, issue := range data {
		issueMap := issue.(map[string]interface{})

		if _, ok := issueMap["pull_request"]; !ok {
			issueSource := issueMap["url"].(string)
			issueTitle := issueMap["title"].(string)
			issueText := issueMap["body"].(string)

			date, err := time.Parse("2006-01-02T15:04:05Z", issueMap["created_at"].(string))

			if err != nil {
				log.Printf("Error reading dates: %v", err)
			}

			card := &pb.Card{}
			card.Text = issueTitle + "\n" + issueText + "\n\n" + issueSource
			card.Hash = "githubissue-" + issueSource
			card.Channel = pb.Card_ISSUES
			log.Printf("CHECKING %v %v %v (%v)", time.Now(), date, time.Now().Sub(date), issueTitle)
			log.Printf("FROM %v", issueMap)
			card.Priority = int32(time.Now().Sub(date)/time.Second) + hash(card.Text)%1000
			log.Printf("CHECKING PR %v", card.Priority)
			cardlist.Cards = append(cardlist.Cards, card)
		}
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
	var addissues = flag.Bool("issues", false, "Just create issues")
	flag.Parse()

	if *quiet {
		log.SetOutput(ioutil.Discard)
		grpclog.SetLogger(log.New(ioutil.Discard, "", -1))
	}

	bridge := GithubBridge{accessCode: *token}

	log.Printf("HERE %v", addissues)
	if *addissues {
		ip, port := getIP("cardserver", "192.168.86.64", 50055)
		conn, err := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Error here: %v", err)
		}
		defer conn.Close()
		client := pb.NewCardServiceClient(conn)
		cards, err := client.GetCards(context.Background(), &pb.Empty{})
		if err != nil {
			log.Fatalf("Error here: %v", (err))
		}

		for _, card := range cards.Cards {
			log.Printf("CARD = %v", card.Hash)
			if strings.HasPrefix(card.Hash, "addgithubissue") {
				bridge.AddIssue("brotherlogic", strings.Split(card.Hash, "-")[2], strings.Split(card.Text, "|")[0], strings.Split(card.Text, "|")[1])
			}
		}

		_, err = client.DeleteCards(context.Background(), &pb.DeleteRequest{HashPrefix: "addgithubissue"})
		if err != nil {
			panic(err)
		}
	}

	projects := bridge.GetProjects()
	issues := pb.CardList{}
	for _, project := range projects {
		tempIssues := bridge.GetIssues("brotherlogic/" + project.Name)
		issues.Cards = append(issues.Cards, tempIssues.Cards...)
	}

	if !*dryRun {
		ip, port := getIP("cardserver", "192.168.86.64", 50055)
		conn, err := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		defer conn.Close()
		client := pb.NewCardServiceClient(conn)
		_, err = client.DeleteCards(context.Background(), &pb.DeleteRequest{HashPrefix: "githubissue"})
		if err != nil {
			log.Printf("Error deleting cards")
		}
		_, err = client.AddCards(context.Background(), &issues)
		if err != nil {
			log.Printf("Problem adding cards %v", err)
		}
	} else {
		log.Printf("Would write: %v", issues)
	}

}
