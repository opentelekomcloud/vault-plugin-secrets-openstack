package common

import (
	"fmt"
	golangsdk "github.com/gophercloud/gophercloud"
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

func LogHttpError(err error) error {
	switch httpErr := err.(type) {
	case golangsdk.ErrDefault400:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault401:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault403:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault404:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault405:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault408:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault409:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault429:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault500:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault502:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault503:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrDefault504:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	case golangsdk.ErrUnexpectedResponseCode:
		return fmt.Errorf("%s\n %s", httpErr.Error(), httpErr.Body)
	}
	return err
}
