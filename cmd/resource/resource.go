package resource

import (
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kinesisStream "github.com/anandnilkal/aws-services/cmd/kinesis"
	streamResource "github.com/anandnilkal/aws-services/pkg/apis/awsservices/v1alpha1"
	clientset "github.com/anandnilkal/aws-services/pkg/generated/clientset/versioned"
	informers "github.com/anandnilkal/aws-services/pkg/generated/informers/externalversions/awsservices/v1alpha1"
	listers "github.com/anandnilkal/aws-services/pkg/generated/listers/awsservices/v1alpha1"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"

	"k8s.io/klog/v2"
)

type StreamHandler struct {
	StreamClientSet clientset.Clientset
	StreamClient    *kinesisStream.StreamClient
	StreamInformer  informers.StreamInformer
	StreamList      map[string]*kinesisStream.Stream
	StreamLister    listers.StreamLister
	Ticker          *time.Ticker
	Done            chan bool
}

func (s *StreamHandler) AddFunc(name, namespace string) {
	klog.V(4).Infof("Added Stream: %s, Namespace: %s", name, namespace)
	if _, ok := s.StreamList[name]; ok {
		return
	}
	stream, err := s.StreamLister.Streams(namespace).Get(name)
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	kStream := kinesisStream.NewStream(s.StreamClient.Client, stream.Spec.StreamName, stream.ObjectMeta.Namespace, *stream.Spec.ShardCount)
	_, err = kStream.CreateStream()
	if err != nil {
		currentErr := err
		for errors.Unwrap(currentErr) != nil {
			currentErr = errors.Unwrap(currentErr)
		}
		// if _, ok := currentErr.(*types.LimitExceededException); ok {
		// 	time.Sleep(time.Second * 1)
		// 	kStream.CreateStream()
		// } else {
		// 	if _, ok := currentErr.(*types.ResourceInUseException); !ok {
		klog.Errorf(err.Error())
		streamCopy := stream.DeepCopy()
		streamCopy.Status = streamResource.StreamStatus{
			RetryCount:           0,
			Error:                err.Error(),
			Status:               "FAILED",
			RetentionPeriodHours: 0,
			Shards:               []streamResource.Shard{},
			StreamARN:            "",
			StreamName:           name,
			StreamStatus:         "",
			EncryptionType:       "",
			KeyID:                "",
		}
		_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf(err.Error())
		}
		return
		// }
		// }
	}
	tags := s.getTags(stream.Spec.Tags)
	if len(tags) > 0 {
		_, err = kStream.TagStream(tags)
		if err != nil {
			klog.Errorf(err.Error())
			return
		}
	}
	s.StreamList[name] = kStream
	streamCopy := stream.DeepCopy()
	streamDescription, err := kStream.DescribeStream()
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	streamStatus := s.getStreamStatus(streamDescription, "SUCCESS", "", 0)
	streamCopy.Status = *streamStatus

	_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf(err.Error())
	}
	return
}

func (s *StreamHandler) DeleteFunc(name, namespace string) {
	klog.V(4).Infof("Deleted Stream: %s, Namespace: %s", name, namespace)
	var existingStream *kinesisStream.Stream
	var ok bool
	if existingStream, ok = s.StreamList[name]; !ok {
		return
	}
	consumersList, err := existingStream.ListStreamConsumers()
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	for _, consumer := range consumersList.Consumers {
		_, err := existingStream.DeregisterStreamConsumers(*consumer.ConsumerARN)
		if err != nil {
			klog.Errorf(err.Error())
		}
	}
	_, err = existingStream.DeleteStream()
	if err != nil {
		klog.Errorf(err.Error())
	}
	delete(s.StreamList, name)
	return
}

func (s *StreamHandler) UpdateFunc(name, namespace string) {
	klog.V(4).Infof("Updated Stream: %s, Namespace: %s", name, namespace)
	stream, err := s.StreamLister.Streams(namespace).Get(name)
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	if stream.Status.Status == "FAILED" {
		if stream.Status.RetryCount < 5 {
			kStream := kinesisStream.NewStream(s.StreamClient.Client, stream.Spec.StreamName, stream.ObjectMeta.Namespace, *stream.Spec.ShardCount)
			_, err = kStream.CreateStream()
			if err != nil {
				streamCopy := stream.DeepCopy()
				streamCopy.Status = streamResource.StreamStatus{
					RetryCount:           stream.Status.RetryCount + 1,
					Error:                err.Error(),
					Status:               "FAILED",
					RetentionPeriodHours: 0,
					Shards:               []streamResource.Shard{},
					StreamARN:            "",
					StreamName:           name,
					StreamStatus:         "",
					EncryptionType:       "",
					KeyID:                "",
				}
				_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf(err.Error())
				}
				return
			}
			s.StreamList[name] = kStream
		} else {
			return
		}
	}
	var existingStream *kinesisStream.Stream
	var ok bool
	if existingStream, ok = s.StreamList[name]; !ok {
		return
	}
	if *stream.Spec.ShardCount != existingStream.ShardCount {
		existingStream.ShardCount = *stream.Spec.ShardCount
		_, err := existingStream.UpdateStream(existingStream.ShardCount)
		if err != nil {
			klog.Errorf(err.Error())
			return
		}
	}
	streamCopy := stream.DeepCopy()
	streamDescription, err := existingStream.DescribeStream()
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	tags := s.getTags(stream.Spec.Tags)
	if len(tags) > 0 {
		_, err = existingStream.TagStream(tags)
		if err != nil {
			klog.Errorf(err.Error())
			return
		}
	}
	streamStatus := s.getStreamStatus(streamDescription, "SUCCESS", "", stream.Status.RetryCount)
	streamCopy.Status = *streamStatus
	_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf(err.Error())
	}
	return
}

