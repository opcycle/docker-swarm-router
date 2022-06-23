package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/go-cmp/cmp"
)

type ServiceEntry struct {
	ServiceName         string
	ServiceDomain       string
	ServicePath         string
	ServicePort         string
	ServiceMaxBodySize  string
	ServiceProxyTimeout string
}

type Router struct {
	DockerClient       *client.Client
	OutputFile         string
	ServiceEntries     []ServiceEntry
	ServiceEntriesPrev []ServiceEntry
	ServiceTemplate    *template.Template
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func NewRouter(outputFile string, templateFile string) *Router {
	var clientOpts []client.Opt

	helper, err := connhelper.GetConnectionHelper(os.Getenv("DOCKER_HOST"))

	if err != nil {
		panic(err)
	}

	if helper != nil {
		httpClient := &http.Client{
			Transport: &http.Transport{
				DialContext: helper.Dialer,
			},
		}

		clientOpts = append(clientOpts,
			client.WithHTTPClient(httpClient),
			client.WithHost(helper.Host),
			client.WithDialContext(helper.Dialer),
		)
	}

	client, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		panic(err)
	}

	return &Router{
		DockerClient:    client,
		OutputFile:      outputFile,
		ServiceTemplate: template.Must(template.ParseFiles(templateFile)),
	}
}

func (s *Router) StartProxyServer() {
	cmd := exec.Command("/usr/sbin/nginx")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	go func() {
		err = cmd.Wait()
		fmt.Printf("Command finished with error: %v", err)
	}()
}

func (s *Router) ReloadProxyServer() bool {
	outputCmd, err := exec.Command("/usr/sbin/nginx", "-s", "reload").Output()
	if err != nil {
		fmt.Printf("Failed to execute command: %s", outputCmd)
		fmt.Println(err)
		return false
	}

	return true
}

func (s *Router) GenerateTemplate() {
	f, err := os.Create(s.OutputFile)
	if err != nil {
		fmt.Println("Create file error: ", err)
		return
	}

	if err := s.ServiceTemplate.Execute(f, s.ServiceEntries); err != nil {
		fmt.Println(err)
	}

	f.Close()
}

func (s *Router) GetServices() {
	services, err := s.DockerClient.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	s.ServiceEntries = make([]ServiceEntry, 0)

	for _, svc := range services {
		servicePath := "/"
		servicePort := "80"
		serviceMaxBodySize := "10m"
		serviceProxyTimeout := "600"

		if val, ok := svc.Spec.Labels["router.path"]; ok {
			servicePath = val
		}

		if val, ok := svc.Spec.Labels["router.port"]; ok {
			servicePort = val
		}

		if val, ok := svc.Spec.Labels["router.max_body_size"]; ok {
			serviceMaxBodySize = val
		}

		if val, ok := svc.Spec.Labels["router.proxy_timeout"]; ok {
			serviceProxyTimeout = val
		}

		domainKeys := make([]string, 0, len(svc.Spec.Labels))

		for key := range svc.Spec.Labels {
			if strings.HasPrefix(key, "router.host") {
				domainKeys = append(domainKeys, key)
			}
		}

		for _, domainKey := range domainKeys {
			if val, ok := svc.Spec.Labels[domainKey]; ok {
				entry := &ServiceEntry{
					ServiceName:         svc.Spec.Name,
					ServiceDomain:       val,
					ServicePath:         servicePath,
					ServicePort:         servicePort,
					ServiceMaxBodySize:  serviceMaxBodySize,
					ServiceProxyTimeout: serviceProxyTimeout,
				}

				s.ServiceEntries = append(s.ServiceEntries, *entry)
			}
		}
	}
}

func (s *Router) IsConfigExists() bool {
	if _, err := os.Stat(s.OutputFile); err != nil {
		return false
	}

	return true
}

func (s *Router) IsReloadRequired() bool {
	count := 0

	for _, current := range s.ServiceEntries {
		for _, prev := range s.ServiceEntriesPrev {
			if cmp.Equal(current, prev) {
				count++
			}
		}
	}

	if count == len(s.ServiceEntries) && len(s.ServiceEntries) == len(s.ServiceEntriesPrev) {
		return false
	}

	return true
}

func (s *Router) UpdatePrevState() {
	s.ServiceEntriesPrev = make([]ServiceEntry, len(s.ServiceEntries))
	copy(s.ServiceEntriesPrev, s.ServiceEntries)
}

func main() {
	templateFile := GetEnv("TEMPLATE_FILE", "router.tpl")
	outputFile := GetEnv("OUTPUT_FILE", "proxy.conf")
	updateInterval, err := strconv.ParseInt(GetEnv("UPDATE_INTERVAL", "1"), 10, 64)
	if err != nil {
		fmt.Println("Wrong UPDATE_INTERVAL value:")
		fmt.Println(err)
	}

	router := NewRouter(outputFile, templateFile)
	router.StartProxyServer()

	for {
		router.GetServices()

		if router.IsReloadRequired() || !router.IsConfigExists() {
			fmt.Println("Configuration updated, reload proxy server...")

			router.GenerateTemplate()
			if router.ReloadProxyServer() {
				fmt.Println("Proxy server reloaded")
				router.UpdatePrevState()
			}
		}

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}
