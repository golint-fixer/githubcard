mkdir -p testdata/repos/brotherlogic/Home/issues/

sleep 1
curl -X POST -H "Content-Type: application/json" --user-agent "GithubAgent" "https://api.github.com/repos/brotherlogic/Home/issues?access_token=$1" -d '{"title": "Testing", "body": "This is a test issue", "assignee": "brotherlogic"}' > testdata/repos/brotherlogic/Home/issues_access_token=token

sleep 1
curl -H "Content-Type: application/json" --user-agent "GithubAgent" "https://api.github.com/repos/brotherlogic/Home/issues/12?access_token=$1"  > testdata/repos/brotherlogic/Home/issues/12_access_token=token

sleep 1
curl -X POST -H "Content-Type: application/json" --user-agent "GithubAgent" "https://api.github.com/repos/brotherlogic/crasher/issues?access_token=$1" -d '{"title": "Crash Report", "body": "2017/09/26 17:48:18 ip:\"192.168.86.28\" port:50057 name:\"crasher\" identifier:\"framethree\"  is Servingpanic: Whoopsiegoroutine 41 [running]:panic(0x3b13f8, 0x109643f8)\t/usr/lib/go-1.7/src/runtime/panic.go:500 +0x33cmain.crash()\t/home/simon/gobuild/src/github.com/brotherlogic/crasher/Crasher.go:36 +0x6ccreated by github.com/brotherlogic/goserver.(*GoServer).Serve\t/home/simon/gobuild/src/github.com/brotherlogic/goserver/goserverapi.go:126+0x254", "assignee": "brotherlogic"}' > testdata/repos/brotherlogic/crasher/issues_access_token=token
