package resource

import (
	"k8s.io/klog/v2"
)

type stream struct {
	StreamName string `json:"streamName"`
	ShardCount string `json:"shardCount"`
}

type StreamHandler struct {
	StreamList []stream `json:"stream_list"`
}

func (s *StreamHandler) AddFunc(name, namespace string) {
	klog.V(4).Infof("Added Stream: %s, Namespace: %s", name, namespace)
	return
}

func (s *StreamHandler) DeleteFunc(name, namespace string) {
	klog.V(4).Infof("Deleted Stream: %s, Namespace: %s", name, namespace)
	return
}

func (s *StreamHandler) UpdateFunc(name, namespace string) {
	klog.V(4).Infof("Updated Stream: %s, Namespace: %s", name, namespace)
	return
}

func NewStreamHandler() *StreamHandler {
	return &StreamHandler{
		StreamList: make([]stream, 0),
	}
}
