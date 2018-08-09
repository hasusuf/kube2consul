package cmd

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/golang/glog"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/hasusuf/kube2consul/templates"
	"github.com/hasusuf/kube2consul/util"
	"github.com/hasusuf/kube2consul/util/i18n"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	// Only required to authenticate against GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	noFlags		= true
	k8s			*kubernetes.Clientset
	consul  	*consulAPI.Client
	syncLong 	= templates.LongDesc(i18n.T(`
		Mirror your Kubernetes secrets/configMaps to Consul`))
)

// NewCmdSync is a parent command to multiple nested sub-commands
func NewCmdSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: i18n.T("Mirror your Kubernetes secrets/configMaps to Consul"),
		Long:  syncLong,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				noFlags = true
				cmd.Help()
				os.Exit(0)
			} else {
				noFlags = false
			}

			syncKube2Consul(cmd)
		},
	}

	addKube2ConsulFlags(cmd)

	return cmd
}

func addKube2ConsulFlags(cmd *cobra.Command) {
	cmd.Flags().String(
		"context",
		"",
		"Kubernetes context")

	cmd.Flags().String(
		"namespace",
		"",
		"Kubernetes namespace")

	cmd.Flags().String(
		"connection-type",
		"internal",
		"Kubernetes cluster connection type")

	cmd.Flags().String(
		"config-path",
		getLocalConfig(),
		"Kubernetes config path")

	cmd.Flags().String(
		"consul-uri",
		"",
		"Consul endpoint")
}


func setSyncKube2ConsulRequiredFlags(cmd *cobra.Command) error {
	if ! noFlags {
		connectionType := util.GetFlagString(cmd, "connection-type")

		if connectionType == "external" {
			cmd.MarkFlagRequired("context")
		}
	}

	cmd.MarkFlagRequired("consul-uri")
	cmd.MarkFlagRequired("namespace")

	return nil
}

func syncKube2Consul(cmd *cobra.Command) {
	consul, k8s = setUpConnection(cmd)
	namespace := util.GetFlagString(cmd, "namespace")
	deleteAll()
	syncSecrets(namespace)
	syncConfigs(namespace)
}



// Start of Kubernetes methods
func syncSecrets(namespace string) {
	secrets, _ := k8s.CoreV1().Secrets(namespace).List(metav1.ListOptions{})

	for _, secret := range secrets.Items {

		if secret.Type == "Opaque" {
			if strEndWith(secret.Name, "secret", "-") {
				if debugFlag {
					fmt.Println()
				}
				fmt.Printf("%v has been successfuly synced!\n", secret.Name)
				if debugFlag {
					fmt.Println("========================================")
				}

				resourceName := getResourceName(secret.Name)

				for k, v := range secret.Data {
					printToConsole(k, string(v))
					syncToConsul(resourceName, k, string(v))
				}

				fetchRecordsFromConsul(resourceName)
			}
		}
	}
}

func syncConfigs(namespace string) {
	configs, _ := k8s.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{})

	for _, config := range configs.Items {

		if strEndWith(config.Name, "config", "-") {
			if debugFlag {
				fmt.Println()
			}
			fmt.Printf("%v has been successfuly synced!\n", config.Name)
			if debugFlag {
				fmt.Println("========================================")
			}

			resourceName := getResourceName(config.Name)

			for k, v := range config.Data {
				printToConsole(k, v)
				syncToConsul(getResourceName(config.Name), k, v)
			}

			fetchRecordsFromConsul(resourceName)
		}
	}
}

func getResourceName(haystack string) string {
	arr := strings.Split(haystack, "-")
	idx := len(arr) - 1
	slice := append(arr[:idx], arr[idx+1:]...)

	return strings.Join(slice, "-")
}

func strEndWith(haystack string, needle string, delimiter string) bool {
	arr := strings.Split(haystack, delimiter)
	x := arr[len(arr)-1]

	return x == needle
}

func printToConsole(k string, v string) {
	if debugFlag {
		fmt.Printf("Key: %v, Value: %v \n", k, v)
	}
}

func setKubeClient(cmd *cobra.Command) *kubernetes.Clientset {
	config, err := setKubeConfig(cmd)
	errorHandler(err)

	client, err := kubernetes.NewForConfig(config)
	errorHandler(err)

	return client
}

func setKubeConfig(cmd *cobra.Command) (*rest.Config, error) {
	connectionType := util.GetFlagString(cmd, "connection-type")
	context := util.GetFlagString(cmd, "context")
	configPath := util.GetFlagString(cmd, "config-path")

	if !util.IsEmpty(context) {
		connectionType = "external"
	}
	if connectionType == "external" {
		return buildConfigFromFlags(context, configPath)
	}

	return rest.InClusterConfig()
}

func buildConfigFromFlags(context, kubeConfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

func getLocalConfig() string {
	var homeDir string

	u, err := user.Current()
	if err != nil {
		for _, name := range []string{"HOME", "USERPROFILE"} { // *nix, windows
			if dir := os.Getenv(name); dir != "" {
				homeDir = dir
				break
			}
		}
	} else {
		homeDir = u.HomeDir
	}

	return fmt.Sprintf("%s/.kube/config", homeDir)
}

// End of Kubernetes methods



// Start of Consul methods
func deleteAll() {
	var success bool

	pairs, _, err := consul.KV().List("/", nil)
	errorHandler(err)

	for i := range pairs {
		pair := pairs[i]

		ok, _, err := consul.KV().DeleteCAS(pair, nil)
		if err != nil {
			glog.Fatalf("Unable to delete keys %v\n", err)
		}

		success = ok
	}

	if success {
		fmt.Println("Existing records has been successfully deleted!")
	}
}

func syncToConsul(name string, key string, value string) {
	id, _ := createSession()
	createConsulPair(id, fmt.Sprintf("%s/env/%s", name, key), value)
}

func fetchRecordsFromConsul(name string) {
	if debugFlag {
		fmt.Println()
		fmt.Printf("%v consul records \n", name)
		fmt.Println("========================================")
		pairs, _, err := consul.KV().List(name, nil)
		errorHandler(err)

		for i := range pairs {
			pair := pairs[i]
			fmt.Printf("Key: %s Value: %s\n", pair.Key, pair.Value)
		}
	}
}

func setConsulClient(addr string) *consulAPI.Client {
	config := consulAPI.DefaultConfig()
	config.Address = addr

	client, err := consulAPI.NewClient(config)
	errorHandler(err)

	return client
}

func createSession() (string, *consulAPI.WriteMeta) {
	session := consul.Session()
	id, wm, err := session.Create(&consulAPI.SessionEntry{
		Behavior: consulAPI.SessionBehaviorRelease,
		TTL:      "20s",
	}, nil)

	if err != nil {
		glog.Fatalf("failed to create session: %v", err)
	}

	return id, wm
}

func createConsulPair(id string, key string, value string) {
	success, _, err := consul.KV().Acquire(&consulAPI.KVPair{
		Key:     key,
		Value:   []byte(value),
		Session: id,
	}, nil)
	if err != nil {
		glog.Fatalf("failed to create KV pair: %v", err)
	}
	if !success {
		glog.Fatalf("failed to acquire lock")
	}
}

// Setup client connections
func setUpConnection(cmd *cobra.Command) (*consulAPI.Client, *kubernetes.Clientset) {
	consulURI := util.GetFlagString(cmd, "consul-uri")
	return setConsulClient(consulURI), setKubeClient(cmd)
}