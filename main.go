package main

import (
	"math/rand"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	appName = "chaoskube"
	image   = "quay.io/linki/chaoskube"
	version = "v0.3.1"
)

var (
	kubeconfig string
	interval   time.Duration
	inCluster  bool
	deploy     bool
	dryRun     bool
	debug      bool
)

func init() {
	kingpin.Flag("kubeconfig", "Path to a kubeconfig file").Default(clientcmd.RecommendedHomeFile).StringVar(&kubeconfig)
	kingpin.Flag("interval", "Interval between Pod terminations").Short('i').Default("10m").DurationVar(&interval)
	kingpin.Flag("in-cluster", "If true, finds the Kubernetes cluster from the environment").Short('c').BoolVar(&inCluster)
	kingpin.Flag("deploy", "If true, deploys chaoskube in the current cluster with the provided configuration").Short('d').BoolVar(&deploy)
	kingpin.Flag("dry-run", "If true, don't actually do anything.").Default("true").BoolVar(&dryRun)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&debug)

	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if dryRun {
		log.Infof("Dry run enabled. I won't kill anything. Use --no-dry-run when you're ready.")
	}

	client, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	if deploy {
		log.Debugf("Deploying %s:%s", image, version)

		manifest := generateManifest()

		deployment := client.Extensions().Deployments(manifest.Namespace)

		_, err := deployment.Get(manifest.Name)
		if err != nil {
			_, err = deployment.Create(manifest)
		} else {
			_, err = deployment.Update(manifest)
		}
		if err != nil {
			log.Fatal(err)
		}

		log.Infof("Deployed %s:%s", image, version)
		os.Exit(0)
	}

	for {
		pods, err := client.Core().Pods(v1.NamespaceAll).List(v1.ListOptions{})
		if err != nil {
			log.Fatal(err)
		}

		victim := pods.Items[rand.Intn(len(pods.Items))]

		log.Infof("Killing pod %s/%s", victim.Namespace, victim.Name)

		if !dryRun {
			err = client.Core().Pods(victim.Namespace).Delete(victim.Name, nil)
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Debugf("Sleeping for %s...", interval)
		time.Sleep(interval)
	}
}

func newClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		config, err = rest.InClusterConfig()
		log.Infof("Using in-cluster config.")
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		log.Infof("Using current context from kubeconfig at %s.", kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	log.Debugf("Targeting cluster at %s", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func generateManifest() *v1beta1.Deployment {
	// modifies flags for deployment
	args := append(os.Args[1:], "--in-cluster")
	args = stripFlags(args, "--kubeconfig")
	args = stripFlags(args, "--deploy")

	return &v1beta1.Deployment{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: v1.NamespaceDefault,
			Labels: map[string]string{
				"app":      appName,
				"heritage": appName,
			},
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"app": appName,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  appName,
							Image: image + ":" + version,
							Args:  args,
						},
					},
				},
			},
		},
	}
}

func stripFlags(elements []string, candidate string) []string {
	for i := range elements {
		if strings.Contains(elements[i], candidate) {
			elements = append(elements[:i], elements[i+1:]...)
			break
		}
	}

	return elements
}
