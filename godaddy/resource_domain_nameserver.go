package godaddy

import (
	"fmt"
	"log"
	"strconv"

	"github.com/andrewstucki/terraform-provider-godaddy/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type domainNameserversResource struct {
	Customer    string
	Domain      string
	NameServers []string
}

func (d *domainNameserversResource) ToAPI() *api.DomainPurchase {
	return &api.DomainPurchase{
		NameServers: d.NameServers,
	}
}

func newDomainNameserversResource(d *schema.ResourceData) (*domainNameserversResource, error) {
	var err error
	r := &domainNameserversResource{}

	if attr, ok := d.GetOk(attrCustomer); ok {
		r.Customer = attr.(string)
	}
	if attr, ok := d.GetOk(attrDomain); ok {
		r.Domain = attr.(string)
	}
	if attr, ok := d.GetOk(attrNameservers); ok {
		records := attr.([]interface{})
		r.NameServers = make([]string, len(records))
		for i, rec := range records {
			if err = api.ValidateData(api.NSType, rec.(string)); err != nil {
				return nil, err
			}
			r.NameServers[i] = rec.(string)
		}
	}
	return r, nil
}

func resourceDomainNameservers() *schema.Resource {
	return &schema.Resource{
		Create: resourceDomainNameserversUpdate,
		Read:   resourceDomainNameserversRead,
		Update: resourceDomainNameserversUpdate,
		Delete: resourceDomainNameserversReset,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			// Required
			attrDomain: {
				Type:     schema.TypeString,
				Required: true,
			},
			attrNameservers: {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			// Optional
			attrCustomer: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceDomainNameserversRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	customer := d.Get(attrCustomer).(string)
	domain := d.Get(attrDomain).(string)

	// Importer support
	if domain == "" {
		domain = d.Id()
	}

	return populateNameserverInfo(client, customer, domain, d)
}

func resourceDomainNameserversUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	r, err := newDomainNameserversResource(d)
	if err != nil {
		return err
	}

	if err = populateNameserverInfo(client, r.Customer, r.Domain, d); err != nil {
		return err
	}

	log.Println("setting nameservers", r.Domain)
	return client.UpdateDomain(r.Customer, r.Domain, r.ToAPI())
}

func resourceDomainNameserversReset(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	r, err := newDomainNameserversResource(d)
	if err != nil {
		return err
	}

	if err = populateNameserverInfo(client, r.Customer, r.Domain, d); err != nil {
		return err
	}

	r.NameServers = []string{
		"ns53.domaincontrol.com",
		"ns54.domaincontrol.com",
	}

	log.Println("resetting nameservers", r.Domain)
	return client.UpdateDomain(r.Customer, r.Domain, r.ToAPI())
}

func populateNameserverInfo(client *api.Client, cust, dom string, d *schema.ResourceData) error {
	var err error
	var domain *api.Domain

	domain, err = client.GetDomain(cust, dom)
	if err != nil {
		return fmt.Errorf("couldn't find domain (%s): %s", dom, err.Error())
	}

	d.SetId(strconv.FormatInt(domain.ID, 10))
	return nil
}
