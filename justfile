image := "ssl-checker"
builder_image := "golang:latest"

build:
  docker build -t {{image}}:latest -f Containerfile .

run-env:
    docker run --rm -t -v "${PWD}:/app" \
    -e "gitlabtoken=glpat-4qFDw3SDiu5QlSlnlYXrnG86MQp1OmprNGM0Cw.01.120kyufj7" \
    -e "pagerdutykey=" \
    -e "type=config"
    -e "gitlaburl=https://gitlab.com" \
    -e "gitlabprojectid=77549676" \
    -e "gitlabfilepath=test.json" \
    -e "gitlabref=main" \
    -e "verbose=true" \
    -e "timeout=4" \
    -e "split=20" \
    -e "outputfile=/app/test" \
    -e "alertdays=365" \
    {{image}}:latest

run-test:
  docker run --rm -it -v "${PWD}:/app" \
  -e "verbose=true" \
  -e "timeout=4" \
  -e "split=20" \
  {{image}}:latest \
  --config /app/test.json \
  --type config \
  --alertdays 365 \
  --outputfile /app/test

test: 
  docker run --rm -it \
    -v "${PWD}:/app" \
    -w /app/src \
    {{builder_image}} \
    go test ./config/... -v

tidy:
    docker run --rm -it \
      -v "${PWD}:/app" \
      -w /app/ \
      {{builder_image}} \
      go mod tidy 
      

get:
    docker run --rm -it \
      -v "${PWD}:/app" \
      -w /app/src \
      {{builder_image}} \
      go get \
      github.com/3th1nk/cidr  \
      github.com/peterbourgon/ff/v3 \
      github.com/aws/aws-sdk-go/aws\
      github.com/aws/aws-sdk-go/aws/session\
      github.com/aws/aws-sdk-go/service/route53\
      github.com/cloudflare/cloudflare-go\
      gitlab.com/gitlab-org/api/client-go\

clean:
    docker run --rm -it \
      -v "${PWD}:/app" \
      -w /app/src \
      {{builder_image}} \
      go clean -modcache
      
sign:
  git config --global user.signingkey 4C86C3BE2A5F2A50BFA4CF55975968D3C75F4CC5
  git config --global commit.gpgsign true
  git config --global tag.gpgSign true
  git config --global user.name "Andre Koster"