func (s *StreamHandler) getStreamStatus(description *kinesis.DescribeStreamOutput, streamStatus, err string, retryCount int32) *streamResource.StreamStatus {
	status := streamResource.StreamStatus{
		Status:               streamStatus,
		Error:                err,
		RetryCount:           retryCount,
		RetentionPeriodHours: *description.StreamDescription.RetentionPeriodHours,
		StreamARN:            *description.StreamDescription.StreamARN,
		StreamStatus:         string(description.StreamDescription.StreamStatus),
		StreamName:           *description.StreamDescription.StreamName,
	}
	if description.StreamDescription.KeyId != nil {
		status.KeyID = *description.StreamDescription.KeyId
		status.EncryptionType = string(description.StreamDescription.EncryptionType)
	}

	status.Shards = make([]streamResource.Shard, 0)
	for _, shard := range description.StreamDescription.Shards {
		shardInfo := streamResource.Shard{}
		if shard.ShardId != nil {
			shardInfo.ShardID = *shard.ShardId
		}
		if shard.AdjacentParentShardId != nil {
			shardInfo.AdjacentParentShardID = *shard.AdjacentParentShardId
		}
		if shard.ParentShardId != nil {
			shardInfo.ParentShardID = *shard.ParentShardId
		}
		if shard.HashKeyRange != nil {
			if shard.HashKeyRange.StartingHashKey != nil {
				shardInfo.HashKeyRange.StartingHashKey = *shard.HashKeyRange.StartingHashKey
			}
			if shard.HashKeyRange.EndingHashKey != nil {
				shardInfo.HashKeyRange.EndingHashKey = *shard.HashKeyRange.EndingHashKey
			}
		}
		if shard.SequenceNumberRange != nil {
			if shard.SequenceNumberRange.StartingSequenceNumber != nil {
				shardInfo.SequenceNumberRange.StartingSequenceNumber = *shard.SequenceNumberRange.StartingSequenceNumber
			}
			if shard.SequenceNumberRange.EndingSequenceNumber != nil {
				shardInfo.SequenceNumberRange.EndingSequenceNumber = *shard.SequenceNumberRange.EndingSequenceNumber
			}
		}
		status.Shards = append(status.Shards, shardInfo)
	}

	return &status
}

func (s *StreamHandler) getTags(tags []streamResource.Tag) map[string]string {
	streamTags := make(map[string]string)
	for _, tag := range tags {
		streamTags[tag.Key] = tag.Value
	}
	return streamTags
}

func NewStreamHandler(informer informers.StreamInformer, clientSet clientset.Clientset, region string, duration int64) *StreamHandler {
	client := kinesisStream.NewStreamClient(region)
	ticker := time.NewTicker(time.Duration(duration * int64(time.Second)))
	done := make(chan bool)
	streamHandler := StreamHandler{
		StreamClientSet: clientSet,
		StreamClient:    client,
		StreamList:      make(map[string]*kinesisStream.Stream),
		StreamInformer:  informer,
		StreamLister:    informer.Lister(),
		Ticker:          ticker,
		Done:            done,
	}
	go streamHandler.periodicStatusFetch()
	return &streamHandler
}

func (s *StreamHandler) periodicStatusFetch() {
	for {
		select {
		case <-s.Done:
			return
		case <-s.Ticker.C:
			for _, stream := range s.StreamList {
				streamDescription, err := stream.DescribeStream()
				if err != nil {
					klog.Errorf(err.Error())
					continue
				}
				stream, err := s.StreamLister.Streams(stream.Namespace).Get(stream.StreamName)
				if err != nil {
					klog.Errorf(err.Error())
					continue
				}
				streamStatus := s.getStreamStatus(streamDescription, "SUCCESS", "", stream.Status.RetryCount)
				streamCopy := stream.DeepCopy()
				streamCopy.Status = *streamStatus
				_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
				if err != nil {
					klog.Errorf(err.Error())
				}
			}
		}
	}
}
