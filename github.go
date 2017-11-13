package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/cardserver/card"
	pbgh "github.com/brotherlogic/githubcard/proto"
	pbgs "github.com/brotherlogic/goserver/proto"
)

// GithubBridge the bridge to the github API
type GithubBridge struct {
	*goserver.GoServer
	accessCode string
	serving    bool
	getter     httpGetter
}

type httpGetter interface {
	Post(url string, data string) (*http.Response, error)
	Get(url string) (*http.Response, error)
}

type prodHTTPGetter struct{}

func (httpGetter prodHTTPGetter) Post(url string, data string) (*http.Response, error) {
	return http.Post(url, "application/json", bytes.NewBuffer([]byte(data)))
}

func (httpGetter prodHTTPGetter) Get(url string) (*http.Response, error) {
	return http.Get(url)
}

//Init a record getter
func Init() *GithubBridge {
	s := &GithubBridge{GoServer: &goserver.GoServer{}, serving: true, getter: prodHTTPGetter{}}
	s.Register = s
	return s
}

// DoRegister does RPC registration
func (b *GithubBridge) DoRegister(server *grpc.Server) {
	pbgh.RegisterGithubServer(server, b)
}

// ReportHealth alerts if we're not healthy
func (b GithubBridge) ReportHealth() bool {
	return true
}

// Mote promotes this server
func (b GithubBridge) Mote(master bool) error {
	return nil
}

// GetState gets the state of the server
func (b GithubBridge) GetState() []*pbgs.State {
	return []*pbgs.State{}
}

const (
	wait = 5 * time.Minute // Wait five minute between runs
)

func (b *GithubBridge) postURL(urlv string, data string) (*http.Response, error) {
	url := urlv
	if len(b.accessCode) > 0 && strings.Contains(urlv, "?") {
		url = url + "&access_token=" + b.accessCode
	} else {
		url = url + "?access_token=" + b.accessCode
	}

	b.Log(fmt.Sprintf("POST: %v", url))
	return b.getter.Post(url, data)
}

func (b *GithubBridge) visitURL(urlv string) (string, error) {

	url := urlv
	if len(b.accessCode) > 0 && strings.Contains(urlv, "?") {
		url = url + "&access_token=" + b.accessCode
	} else {
		url = url + "?access_token=" + b.accessCode
	}

	b.Log(fmt.Sprintf("GET: %v", url))
	resp, err := b.getter.Get(url)
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

func (b *GithubBridge) issueExists(title string) (*pbgh.Issue, error) {
	urlv := "https://api.github.com/user/issues"
	body, err := b.visitURL(urlv)

	b.Log(fmt.Sprintf("Checked %v -> %v", err, string(body)))

	if err != nil {
		return nil, err
	}

	var data []interface{}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}

	for _, d := range data {
		dp := d.(map[string]interface{})
		if dp["title"].(string) == title {
			return &pbgh.Issue{Title: title}, nil
		}
	}

	return nil, nil
}

// AddIssueLocal adds an issue
func (b *GithubBridge) AddIssueLocal(owner, repo, title, body string) ([]byte, error) {
	issue, _ := b.issueExists(title)
	if issue != nil {
		return nil, errors.New("Issue already exists")
	}

	data := fmt.Sprintf("{\"title\": \"%s\", \"body\": \"%s\", \"assignee\": \"%s\"}", title, strings.Replace(strings.Replace(body, "\t", " ", -1), "\n", " ", -1), owner)
	urlv := "https://api.github.com/repos/" + owner + "/" + repo + "/issues"
	resp, err := b.postURL(urlv, data)
	if err != nil {
		b.Log(fmt.Sprintf("Writing issues has failed: %v", err))
		return nil, err
	}

	defer resp.Body.Close()
	rb, _ := ioutil.ReadAll(resp.Body)

	log.Printf("SOURCE data: " + data)
	b.Log(fmt.Sprintf("From %s and %s", data, urlv))
	b.Log("Read " + string(rb))

	return rb, nil
}

func hash(s string) int32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int32(h.Sum32())
}

// GetIssueLocal Gets github issues for a given project
func (b *GithubBridge) GetIssueLocal(owner string, project string, number int) (*pbgh.Issue, error) {
	urlv := "https://api.github.com/repos/" + owner + "/" + project + "/issues/" + strconv.Itoa(number)
	body, err := b.visitURL(urlv)

	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(body), &data)
	if err != nil {
		return nil, err
	}

	log.Printf("HERE: (%v %v) %v", data["state"].(string), data["state"].(string) == "open", data)
	issue := &pbgh.Issue{Number: int32(number), Service: project, Title: data["title"].(string), Body: data["body"].(string)}
	if data["state"].(string) == "open" {
		issue.State = pbgh.Issue_OPEN
	} else {
		issue.State = pbgh.Issue_CLOSED
	}

	log.Printf("ISSUE = %v", issue)
	return issue, nil
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
		return cardlist
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
			card.Priority = int32(time.Now().Sub(date)/time.Second) + hash(card.Text)%1000
			cardlist.Cards = append(cardlist.Cards, card)
		}
	}

	return cardlist
}

// RunPass runs a pass over
func (b GithubBridge) RunPass() {
	for b.serving {
		time.Sleep(wait)
		if b.GoServer.Registry.Master {
			err := b.passover()
			if err != nil {
				log.Printf("FAILED to run: %v", err)
			}
		}
	}

	log.Printf("Ducking out of serving")
}

func (b GithubBridge) passover() error {
	log.Printf("RUNNING PASSOVER")
	ip, port := b.GetIP("cardserver")
	conn, err := grpc.Dial(ip+":"+strconv.Itoa(port), grpc.WithInsecure())
	if err != nil {
		log.Printf("Error here: %v", err)
		return err
	}
	defer conn.Close()
	client := pb.NewCardServiceClient(conn)
	cards, err := client.GetCards(context.Background(), &pb.Empty{})
	if err != nil {
		log.Printf("Error here: %v", (err))
		return err
	}

	for _, card := range cards.Cards {
		if strings.HasPrefix(card.Hash, "addgithubissue") {
			b.AddIssueLocal("brotherlogic", strings.Split(card.Hash, "-")[2], strings.Split(card.Text, "|")[0], strings.Split(card.Text, "|")[1])
		}
	}

	_, err = client.DeleteCards(context.Background(), &pb.DeleteRequest{HashPrefix: "addgithubissue"})
	if err != nil {
		return err
	}

	log.Printf("Doing project call")
	projects := b.GetProjects()
	issues := pb.CardList{}
	log.Printf("Getting projects")
	for _, project := range projects {
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
	b.GoServer.KSclient = *keystoreclient.GetClient(b.GetIP)

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
		m, _, err := b.Read("/github.com/brotherlogic/githubcard/token", &pbgh.Token{})
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
