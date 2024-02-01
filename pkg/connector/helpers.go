package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

func annotationsForUserResourceType() annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.SkipEntitlementsAndGrants{})
	return annos
}

func parsePageToken(i string, resourceID *v2.ResourceId) (*pagination.Bag, string, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(i)
	if err != nil {
		return nil, "", err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: resourceID.ResourceType,
			ResourceID:     resourceID.Resource,
		})
	}

	return b, b.PageToken(), nil
}

// func prepareParentID(pID *v2.ResourceId) (string, error) {
// 	switch pID.ResourceType {
// 	case accountResourceType.Id:
// 		return fmt.Sprintf("accounts/%s", pID), nil
// 	case containerResourceType.Id:
// 		return fmt.Sprintf("containers/%s", pID), nil
// 	default:
// 		return "", fmt.Errorf("invalid parent type: %s", pID.ResourceType)
// 	}
// }
