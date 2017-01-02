package main

import (
	"log"
	"strings"
	"testing"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
)

func TestDeletePod(t *testing.T) {
	client := fake.NewSimpleClientset()

	_, err := client.Core().Pods("default").Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "default",
			Name:      "bar",
		},
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = client.Core().Pods("foo").Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "foo",
			Name:      "bar",
		},
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	_, err = client.Core().Pods("baz").Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: "baz",
			Name:      "bar",
		},
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	// should be noop
	err = deletePod("", client)
	if err != nil {
		log.Fatal(err.Error())
	}

	// should be noop
	err = deletePod("kux/bar", client)
	if err != nil {
		log.Fatal(err.Error())
	}

	// should be noop
	err = deletePod("kux/bar/foo", client)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = deletePod("foo/bar", client)
	if err != nil {
		log.Fatal(err.Error())
	}

	pods, err := client.Core().Pods("foo").List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(pods.Items) != 0 {
		t.Errorf("pod not killed")
	}

	pods, err = client.Core().Pods("default").List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(pods.Items) != 1 {
		t.Errorf("pod was killed")
	}

	pods, err = client.Core().Pods("baz").List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(pods.Items) != 1 {
		t.Errorf("pod was killed")
	}

	err = deletePod("bar", client)
	if err != nil {
		log.Fatal(err.Error())
	}

	pods, err = client.Core().Pods("default").List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(pods.Items) != 0 {
		t.Errorf("pod not killed")
	}

	pods, err = client.Core().Pods("baz").List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}

	if len(pods.Items) != 1 {
		t.Errorf("pod was killed")
	}

	// // pods, err := client.Core().Pods("foo").Get("bar")
	// pods, err := client.Core().Pods("foo").List(v1.ListOptions{})
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }
	//
	// for _, pod := range pods.Items {
	// 	fmt.Println(pod.Name)
	// }
}

func deletePod(podID string, client clientset.Interface) error {
	parts := strings.Split(podID, "/")

	if len(parts) == 1 {
		parts = []string{"default", parts[0]}
	}

	err := client.Core().Pods(parts[0]).Delete(parts[1], &v1.DeleteOptions{})
	// err := client.Core().Pods("foo").Delete("bar", &v1.DeleteOptions{})
	// err := client.Core().Pods("foo").DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
	// err := client.Core().Pods(v1.NamespaceAll).DeleteCollection(&v1.DeleteOptions{}, v1.ListOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}

	return nil
}
