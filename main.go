package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nsqio/go-nsq"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	endpoint         = kingpin.Flag("endpoint", "s3 endpoint").Required().String()
	accesskey        = kingpin.Flag("accesskey", "s3 accesskey").Required().String()
	secretkey        = kingpin.Flag("secret", "s3 secret").Required().String()
	seafileserver    = kingpin.Flag("seafileserver", "url to seafile server").Required().String()
	seafiletoken     = kingpin.Flag("seafiletoken", "token see https://download.seafile.com/published/web-api/home.md").Required().String()
	seafilelibraryid = kingpin.Flag("seafilelibraryid", "e.g. 3e040126-4533-4d0c-97f3-baa284915515").Required().String()

	topic            = kingpin.Flag("topic", "nsq topic name [Env: NSQ_TOPIC]").Default("minio").OverrideDefaultFromEnvar("NSQ_TOPIC").String()
	channel          = kingpin.Flag("channel", "nsq channelname [Env: NSQ_CHANNEL]").Default("ocrsuper").OverrideDefaultFromEnvar("NSQ_CHANNEL").String()
	maxInFlight      = kingpin.Flag("max-in-flight", "max number of messages to allow in flight [Env: NSQ_MAXINFLIGHT]").Default("200").OverrideDefaultFromEnvar("NSQ_MAXINFLIGHT").Int()
	lookupdHTTPAddrs = kingpin.Flag("lookupdHTTPAddrs", "lookupdHTTPAddrs [Env: NSQ_LOOKUPD]").Default("localhost:4161").OverrideDefaultFromEnvar("NSQ_LOOKUPD").Strings()
	kubeconfig       = kingpin.Flag("kubeconfig", "kubeconfig file  HOME/.kube/config").Required().String()
)

type handler struct {
	kube *kubernetes.Clientset
}

func (h *handler) HandleMessage(message *nsq.Message) error {
	var e Event
	err := json.Unmarshal(message.Body, &e)
	if err != nil {
		return err
	}

	if e.EventName != "s3:ObjectCreated:Put" {
		return nil
	}

	file := filepath.Base(e.Key)
	bucket := filepath.Dir(e.Key)

	pod := getPodObject(file, bucket)

	pod, err = h.kube.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Pod created successfully...")

	return nil
}

func getPodObject(filename, bucket string) *core.Pod {
	return &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ocrfile",
			Namespace: "ocr",
			Labels: map[string]string{
				"app": "s3ocr",
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            "s3ocr",
					Image:           "mschneider82/s3ocr",
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"--endpoint=" + *endpoint,
						"--accesskey=" + *accesskey,
						"--secret=" + *secretkey,
						"--useSSL",
						"--bucket=" + bucket,
						"--object=" + filename,
						"--seafileserver=" + *seafileserver,
						"--seafiletoken=" + *seafiletoken,
						"--seafilelibraryid=" + *seafilelibraryid,
					},
				},
			},
		},
	}
}

func main() {
	// build configuration from the config file.
	config, err := rest.InClusterConfig()
	//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	// create kubernetes clientset. this clientset can be used to create,delete,patch,list etc for the kubernetes resources
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	cfg := nsq.NewConfig()

	consumer, err := nsq.NewConsumer(*topic, *channel, cfg)
	if err != nil {
		log.Fatalln(err)
	}

	h := &handler{kube: clientset}
	consumer.AddHandler(h)

	err = consumer.ConnectToNSQLookupds(*lookupdHTTPAddrs)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Awaiting messages from NSQ topic ", *topic)
	select {
	case <-termChan:
		consumer.Stop()
		<-consumer.StopChan
		log.Printf("Exiting.")
	}
}
