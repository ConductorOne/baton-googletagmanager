package connector

import (
	"context"
	"fmt"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"google.golang.org/api/tagmanager/v2"
)

const (
	RolePermissionUnspecifiedRole      = "accountPermissionUnspecified"
	AdminRole                          = "admin"
	NoAccessRole                       = "noAccess"
	UserRole                           = "user"
	ApproveRole                        = "approve"
	ContainerPermissionUnspecifiedRole = "containerPermissionUnspecified"
	EditRole                           = "edit"
	PublishRole                        = "publish"
	ReadRole                           = "read"
)

var (
	accountPermissions = []string{
		RolePermissionUnspecifiedRole,
		AdminRole,
		NoAccessRole,
		UserRole,
	}
	containerPermissions = []string{
		ApproveRole,
		ContainerPermissionUnspecifiedRole,
		EditRole,
		PublishRole,
		ReadRole,
	}
)

type roleBuilder struct {
	client       *tagmanager.Service
	resourceType *v2.ResourceType
}

func (r *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return roleResourceType
}

func roleResource(ctx context.Context, role string, parent *v2.ResourceId) (*v2.Resource, error) {
	roleID := fmt.Sprintf("%s:%s", parent.Resource, role)
	resource, err := rs.NewRoleResource(
		role,
		roleResourceType,
		roleID,
		nil,
		rs.WithParentResourceID(parent),
	)

	if err != nil {
		return nil, err
	}

	return resource, nil
}

// List returns all the roles from the database as resource objects.
func (r *roleBuilder) List(ctx context.Context, parentResource *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentResource == nil {
		return nil, "", nil, nil
	}

	var rv []*v2.Resource

	if parentResource.ResourceType == accountResourceType.Id {
		for _, p := range accountPermissions {
			ar, err := roleResource(ctx, p, parentResource)
			if err != nil {
				return nil, "", nil, err
			}

			rv = append(rv, ar)
		}
	}

	if parentResource.ResourceType == containerResourceType.Id {
		for _, p := range containerPermissions {
			cr, err := roleResource(ctx, p, parentResource)
			if err != nil {
				return nil, "", nil, err
			}

			rv = append(rv, cr)
		}
	}

	return rv, "", nil, nil
}

// TODO: Implement and comment this method.
func (r *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// TODO: Implement and comment this method.
func (r *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newRoleBuilder(client *tagmanager.Service) *roleBuilder {
	return &roleBuilder{
		client:       client,
		resourceType: roleResourceType,
	}
}
