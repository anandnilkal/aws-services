package kinesis

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"k8s.io/klog/v2"
)

type StreamClient struct {
	Client *kinesis.Client
	Region string
}

type Stream struct {
	Client     *kinesis.Client
	StreamName string
	ShardCount int32
	StreamARN  string
	Namespace  string
}

func NewStreamClient(region string) *StreamClient {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		log.Fatalf("failed to load SDK configuration, %v", err)
	}
	return &StreamClient{
		Client: kinesis.NewFromConfig(cfg),
		Region: region,
	}
}

func NewStream(client *kinesis.Client, name, namespace string, shard int32) *Stream {
	return &Stream{
		Client:     client,
		StreamName: name,
		ShardCount: shard,
		Namespace:  namespace,
	}
}

func (s *Stream) CreateStream() (*kinesis.CreateStreamOutput, error) {
	streamOut, err := s.Client.CreateStream(context.Background(), &kinesis.CreateStreamInput{
		ShardCount: &s.ShardCount,
		StreamName: &s.StreamName,
	})
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}
	return streamOut, nil
}

func (s *Stream) DescribeStream() (*kinesis.DescribeStreamOutput, error) {
	describeOut, err := s.Client.DescribeStream(context.Background(), &kinesis.DescribeStreamInput{
		StreamName: &s.StreamName,
	})
	if describeOut.StreamDescription.StreamARN != nil {
		s.StreamARN = *describeOut.StreamDescription.StreamARN
	}

	return describeOut, err
}

func (s *Stream) DescribeStreamSummary() (*kinesis.DescribeStreamSummaryOutput, error) {
	return s.Client.DescribeStreamSummary(context.Background(), &kinesis.DescribeStreamSummaryInput{
		StreamName: &s.StreamName,
	})
}

func (s *Stream) DeleteStream() (*kinesis.DeleteStreamOutput, error) {
	return s.Client.DeleteStream(context.Background(), &kinesis.DeleteStreamInput{
		StreamName: &s.StreamName,
	})
}

func (s *Stream) ListStreamConsumers() (*kinesis.ListStreamConsumersOutput, error) {
	return s.Client.ListStreamConsumers(context.Background(), &kinesis.ListStreamConsumersInput{
		StreamARN: &s.StreamARN,
	})
}

func (s *Stream) DeregisterStreamConsumers(consumerARN string) (*kinesis.DeregisterStreamConsumerOutput, error) {
	return s.Client.DeregisterStreamConsumer(context.Background(), &kinesis.DeregisterStreamConsumerInput{
		StreamARN:   &s.StreamARN,
		ConsumerARN: &consumerARN,
	})
}

func (s *Stream) UpdateStream(shardCount int32) (*kinesis.UpdateShardCountOutput, error) {
	return s.Client.UpdateShardCount(context.Background(), &kinesis.UpdateShardCountInput{
		ScalingType:      types.ScalingTypeUniformScaling,
		StreamName:       &s.StreamName,
		TargetShardCount: &shardCount,
	})
}

func (s *Stream) TagStream(tags map[string]string) (*kinesis.AddTagsToStreamOutput, error) {
	return s.Client.AddTagsToStream(context.Background(), &kinesis.AddTagsToStreamInput{
		StreamName: &s.StreamName,
		Tags:       tags,
	})
}
