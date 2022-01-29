package tags

import (
	"github.com/crra/id3v2/v2"
)

type tagResolver struct {
	errTagNonStandard error
	knownTags         map[string]string
}

func NewV24(errTagNonStandard error) *tagResolver {
	knownTags := make(map[string]string, len(id3v2.V24CommonIDs))
	for description, tagName := range id3v2.V24CommonIDs {
		knownTags[tagName] = description
	}

	return &tagResolver{
		errTagNonStandard: errTagNonStandard,
		knownTags:         knownTags,
	}
}

func (r *tagResolver) DescriptionFor(id string) (string, error) {
	description, exist := r.knownTags[id]
	if !exist {
		return "", r.errTagNonStandard
	}

	return description, nil
}
