package godaddy

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/andrewstucki/terraform-provider-godaddy/api"
)

const (
	attrAutoRenew         = "auto_renew"
	attrEnablePrivacy     = "enable_privacy"
	attrYearsLeased       = "years_leased"
	attrAdminContact      = "admin"
	attrBillingContact    = "billing"
	attrRegistrantContact = "registrant"
	attrTechContact       = "tech"

	contactAttrAddress = "address"

	contactEmail        = "email"
	contactFax          = "fax"
	contactJobTitle     = "job_title"
	contactFirstName    = "first_name"
	contactLastName     = "last_name"
	contactMiddleName   = "middle_name"
	contactOrganization = "organization"
	contactPhone        = "phone"

	addressLine1      = "line_1"
	addressLine2      = "line_2"
	addressCity       = "city"
	addressCountry    = "country"
	addressPostalCode = "postal_code"
	addressState      = "state"
)

type domainPurchaseResource struct {
	Customer      string
	Domain        string
	NameServers   []string
	YearsLeased   int
	EnablePrivacy bool
	AutoRenew     bool
	Consent       *api.Consent
	Admin         *api.Contact
	Billing       *api.Contact
	Registrant    *api.Contact
	Tech          *api.Contact
}

func (d *domainPurchaseResource) ToAPI() *api.DomainPurchase {
	return &api.DomainPurchase{
		Domain:            d.Domain,
		NameServers:       d.NameServers,
		YearsLeased:       d.YearsLeased,
		EnablePrivacy:     d.EnablePrivacy,
		AutoRenew:         d.AutoRenew,
		Consent:           d.Consent,
		AdminContact:      d.Admin,
		BillingContact:    d.Billing,
		RegistrantContact: d.Registrant,
		TechContact:       d.Tech,
	}
}

func getContact(field string, d *schema.ResourceData) (*api.Contact, error) {
	if attr, ok := d.GetOk(field); ok {
		records := attr.(*schema.Set).List()
		if len(records) > 1 {
			return nil, fmt.Errorf("must specify only one '%s' block", field)
		}
		data := records[0].(map[string]interface{})
		contact := &api.Contact{}
		if v, ok := data[contactEmail].(string); ok {
			contact.Email = v
		}
		if v, ok := data[contactFax].(string); ok {
			contact.Fax = v
		}
		if v, ok := data[contactJobTitle].(string); ok {
			contact.JobTitle = v
		}
		if v, ok := data[contactFirstName].(string); ok {
			contact.FirstName = v
		}
		if v, ok := data[contactLastName].(string); ok {
			contact.LastName = v
		}
		if v, ok := data[contactMiddleName].(string); ok {
			contact.MiddleName = v
		}
		if v, ok := data[contactOrganization].(string); ok {
			contact.Organization = v
		}
		if v, ok := data[contactPhone].(string); ok {
			contact.Phone = v
		}

		if attr, ok := data[contactAttrAddress]; ok {
			records := attr.(*schema.Set).List()
			if len(records) > 1 {
				return nil, fmt.Errorf("must specify only one '%s' block", contactAttrAddress)
			}
			data := records[0].(map[string]interface{})

			address := &api.Address{}
			if v, ok := data[addressLine1].(string); ok {
				address.Line1 = v
			}
			if v, ok := data[addressLine2].(string); ok {
				address.Line2 = v
			}
			if v, ok := data[addressCity].(string); ok {
				address.City = v
			}
			if v, ok := data[addressCountry].(string); ok {
				address.Country = v
			}
			if v, ok := data[addressPostalCode].(string); ok {
				address.PostalCode = v
			}
			if v, ok := data[addressState].(string); ok {
				address.State = v
			}
			contact.Address = address
		}
		return contact, nil
	}
	return nil, nil
}

