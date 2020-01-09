package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// structure for app request json send from containerdesktop
type appRequest struct {
	User string `json:"user"`
	App  string `json:"app"`
}

// AppProvider use: curl -X POST localhost -H 'content-type: application/json' -d '{"user":"'$USER'","app":"'$APP'"}'
func AppProvider(writer http.ResponseWriter, request *http.Request) {

	// Read body of curl request into bytes format
	body, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	// Unmarshal body of curl request into interface
	curlRequest := appRequest{}
	err = json.Unmarshal(body, &curlRequest)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}

	// output
	output, err := json.Marshal(curlRequest.App)
	if err != nil {
		http.Error(writer, err.Error(), 500)
		return
	}
	writer.Header().Set("content-type", "application/json")
	writer.Write(output)

	// variables
	baseDir := "/opt/vcde/"
	requestUser := curlRequest.User
	requestApp := curlRequest.App
	netWork := "vcd_frontend"
	ctx := context.Background()
	apps := []string{"gimp", "firefox", "chrome", "libreoffice", "evolution", "geary", "thunderbird", "nautilus"}
	appImage := make(map[string]string)
	for _, i := range apps {
		appImage[i] = "rayvtoll/vcd-" + i + ":latest"
	}

	// testprint the results
	fmt.Println("user: ", requestUser)
	fmt.Println("app: ", requestApp)

	// docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// shmSize and securityOpt are different per application containers
	var shmSize int64
	var securityOpt []string
	switch requestApp {
	case "chrome":
		// 2g
		shmSize = 2147483648

		// This code is working, I have no idea why (yet)..
		jsonFile, err := os.Open("/app/chrome.json")
		if err != nil {
			panic(err)
		}
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		var result interface{}
		err = json.Unmarshal([]byte(byteValue), &result)
		if err != nil {
			panic(err)
		}
		b, _ := json.Marshal(result)
		// fmt.Println(string(b))
		securityOpt = []string{"seccomp=" + string(b)}
	case "firefox", "gimp":
		// 2g
		shmSize = 2147483648
		securityOpt = []string{}
	default:
		shmSize = 67108864
		securityOpt = []string{}
	}

	// docker run command constructor with docker syntax in comments
	resp, err := cli.ContainerCreate(
		ctx,
		// image, --hostname, -e, -t
		&container.Config{Image: appImage[requestApp], Hostname: "vcd-" + requestUser + "-" + requestApp, Env: []string{"USER=" + requestUser}, Tty: false},
		// -v, --rm,
		&container.HostConfig{
			Binds: []string{
				baseDir + requestUser + ":/home/" + requestUser,
				baseDir + "Public:/home/" + requestUser + "/Public",
				baseDir + requestUser + "/.ssh/id_rsa.pub:/home/" + requestUser + "/.ssh/authorized_keys:ro",
			},
			ShmSize:     shmSize,
			SecurityOpt: securityOpt,
			AutoRemove:  true,
		},
		// --network
		&networktypes.NetworkingConfig{
			EndpointsConfig: map[string]*networktypes.EndpointSettings{
				netWork: {},
			},
		},
		// --name
		"vcd-"+requestUser+"-"+requestApp,
	)
	if err != nil {
		panic(err)
	}

	// start the desktopcontainer
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
}

func main() {
	http.HandleFunc("/", AppProvider)
	fmt.Println("Starting server on port 80")
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		panic(err)
	}
}
