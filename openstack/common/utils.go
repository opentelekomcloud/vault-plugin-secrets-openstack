package common

import (
	"github.com/gophercloud/gophercloud/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/roles"
)

func CheckGroupSlices(groups []groups.Group, userEntities []string) []string {
	var existingEntity []string
	for _, entity := range groups {
		existingEntity = append(existingEntity, entity.Name)
	}
	return sliceSubtraction(userEntities, existingEntity)
}

func CheckRolesSlices(roles []roles.Role, userEntities []string) []string {
	var existingEntity []string
	for _, entity := range roles {
		existingEntity = append(existingEntity, entity.Name)
	}
	return sliceSubtraction(userEntities, existingEntity)
}

func sliceSubtraction(a, b []string) (diff []string) {
	m := make(map[string]bool)

	for _, item := range b {
		m[item] = true
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}
	return
}
