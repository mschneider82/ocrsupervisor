package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nsqio/go-nsq"
	"github.com/rs/zerolog/log"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	endpoint         = kingpin.Flag("endpoint", "s3 endpoint").Required().String()
	accesskey        = kingpin.Flag("accesskey", "s3 accesskey").Required().String()
	secretkey        = kingpin.Flag("secret", "s3 secret").OverrideDefaultFromEnvar("S3_SECRET").Required().String()
	seafileserver    = kingpin.Flag("seafileserver", "url to seafile server").Required().String()
	seafiletoken     = kingpin.Flag("seafiletoken", "token see https://download.seafile.com/published/web-api/home.md").OverrideDefaultFromEnvar("SEAFILE_TOKEN").Required().String()
	seafilelibraryid = kingpin.Flag("seafilelibraryid", "e.g. 3e040126-4533-4d0c-97f3-baa284915515").Required().String()

	topic            = kingpin.Flag("topic", "nsq topic name [Env: NSQ_TOPIC]").Default("minio").OverrideDefaultFromEnvar("NSQ_TOPIC").String()
	channel          = kingpin.Flag("channel", "nsq channelname [Env: NSQ_CHANNEL]").Default("ocrsuper").OverrideDefaultFromEnvar("NSQ_CHANNEL").String()
	maxInFlight      = kingpin.Flag("max-in-flight", "max number of messages to allow in flight [Env: NSQ_MAXINFLIGHT]").Default("200").OverrideDefaultFromEnvar("NSQ_MAXINFLIGHT").Int()
	lookupdHTTPAddrs = kingpin.Flag("lookupdHTTPAddrs", "lookupdHTTPAddrs [Env: NSQ_LOOKUPD]").Default("localhost:4161").OverrideDefaultFromEnvar("NSQ_LOOKUPD").Strings()
)

type handler struct {
	kube *kubernetes.Clientset
}

func (h *handler) HandleMessage(message *nsq.Message) error {
	var e Event
	err := json.Unmarshal(message.Body, &e)
	if err != nil {
		log.Info().Msgf("event is not a s3 bucket notification msg from minio, skipping nsq message: %s", err.Error())
		return err
	}

	if e.EventName != "s3:ObjectCreated:Put" {
		log.Info().Msg("event is not a s3 bucket notification msg from minio, skipping nsq message")
		return nil
	}

	file := filepath.Base(e.Key)
	bucket := filepath.Dir(e.Key)

	jobSpec := getJobObject(file, bucket)
	jobs := h.kube.BatchV1().Jobs("ocr")

	_, err = jobs.Create(context.TODO(), jobSpec, metav1.CreateOptions{})
	if err != nil {
		log.Err(err).Msg("Failed to create K8s job")
	}

	log.Info().Msg("Pod created successfully...")

	return nil
}

func getJobObject(filename, bucket string) *batchv1.Job {
	var backOffLimit int32 = 0
	var ttl int32 = 120

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			//	Name:      "ocrjob",
			GenerateName: "ocr",
			Namespace:    "ocr",
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:            "s3ocr",
							Image:           "mschneider82/s3ocr",
							ImagePullPolicy: core.PullIfNotPresent,
							Command:         []string{"/home/docker/s3ocr"},
							Args: []string{
								"--endpoint=" + *endpoint,
								"--accesskey=" + *accesskey,
								//	"--secret=" + *secretkey,
								"--useSSL",
								"--bucket=" + bucket,
								"--object=" + filename,
								"--seafileserver=" + *seafileserver,
								//	"--seafiletoken=" + *seafiletoken,
								"--seafilelibraryid=" + *seafilelibraryid,
							},
							Env: []core.EnvVar{
								{
									Name: "S3_SECRET",
									ValueFrom: &core.EnvVarSource{
										SecretKeyRef: &core.SecretKeySelector{
											LocalObjectReference: core.LocalObjectReference{Name: "s3secret"},
											Key:                  "s3secret",
										},
									},
								},
								{
									Name: "SEAFILE_TOKEN",
									ValueFrom: &core.EnvVarSource{
										SecretKeyRef: &core.SecretKeySelector{
											LocalObjectReference: core.LocalObjectReference{Name: "seafiletoken"},
											Key:                  "seafiletoken",
										},
									},
								},
							},
						},
					},
					RestartPolicy: core.RestartPolicyNever,
				},
			},
			BackoffLimit: &backOffLimit,
		},
	}
}

func main() {
	kingpin.Parse()
	// build configuration from the config file.
	config, err := rest.InClusterConfig()
	//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal().Err(err).Msg("rest InClusterConfig")
	}
	// create kubernetes clientset. this clientset can be used to create,delete,patch,list etc for the kubernetes resources
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msg("create kubernetes clientset")
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	cfg := nsq.NewConfig()

	log.Info().Msgf("Create Nsq consumer for topic: %s chan: %s\n", *topic, *channel)
	consumer, err := nsq.NewConsumer(*topic, *channel, cfg)
	if err != nil {
		log.Fatal().Err(err).Str("topic", *topic).
			Str("channel", *channel).
			Msg("create nsq consumer")
	}

	h := &handler{kube: clientset}
	consumer.AddHandler(h)

	err = consumer.ConnectToNSQLookupds(*lookupdHTTPAddrs)
	if err != nil {
		log.Fatal().Err(err).Str("topic", *topic).
			Str("channel", *channel).
			Strs("nsqlookupds", *lookupdHTTPAddrs).
			Msg("connectToNSQLookupds")
	}

	log.Info().Msgf("Awaiting messages from NSQ topic %s", *topic)
	select {
	case <-termChan:
		consumer.Stop()
		<-consumer.StopChan
		log.Info().Msg("Exiting.")
	}
}

/*
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
					Command:         []string{"/home/docker/s3ocr"},
					Args: []string{
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
*/
