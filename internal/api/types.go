package api

import (
	"errors"
	"fmt"
	"strings"
)

// RecordType is an enumeration of possible DNS record types
type RecordType int

// RecordFactory is a factory method for creating new DomainRecords
type RecordFactory func(string) (*DomainRecord, error)

const (
	// A is an address record type
	A RecordType = iota
	// AAAA is an IPv6 address record type
	AAAA
	// CAA is a Certificate Authority record type
	CAA
	// CNAME is a Canonical record name (alias) type
	CNAME
	// MX is a mail exchange record type
	MX
	// NS is a name server record type
	NS
	// SOA is a start of authority record type
	SOA
	// SRV is a service locator type
	SRV
	// TXT is a text record type
	TXT
)

func (rt RecordType) String() string {
	switch rt {
	case A:
		return AType
	case AAAA:
		return AAAAType
	case CAA:
		return CAAType
	case CNAME:
		return CNameType
	case MX:
		return MXType
	case NS:
		return NSType
	case SOA:
		return SOAType
	case SRV:
		return SRVType
	case TXT:
		return TXTType
	}
	return ""
}

const (
	DefaultTTL      = 3600
	DefaultPriority = 0
	DefaultWeight   = 0
	DefaultPort     = 0

	StatusActive    = "ACTIVE"
	StatusCancelled = "CANCELLED"

	Ptr       = "@"
	AType     = "A"
	AAAAType  = "AAAA"
	CAAType   = "CAA"
	CNameType = "CNAME"
	MXType    = "MX"
	NSType    = "NS"
	SOAType   = "SOA"
	SRVType   = "SRV"
	TXTType   = "TXT"
)

var supportedTypes = []string{
	AType, AAAAType, CAAType, CNameType, MXType, NSType, SOAType, SRVType, TXTType,
}

// Address is the structure used for mailing addresses
type Address struct {
	Line1      string `json:"address1,omitempty"`
	Line2      string `json:"address2,omitempty"`
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	State      string `json:"state,omitempty"`
}

// Consent is the structure containing consent info
type Consent struct {
	AgreedAt      string   `json:"agreedAt"`
	AgreedBy      string   `json:"agreedBy,omitempty"`
	AgreementKeys []string `json:"agreementKeys,omitempty"`
}

// Contact is the structure used to hold contact info
type Contact struct {
	Address      *Address `json:"addressMailing,omitempty"`
	Email        string   `json:"email,omitempty"`
	Fax          string   `json:"fax,omitempty"`
	JobTitle     string   `json:"jobTitle,omitempty"`
	FirstName    string   `json:"nameFirst,omitempty"`
	LastName     string   `json:"nameLast,omitempty"`
	MiddleName   string   `json:"nameMiddle,omitempty"`
	Organization string   `json:"organization,omitempty"`
	Phone        string   `json:"phone,omitempty"`
}

// DomainPurchase is the structure used for purchasing a domain
type DomainPurchase struct {
	Consent           *Consent `json:"consent,omitempty"`
	AdminContact      *Contact `json:"contactAdmin,omitempty"`
	BillingContact    *Contact `json:"contactBilling,omitempty"`
	RegistrantContact *Contact `json:"contactRegistrant,omitempty"`
	TechContact       *Contact `json:"contactTech,omitempty"`
	Domain            string   `json:"domain,omitempty"`
	NameServers       []string `json:"nameServers"`
	YearsLeased       int      `json:"period,omitempty"`
	EnablePrivacy     bool     `json:"privacy,omitempty"`
	AutoRenew         bool     `json:"renewAuto,omitempty"`
}

// DomainPurchaseReceipt is the receipt of a purchase
type DomainPurchaseReceipt struct {
	Currency string `json:"currency"`
	Count    int    `json:"itemCount"`
	OrderID  int    `json:"orderId"`
	Total    int    `json:"total"`
}

// DomainPurchaseOpt provides support for setting optional parameters
type DomainPurchaseOpt func(*DomainPurchase) error

// NewDomainPurchase validates and constructs a DomainPurchase, if valid.
func NewDomainPurchase(name string, opts ...DomainPurchaseOpt) (*DomainPurchase, error) {
	name = strings.TrimSpace(name)
	dp := &DomainPurchase{
		Domain: name,
	}
	for _, opt := range opts {
		if err := opt(dp); err != nil {
			return nil, err
		}
	}
	return dp, nil
}

// PurchaseConsent sets the purchase consent info
func PurchaseConsent(consent Consent) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.Consent = &consent
		return nil
	}
}

// Admin sets the purchase admin info
func Admin(contact Contact) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.AdminContact = &contact
		return nil
	}
}

// Billing sets the purchase billing info
func Billing(contact Contact) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.BillingContact = &contact
		return nil
	}
}

// Registrant sets the purchase registrant info
func Registrant(contact Contact) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.RegistrantContact = &contact
		return nil
	}
}

// Tech sets the purchase tech info
func Tech(contact Contact) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.TechContact = &contact
		return nil
	}
}

// Nameservers sets the purchase ns info
func Nameservers(ns []string) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.NameServers = ns
		return nil
	}
}

// YearsLeased sets the purchase period info
func YearsLeased(years int) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.YearsLeased = years
		return nil
	}
}

// EnablePrivacy sets the purchase privacy info
func EnablePrivacy(privacy bool) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.EnablePrivacy = privacy
		return nil
	}
}

// AutoRenew sets the purchase renewal info
func AutoRenew(renew bool) DomainPurchaseOpt {
	return func(rec *DomainPurchase) error {
		rec.AutoRenew = renew
		return nil
	}
}

