package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
)

const (
	KindNamePush  = "push"
	KindNameMerge = "merge_request"
)

var executerQueue chan struct{}

func init() {
	executerQueue = make(chan struct{}, 1)
}

type requestBody struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		DefaultBranch     string `json:"default_branch"`
		GitHTTPURL        string `json:"git_http_url"`
		GitSSHURL         string `json:"git_ssh_url"`
		HTTPURL           string `json:"http_url"`
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		PathWithNamespace string `json:"path_with_namespace"`
		SSHURL            string `json:"ssh_url"`
		URL               string `json:"url"`
		VisibilityLevel   int64  `json:"visibility_level"`
		WebURL            string `json:"web_url"`
	} `json:"project"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// fmt.Printf("Method: %v, Path: %v\n", r.Method, r.URL.Path)
		if r.Method == http.MethodPost {
			HandleGitlabHooks(r)
		}
		w.Write([]byte("OK"))
	})
	listenPort := os.Getenv("LISTEN_PORT")
	if listenPort == "" {
		listenPort = "8000"
	}
	fmt.Printf("Listen http://0.0.0.0:%s/\n", listenPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil); err != nil {
		fmt.Println(err.Error())
	}
}

func HandleGitlabHooks(r *http.Request) {
	var (
		data []byte
		err  error
	)
	if data, err = ioutil.ReadAll(r.Body); err != nil {
		return
	}
	reqbdy := requestBody{}
	if err = json.Unmarshal(data, &reqbdy); err != nil {
		return
	}
	if reqbdy.ObjectKind != KindNamePush && reqbdy.ObjectKind != KindNameMerge {
		return
	}
	if bakpath := getProjectBackupDir(&reqbdy); bakpath != "" {
		go func() {
			executerQueue <- struct{}{}
			backupDefaultBranch(bakpath, &reqbdy)
		}()
	}
}

func getProjectBackupDir(reqbdy *requestBody) string {
	if reqbdy.Project.PathWithNamespace == "" {
		return ""
	}
	var basePath string
	if basePath = os.Getenv("SG_HOOK_BACKUP_DIR"); basePath == "" {
		basePath = "/tmp/sg_hook_backup"
	}
	return fmt.Sprintf("%s_%d", path.Join(basePath, reqbdy.Project.PathWithNamespace), reqbdy.Project.ID)
}

func backupDefaultBranch(bakpath string, reqbdy *requestBody) {
	defer func() {
		<-executerQueue
	}()

	script := fmt.Sprintf(
		"[ ! -d '%s' ] && mkdir -p '%s'; cd %s; if [ -z `ls` ];then git clone %s; else cd %s; git checkout %s; git pull; fi",
		bakpath,
		bakpath,
		bakpath,
		reqbdy.Project.GitSSHURL,
		reqbdy.Project.Name,
		reqbdy.Project.DefaultBranch,
	)
	executer := exec.Command("/bin/bash", "-c", script)
	if err := executer.Run(); err != nil {
		fmt.Fprintf(os.Stderr,
			"project: %s branch: %s, backup failed, error: %v\n",
			reqbdy.Project.Name,
			reqbdy.Project.DefaultBranch,
			err.Error())
		return
	}
	fmt.Printf("project: %s branch: %s, backup to <%s> successful\n",
		reqbdy.Project.Name,
		reqbdy.Project.DefaultBranch,
		bakpath)
}
