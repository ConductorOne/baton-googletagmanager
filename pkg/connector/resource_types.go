package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

var (
	// The user resource type is for all user objects from the database.
	userResourceType = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_USER},
		Annotations: annotationsForUserResourceType(),
	}

	// The account resource type is for all account objects from the database.
	accountResourceType = &v2.ResourceType{
		Id:          "account",
		DisplayName: "Account",
	}

	// The container resource type is for all container objects from the database.
	containerResourceType = &v2.ResourceType{
		Id:          "container",
		DisplayName: "Container",
	}

	// The role resource type is for all possible roles (either account or container).
	roleResourceType = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}
)