// Domain encapsulates a domain resource
type Domain struct {
	ID                int64    `json:"domainId"`
	Name              string   `json:"domain"`
	Status            string   `json:"status"`
	AdminContact      *Contact `json:"contactAdmin,omitempty"`
	BillingContact    *Contact `json:"contactBilling,omitempty"`
	RegistrantContact *Contact `json:"contactRegistrant,omitempty"`
	TechContact       *Contact `json:"contactTech,omitempty"`
	NameServers       []string `json:"nameservers,omitempty"`
	YearsLeased       int      `json:"period,omitempty"`
	EnablePrivacy     bool     `json:"privacy,omitempty"`
	AutoRenew         bool     `json:"renewAuto,omitempty"`
}

// DomainRecord encapsulates a domain record resource
type DomainRecord struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority"`
	TTL      int    `json:"ttl"`
	Service  string `json:"service,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Weight   int    `json:"weight"`
	Port     *int   `json:"port,omitempty"`
}

// DomainRecordOpt provides support for setting optional parameters
type DomainRecordOpt func(*DomainRecord) error

// NewDomainRecord validates and constructs a DomainRecord, if valid.
func NewDomainRecord(name, t, data string, ttl int, opts ...DomainRecordOpt) (*DomainRecord, error) {
	name = strings.TrimSpace(name)
	data = strings.TrimSpace(data)
	if err := ValidateData(t, data); err != nil {
		return nil, err
	}

	parts := strings.Split(name, ".")
	if len(parts) < 1 || len(parts) > 255 {
		return nil, errors.New("name must be between 1..255 octets")
	}
	for _, part := range parts {
		if len(part) > 63 {
			return nil, errors.New("invalid domain name. name octets should be less than 63 characters")
		}
	}

	if ttl < 0 {
		return nil, errors.New("ttl must be a positive value")
	}
	if !isSupportedType(t) {
		return nil, fmt.Errorf("type must be one of: %s", supportedTypes)
	}
	dr := &DomainRecord{
		Name: name,
		Type: t,
		Data: data,
		TTL:  ttl,
	}
	for _, opt := range opts {
		if err := opt(dr); err != nil {
			return nil, err
		}
	}
	return dr, nil
}

func Priority(priority int) DomainRecordOpt {
	return func(rec *DomainRecord) error {
		if err := ValidatePriority(priority); err != nil {
			return err
		}
		rec.Priority = priority
		return nil
	}
}

func Weight(weight int) DomainRecordOpt {
	return func(rec *DomainRecord) error {
		if err := ValidateWeight(weight); err != nil {
			return err
		}
		rec.Weight = weight
		return nil
	}
}

func Port(port int) DomainRecordOpt {
	return func(rec *DomainRecord) error {
		if port == 0 {
			return nil
		}
		if err := ValidatePort(port); err != nil {
			return err
		}
		rec.Port = &port
		return nil
	}
}

func Service(service string) DomainRecordOpt {
	return func(rec *DomainRecord) error {
		if strings.TrimSpace(service) != "" && !strings.HasPrefix(service, "_") {
			return errors.New("service must start with an underscore (e.g. _ldap)")
		}
		rec.Service = service
		return nil
	}
}

func Protocol(proto string) DomainRecordOpt {
	return func(rec *DomainRecord) error {
		if strings.TrimSpace(proto) != "" && !strings.HasPrefix(proto, "_") {
			return errors.New("protocol must start with an underscore (e.g. _tcp)")
		}
		rec.Protocol = proto
		return nil
	}
}

// NewNSRecord constructs a nameserver record from the supplied data
func NewNSRecord(data string) (*DomainRecord, error) {
	return NewDomainRecord(Ptr, NSType, data, DefaultTTL)
}

// NewARecord constructs a new address record from the supplied data
func NewARecord(data string) (*DomainRecord, error) {
	return NewDomainRecord(Ptr, AType, data, DefaultTTL)
}

// ValidateData performs bounds checking on a data element
func ValidateData(t, data string) error {
	switch t {
	case SRVType:
		return nil
	case TXTType:
		if len(data) < 0 || len(data) > 512 {
			return errors.New("TXT data must be between 0..512 characters in length")
		}
	default:
		if len(data) < 0 || len(data) > 255 {
			return errors.New("data must be between 0..255 characters in length")
		}
	}
	return nil
}

// ValidatePriority performs bounds checking on priority element
func ValidatePriority(priority int) error {
	if priority < 0 || priority > 65535 {
		return errors.New("priority must be between 0..65535 (16 bit)")
	}
	return nil
}

func ValidateWeight(weight int) error {
	if weight < 0 || weight > 100 {
		return errors.New("weight must be between 0..100")
	}
	return nil
}

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1..65535")
	}
	return nil
}

// IsDefaultARecord is a predicate to place fetched A domain records into the appropriate bucket
func IsDefaultARecord(record *DomainRecord) bool {
	return record.Name == Ptr && record.Type == AType && record.TTL == DefaultTTL
}

// IsDefaultNSRecord is a predicate to place fetched NS domain records into the appropriate bucket
func IsDefaultNSRecord(record *DomainRecord) bool {
	return record.Name == Ptr && record.Type == NSType && record.TTL == DefaultTTL
}

// IsDisallowed prevents empty NS|SOA record lists from being propagated, which is disallowed
func IsDisallowed(t string, records []*DomainRecord) bool {
	return len(records) == 0 && strings.EqualFold(t, NSType) || strings.EqualFold(t, SOAType) || strings.EqualFold(t, CAAType)
}

func isSupportedType(recType string) bool {
	for _, t := range supportedTypes {
		if t == recType {
			return true
		}
	}
	return false
}
