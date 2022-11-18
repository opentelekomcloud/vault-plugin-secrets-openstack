package common

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/pagination"
)

func listAvailableURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("auth", "domains")
}

func ListAvailable(client *gophercloud.ServiceClient) pagination.Pager {
	url := listAvailableURL(client)
	return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return domains.DomainPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
	})
}
