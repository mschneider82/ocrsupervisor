package main

import "time"

/*
{
    "EventName": "s3:ObjectCreated:Put",
    "Key": "ftpserver/CCE29012021.pdf",
    "Records": [
        {
            "eventVersion": "2.0",
            "eventSource": "minio:s3",
            "awsRegion": "",
            "eventTime": "2021-05-14T09:53:18.659Z",
            "eventName": "s3:ObjectCreated:Put",
            "userIdentity": {
                "principalId": "x"
            },
            "requestParameters": {
                "principalId": "x",
                "region": "",
                "sourceIPAddress": "10.42.3.0"
            },
            "responseElements": {
                "content-length": "0",
                "x-amz-request-id": "167EE62D33977CB0",
                "x-minio-deployment-id": "f419d854-d8c2-4ba7-999c-91584753b8c4",
                "x-minio-origin-endpoint": "http://10.42.3.47:9000"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "Config",
                "bucket": {
                    "name": "ftpserver",
                    "ownerIdentity": {
                        "principalId": "x"
                    },
                    "arn": "arn:aws:s3:::ftpserver"
                },
                "object": {
                    "key": "CCE29012021.pdf",
                    "size": 515320,
                    "eTag": "b713d89260bb82d400710bc141e9bfd7",
                    "contentType": "application/pdf",
                    "userMetadata": {
                        "content-type": "application/pdf"
                    },
                    "sequencer": "167EE62D6C014045"
                }
            },
            "source": {
                "host": "10.42.3.0",
                "port": "",
                "userAgent": "aws-sdk-go/1.37.19 (go1.16; linux; amd64) S3Manager"
            }
        }
    ]
}
*/

type Event struct {
	EventName string    `json:"EventName"`
	Key       string    `json:"Key"`
	Records   []Records `json:"Records"`
}
type UserIdentity struct {
	PrincipalID string `json:"principalId"`
}
type RequestParameters struct {
	PrincipalID     string `json:"principalId"`
	Region          string `json:"region"`
	SourceIPAddress string `json:"sourceIPAddress"`
}
type ResponseElements struct {
	ContentLength        string `json:"content-length"`
	XAmzRequestID        string `json:"x-amz-request-id"`
	XMinioDeploymentID   string `json:"x-minio-deployment-id"`
	XMinioOriginEndpoint string `json:"x-minio-origin-endpoint"`
}
type OwnerIdentity struct {
	PrincipalID string `json:"principalId"`
}
type Bucket struct {
	Name          string        `json:"name"`
	OwnerIdentity OwnerIdentity `json:"ownerIdentity"`
	Arn           string        `json:"arn"`
}
type UserMetadata struct {
	ContentType string `json:"content-type"`
}
type Object struct {
	Key          string       `json:"key"`
	Size         int          `json:"size"`
	ETag         string       `json:"eTag"`
	ContentType  string       `json:"contentType"`
	UserMetadata UserMetadata `json:"userMetadata"`
	Sequencer    string       `json:"sequencer"`
}
type S3 struct {
	S3SchemaVersion string `json:"s3SchemaVersion"`
	ConfigurationID string `json:"configurationId"`
	Bucket          Bucket `json:"bucket"`
	Object          Object `json:"object"`
}
type Source struct {
	Host      string `json:"host"`
	Port      string `json:"port"`
	UserAgent string `json:"userAgent"`
}
type Records struct {
	EventVersion      string            `json:"eventVersion"`
	EventSource       string            `json:"eventSource"`
	AwsRegion         string            `json:"awsRegion"`
	EventTime         time.Time         `json:"eventTime"`
	EventName         string            `json:"eventName"`
	UserIdentity      UserIdentity      `json:"userIdentity"`
	RequestParameters RequestParameters `json:"requestParameters"`
	ResponseElements  ResponseElements  `json:"responseElements"`
	S3                S3                `json:"s3"`
	Source            Source            `json:"source"`
}
