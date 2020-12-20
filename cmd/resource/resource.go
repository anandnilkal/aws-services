package resource

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kinesisStream "github.com/anandnilkal/aws-services/cmd/kinesis"
	streamResource "github.com/anandnilkal/aws-services/pkg/apis/awsservices/v1alpha1"
	clientset "github.com/anandnilkal/aws-services/pkg/generated/clientset/versioned"
	informers "github.com/anandnilkal/aws-services/pkg/generated/informers/externalversions/awsservices/v1alpha1"
	listers "github.com/anandnilkal/aws-services/pkg/generated/listers/awsservices/v1alpha1"

	"k8s.io/klog/v2"
)

type StreamHandler struct {
	StreamClientSet clientset.Clientset
	StreamClient    *kinesisStream.StreamClient
	StreamInformer  informers.StreamInformer
	StreamList      map[string]*kinesisStream.Stream
	StreamLister    listers.StreamLister
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
	kStream := kinesisStream.NewStream(s.StreamClient.Client, stream.Spec.StreamName, *stream.Spec.ShardCount, s.StreamClient.Region)
	_, err = kStream.CreateStream()
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			klog.Errorf(err.Error())
			return
		}
	}
	s.StreamList[name] = kStream
	streamCopy := stream.DeepCopy()
	streamDescription, err := kStream.DescribeStream()
	streamCopy.Status = streamResource.StreamStatus{
		RetentionPeriodHours: *streamDescription.StreamDescription.RetentionPeriodHours,
		StreamARN:            *streamDescription.StreamDescription.StreamARN,
		StreamStatus:         string(streamDescription.StreamDescription.StreamStatus),
		StreamName:           *streamDescription.StreamDescription.StreamName,
	}
	if streamDescription.StreamDescription.KeyId != nil {
		streamCopy.Status.KeyID = *streamDescription.StreamDescription.KeyId
		streamCopy.Status.EncryptionType = string(streamDescription.StreamDescription.EncryptionType)
	}

	shards := make([]streamResource.Shard, 0)
	for _, shard := range streamDescription.StreamDescription.Shards {
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
		shards = append(shards, shardInfo)
	}
	streamCopy.Status.Shards = shards

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
	var existingStream *kinesisStream.Stream
	var ok bool
	if existingStream, ok = s.StreamList[name]; !ok {
		return
	}
	stream, err := s.StreamLister.Streams(namespace).Get(name)
	if err != nil {
		klog.Errorf(err.Error())
		return
	}
	if *stream.Spec.ShardCount != existingStream.ShardCount {
		existingStream.ShardCount = *stream.Spec.ShardCount
		_, err := existingStream.UpdateStream(existingStream.ShardCount)
		if err != nil {
			klog.Errorf(err.Error())
			return
		}
		streamCopy := stream.DeepCopy()
		streamDescription, err := existingStream.DescribeStream()
		streamCopy.Status = streamResource.StreamStatus{
			RetentionPeriodHours: *streamDescription.StreamDescription.RetentionPeriodHours,
			StreamARN:            *streamDescription.StreamDescription.StreamARN,
			StreamStatus:         string(streamDescription.StreamDescription.StreamStatus),
			StreamName:           *streamDescription.StreamDescription.StreamName,
		}
		if streamDescription.StreamDescription.KeyId != nil {
			streamCopy.Status.KeyID = *streamDescription.StreamDescription.KeyId
			streamCopy.Status.EncryptionType = string(streamDescription.StreamDescription.EncryptionType)
		}

		shards := make([]streamResource.Shard, 0)
		for _, shard := range streamDescription.StreamDescription.Shards {
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
			shards = append(shards, shardInfo)
		}
		streamCopy.Status.Shards = shards

		_, err = s.StreamClientSet.AwsservicesV1alpha1().Streams(stream.Namespace).UpdateStatus(context.TODO(), streamCopy, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf(err.Error())
		}
	}
	return
}

func NewStreamHandler(informer informers.StreamInformer, clientSet clientset.Clientset) *StreamHandler {
	client := kinesisStream.NewStreamClient("us-west-2")
	return &StreamHandler{
		StreamClientSet: clientSet,
		StreamClient:    client,
		StreamList:      make(map[string]*kinesisStream.Stream),
		StreamInformer:  informer,
		StreamLister:    informer.Lister(),
	}
}
