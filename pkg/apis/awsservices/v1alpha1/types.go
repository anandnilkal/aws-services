/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Stream is a specification for a Stream resource
type Stream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StreamSpec   `json:"spec"`
	Status StreamStatus `json:"status"`
}

// Tag associated with streams
type Tag struct {
	Key   string `json:"tagKey"`
	Value string `json:"tagValue"`
}

// StreamSpec is the spec for a Stream resource
type StreamSpec struct {
	StreamName string `json:"streamName"`
	ShardCount *int32 `json:"shardCount"`
	Tags       []Tag  `json:"tags"`
}

// HashKeyRange range of hash keys supported by shard
type HashKeyRange struct {
	StartingHashKey string `json:"startingHashKey"`
	EndingHashKey   string `json:"endingHashKey"`
}

// SequenceNumberRange used by a shard
type SequenceNumberRange struct {
	StartingSequenceNumber string `json:"startingSequenceNumber"`
	EndingSequenceNumber   string `json:"endingSequenceNumber"`
}

// Shard information of a stream
type Shard struct {
	HashKeyRange          HashKeyRange        `json:"hashKeyRange"`
	SequenceNumberRange   SequenceNumberRange `json:"sequenceNumberRange"`
	ShardID               string              `json:"shardId"`
	AdjacentParentShardID string              `json:"adjacentParentShardId"`
	ParentShardID         string              `json:"parentShardId"`
}

// StreamStatus is the status for a Stream resource
type StreamStatus struct {
	RetentionPeriodHours int32   `json:"retentionPeriodHours"`
	Shards               []Shard `json:"shards"`
	StreamARN            string  `json:"streamARN"`
	StreamName           string  `json:"streamName"`
	StreamStatus         string  `json:"streamStatus"`
	EncryptionType       string  `json:"encryptionType"`
	KeyID                string  `json:"keyId"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StreamList is a list of Stream resources
type StreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Stream `json:"items"`
}
