mkdir -p testdata/repos/brotherlogic/Home/issues/

sleep 1
curl -X POST -H "Content-Type: application/json" --user-agent "GithubAgent" "https://api.github.com/repos/brotherlogic/Home/issues?access_token=$1" -d '{"title": "Testing", "body": "This is a test issue", "assignee": "brotherlogic"}' > testdata/repos/brotherlogic/Home/issues_access_token=token

sleep 1
curl -H "Content-Type: application/json" --user-agent "GithubAgent" "https://api.github.com/repos/brotherlogic/Home/issues/12?access_token=$1"  > testdata/repos/brotherlogic/Home/issues/12_access_token=token
