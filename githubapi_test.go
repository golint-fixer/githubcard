package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	pb "github.com/brotherlogic/githubcard/proto"
)

func InitTest() *GithubBridge {
	s := Init()
	s.getter = testFileGetter{}
	s.accessCode = "token"
	s.SkipLog = true
	return s
}

type testFileGetter struct{}

func (httpGetter testFileGetter) Post(url string, data string) (*http.Response, error) {
	log.Printf("url  %v", url)
	log.Printf("data %v", data)
	response := &http.Response{}
	strippedURL := strings.Replace(strings.Replace(url[22:], "?", "_", -1), "&", "_", -1)
	blah, err := os.Open("testdata" + strippedURL)
	if err != nil {
		log.Printf("Error opening test file %v", err)
	}
	response.Body = blah
	return response, nil
}

func (httpGetter testFileGetter) Get(url string) (*http.Response, error) {
	response := &http.Response{}
	strippedURL := strings.Replace(strings.Replace(url[22:], "?", "_", -1), "&", "_", -1)
	blah, err := os.Open("testdata" + strippedURL)
	if err != nil {
		log.Printf("Error opening test file %v", err)
	}
	response.Body = blah
	return response, nil
}

func TestAddIssue(t *testing.T) {
	issue := &pb.Issue{Title: "Testing", Body: "This is a test issue", Service: "Home"}

	s := InitTest()
	ib, err := s.AddIssue(context.Background(), issue)

	if err != nil {
		t.Fatalf("Error in adding issue: %v", err)
	}

	if ib.Number != 418 {
		t.Errorf("Issue has not been added: %v", ib)
	}
}

func TestSubmitComplexIssue(t *testing.T) {
	issue := &pb.Issue{Title: "CRASH REPORT", Service: "crasher", Body: "2017/09/26 17:48:18 ip:\"192.168.86.28\" port:50057 name:\"crasher\" identifier:\"framethree\"  is Servingpanic: Whoopsiegoroutine 41 [running]:panic(0x3b13f8, 0x109643f8)\t/usr/lib/go-1.7/src/runtime/panic.go:500 +0x33cmain.crash()\t/home/simon/gobuild/src/github.com/brotherlogic/crasher/Crasher.go:36 +0x6ccreated by github.com/brotherlogic/goserver.(*GoServer).Serve\t/home/simon/gobuild/src/github.com/brotherlogic/goserver/goserverapi.go:126+0x254"}
	s := InitTest()
	ib, err := s.AddIssue(context.Background(), issue)

	if err != nil {
		t.Fatalf("Error in adding issue: %v", err)
	}

	if ib.Number != 418 {
		t.Errorf("Issue has not been added: %v", ib)
	}
}

func TestGetIssue(t *testing.T) {
	s := InitTest()
	ib, err := s.Get(context.Background(), &pb.Issue{Service: "Home", Number: 12})

	if err != nil {
		t.Fatalf("Error in getting issue: %v", err)
	}

	if ib.Number != 12 {
		t.Errorf("Issue has not been returned correctly: %v", ib)
	}
}
