package resources

import "code.cloudfoundry.org/cli/types"

type Metadata struct {
	Labels      map[string]types.NullString `json:"labels,omitempty"`
	Annotations map[string]types.NullString `json:"annotations,omitempty"`
}

type ResourceMetadata struct {
	Metadata *Metadata `json:"metadata,omitempty"`
}
