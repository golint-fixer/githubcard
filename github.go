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

	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/cardserver/card"
	pbdi "github.com/brotherlogic/discovery/proto"
	pbgh "github.com/brotherlogic/githubcard/proto"
	"github.com/brotherlogic/goserver"
)

// GithubBridge the bridge to the github API
type GithubBridge struct {
	*goserver.GoServer
	accessCode string
	serving    bool
}

//Init a record getter
func Init() *GithubBridge {
	s := &GithubBridge{GoServer: &goserver.GoServer{}, serving: true}
	s.Register = s
	return s
}

// DoRegister does RPC registration
func (b GithubBridge) DoRegister(server *grpc.Server) {
	// Noop
}

// ReportHealth alerts if we're not healthy
func (b GithubBridge) ReportHealth() bool {
	log.Printf("REPORTING HEALTH")
	return true
}

// Mote promotes this server
func (b GithubBridge) Mote(master bool) error {
	return nil
}

const (
	wait = time.Minute // Wait one minute between runs
)

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
	ip, port := getIP("cardserver")
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

func getIP(servername string) (string, int) {
	conn, _ := grpc.Dial("192.168.86.64:50055", grpc.WithInsecure())
	defer conn.Close()

	registry := pbdi.NewDiscoveryServiceClient(conn)
	entry := pbdi.RegistryEntry{Name: servername}
	r, err := registry.Discover(context.Background(), &entry)
	if err != nil {
		return "", -1
	}
	return r.Ip, int(r.Port)
}

// RunPass runs a pass over
func (b GithubBridge) RunPass() {
	for b.serving {
		time.Sleep(wait)
		err := b.passover()
		if err != nil {
			log.Printf("FAILED to run: %v", err)
		}
	}

	log.Printf("Ducking out of serving")
}

func (b GithubBridge) passover() error {
	log.Printf("RUNNING PASSOVER")
	ip, port := getIP("cardserver")
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
			b.AddIssue("brotherlogic", strings.Split(card.Hash, "-")[2], strings.Split(card.Text, "|")[0], strings.Split(card.Text, "|")[1])
		}
	}

	log.Printf("Deleting cards: %v", &pb.DeleteRequest{HashPrefix: "addgithubissue"})
	_, err = client.DeleteCards(context.Background(), &pb.DeleteRequest{HashPrefix: "addgithubissue"})
	log.Printf("HERE %v", err)
	if err != nil {
		return err
	}

	log.Printf("Doing project call")
	projects := b.GetProjects()
	issues := pb.CardList{}
	log.Printf("Getting projects")
	for _, project := range projects {
		log.Printf("Getting issues for %v", project.Name)
		tempIssues := b.GetIssues("brotherlogic/" + project.Name)
		issues.Cards = append(issues.Cards, tempIssues.Cards...)
	}

	_, err = client.DeleteCards(context.Background(), &pb.DeleteRequest{HashPrefix: "githubissue"})
	if err != nil {
		log.Printf("Error deleting cards")
	}
	_, err = client.AddCards(context.Background(), &issues)
	if err != nil {
		log.Printf("Problem adding cards %v", err)
	} else {
		log.Printf("Would write: %v", issues)
	}

	return nil
}

func main() {
	var quiet = flag.Bool("quiet", true, "Show all output")
	var token = flag.String("token", "", "The token to use to auth")
	flag.Parse()

	b := Init()
	b.GoServer.KSclient = *keystoreclient.GetClient()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	b.PrepServer()
	b.RegisterServer("githubcard", false)

	if len(*token) > 0 {
		b.Save("/github.com/brotherlogic/githubcard/token", &pbgh.Token{Token: *token})
	} else {
		m, err := b.Read("/github.com/brotherlogic/githubcard/token", &pbgh.Token{})
		if err != nil {
			log.Printf("Failed to read token: %v", err)
		} else {
			log.Printf("GOT TOKEN: %v", m)
			b.accessCode = m.(*pbgh.Token).GetToken()
			b.RegisterServingTask(b.RunPass)
			b.Serve()
		}
	}
}