func newDomainPurchaseResource(d *schema.ResourceData) (*domainPurchaseResource, error) {
	var err error
	r := &domainPurchaseResource{}

	if attr, ok := d.GetOk(attrCustomer); ok {
		r.Customer = attr.(string)
	}
	if attr, ok := d.GetOk(attrDomain); ok {
		r.Domain = attr.(string)
	}
	if attr, ok := d.GetOk(attrYearsLeased); ok {
		r.YearsLeased = attr.(int)
	}
	if attr, ok := d.GetOk(attrEnablePrivacy); ok {
		r.AutoRenew = attr.(bool)
	}
	if attr, ok := d.GetOk(attrAutoRenew); ok {
		r.AutoRenew = attr.(bool)
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

	admin, err := getContact(attrAdminContact, d)
	if err != nil {
		return nil, err
	}
	r.Admin = admin

	tech, err := getContact(attrTechContact, d)
	if err != nil {
		return nil, err
	}
	r.Tech = tech

	billing, err := getContact(attrBillingContact, d)
	if err != nil {
		return nil, err
	}
	r.Billing = billing

	registrant, err := getContact(attrRegistrantContact, d)
	if err != nil {
		return nil, err
	}
	r.Registrant = registrant

	r.Consent = &api.Consent{
		AgreementKeys: []string{
			"DNRA",
		},
		AgreedAt: time.Now().UTC().Format(time.RFC3339),
		AgreedBy: "127.0.0.1",
	}

	return r, nil
}

func resourceDomainPurchase() *schema.Resource {
	addressSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			addressLine1: {
				Type:     schema.TypeString,
				Optional: true,
			},
			addressLine2: {
				Type:     schema.TypeString,
				Optional: true,
			},
			addressCity: {
				Type:     schema.TypeString,
				Optional: true,
			},
			addressCountry: {
				Type:     schema.TypeString,
				Optional: true,
			},
			addressPostalCode: {
				Type:     schema.TypeString,
				Optional: true,
			},
			addressState: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

	contactSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			contactAttrAddress: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     addressSchema,
			},
			contactEmail: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactFax: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactJobTitle: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactFirstName: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactLastName: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactMiddleName: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactOrganization: {
				Type:     schema.TypeString,
				Optional: true,
			},
			contactPhone: {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

	return &schema.Resource{
		Create: resourceDomainPurchaseCreate,
		Read:   resourceDomainPurchaseRead,
		Update: resourceDomainPurchaseUpdate,
		Delete: resourceDomainPurchaseCancel,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			// Required
			attrDomain: {
				Type:     schema.TypeString,
				Required: true,
			},
			attrYearsLeased: {
				Type:     schema.TypeInt,
				Required: true,
			},
			// Optional
			attrEnablePrivacy: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			attrAutoRenew: {
				Type:     schema.TypeBool,
				Optional: true,
			},
			attrCustomer: {
				Type:     schema.TypeString,
				Optional: true,
			},
			attrNameservers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			attrAdminContact: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     contactSchema,
			},
			attrBillingContact: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     contactSchema,
			},
			attrRegistrantContact: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     contactSchema,
			},
			attrTechContact: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     contactSchema,
			},
		},
	}
}

func resourceDomainPurchaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	customer := d.Get(attrCustomer).(string)
	domain := d.Get(attrDomain).(string)

	// Importer support
	if domain == "" {
		domain = d.Id()
	}

	log.Println("Fetching", domain, "records...")
	record, err := client.GetDomain(customer, domain)
	if err != nil {
		return fmt.Errorf("couldn't find domain record (%s): %s", domain, err.Error())
	}
	return populatePurchaseDataFromResponse(record, d)
}

func resourceDomainPurchaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	r, err := newDomainPurchaseResource(d)
	if err != nil {
		return err
	}

	log.Println("Purchasing", r.Domain)
	_, err = client.PurchaseDomain(r.Customer, r.ToAPI())
	if err != nil {
		return err
	}

	return populateDomainInfo(client, r.Customer, r.Domain, d)
}

func resourceDomainPurchaseUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	r, err := newDomainPurchaseResource(d)
	if err != nil {
		return err
	}

	if err = populateDomainInfo(client, r.Customer, r.Domain, d); err != nil {
		return err
	}

	log.Println("Updating", r.Domain)
	return client.UpdateDomain(r.Customer, r.Domain, r.ToAPI())
}

func resourceDomainPurchaseCancel(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	r, err := newDomainPurchaseResource(d)
	if err != nil {
		return err
	}

	if err = populateDomainInfo(client, r.Customer, r.Domain, d); err != nil {
		return err
	}

	log.Println("Canceling domain", r.Domain)
	return client.CancelDomain(r.Customer, r.Domain)
}

func populatePurchaseDataFromResponse(domain *api.Domain, d *schema.ResourceData) error {
	if err := d.Set(attrNameservers, domain.NameServers); err != nil {
		return err
	}
	if err := d.Set(attrAutoRenew, domain.AutoRenew); err != nil {
		return err
	}
	if err := d.Set(attrEnablePrivacy, domain.EnablePrivacy); err != nil {
		return err
	}
	if err := d.Set(attrYearsLeased, domain.YearsLeased); err != nil {
		return err
	}
	if err := d.Set(attrAdminContact, contactMap(domain.AdminContact)); err != nil {
		return err
	}
	if err := d.Set(attrBillingContact, contactMap(domain.BillingContact)); err != nil {
		return err
	}
	if err := d.Set(attrRegistrantContact, contactMap(domain.RegistrantContact)); err != nil {
		return err
	}
	if err := d.Set(attrTechContact, contactMap(domain.TechContact)); err != nil {
		return err
	}

	return nil
}

func contactMap(contact *api.Contact) []map[string]interface{} {
	if contact == nil {
		return nil
	}
	return []map[string]interface{}{
		map[string]interface{}{
			contactAttrAddress:  addressMap(contact.Address),
			contactEmail:        contact.Email,
			contactFax:          contact.Fax,
			contactJobTitle:     contact.JobTitle,
			contactFirstName:    contact.FirstName,
			contactLastName:     contact.LastName,
			contactMiddleName:   contact.MiddleName,
			contactOrganization: contact.Organization,
			contactPhone:        contact.Phone,
		},
	}
}

func addressMap(address *api.Address) []map[string]interface{} {
	if address == nil {
		return nil
	}
	return []map[string]interface{}{
		map[string]interface{}{
			addressLine1:      address.Line1,
			addressLine2:      address.Line2,
			addressCity:       address.City,
			addressCountry:    address.Country,
			addressPostalCode: address.PostalCode,
			addressState:      address.State,
		},
	}
}
