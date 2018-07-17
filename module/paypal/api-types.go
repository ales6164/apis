package paypal

type Payment struct {
	// RETURN SUCCESS
	Id               string            `json:"id,omitempty"`
	CreateTime       string            `json:"create_time,omitempty"`
	UpdateTime       string            `json:"update_time,omitempty"`
	State            string            `json:"state,omitempty"`
	Links            []Link            `json:"links,omitempty"`
	RelatedResources []RelatedResource `json:"related_resources,omitempty"`

	// SUBMIT
	Intent       string        `json:"intent,omitempty"`
	Payer        Payer         `json:"payer,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	NoteToPayer  string        `json:"note_to_payer,omitempty"`
	RedirectUrls RedirectUrls  `json:"redirect_urls,omitempty"`

	// ERROR
	Error            string         `json:"error,omitempty"`
	ErrorDescription string         `json:"error_description,omitempty"`
	Name             string         `json:"name,omitempty"`
	Details          []ErrorDetails `json:"details,omitempty"`
	Message          string         `json:"message,omitempty"`
	InformationLink  string         `json:"information_link,omitempty"`
	DebugId          string         `json:"debug_id,omitempty"`
}

type Payer struct {
	PaymentMethod string    `json:"payment_method,omitempty"`
	PayerInfo     PayerInfo `json:"payer_info,omitempty"`
}

type PayerInfo struct {
	Email           string          `json:"email,omitempty"`
	FirstName       string          `json:"first_name,omitempty"`
	LastName        string          `json:"last_name,omitempty"`
	PayerId         string          `json:"payer_id,omitempty"`
	ShippingAddress ShippingAddress `json:"shipping_address,omitempty"`
}

type Transaction struct {
	Amount         Amount         `json:"amount,omitempty"`
	Description    string         `json:"description,omitempty"`
	Custom         string         `json:"custom,omitempty"`
	InvoiceNumber  string         `json:"invoice_number,omitempty"`
	PaymentOptions PaymentOptions `json:"payment_options,omitempty"`
	SoftDescriptor string         `json:"soft_descriptor,omitempty"`
	ItemList       ItemList       `json:"item_list,omitempty"`
}

type Amount struct {
	Total    string        `json:"total,omitempty"`    // 30.11
	Currency string        `json:"currency,omitempty"` // USD
	Details  AmountDetails `json:"details,omitempty"`
}

type AmountDetails struct {
	Subtotal         string `json:"subtotal,omitempty"` // 30.00
	Tax              string `json:"tax,omitempty"`
	Shipping         string `json:"shipping,omitempty"`
	HandlingFee      string `json:"handling_fee,omitempty"`
	ShippingDiscount string `json:"shipping_discount,omitempty"`
	Insurance        string `json:"insurance,omitempty"`
}

type PaymentOptions struct {
	AllowedPaymentMethod string `json:"allowed_payment_method,omitempty"`
	RecurringFlag        bool   `json:"recurring_flag,omitempty"`
	SkipFmf              bool   `json:"skip_fmf,omitempty"`
}

type ItemList struct {
	Items           []Item          `json:"items,omitempty"`
	ShippingAddress ShippingAddress `json:"shipping_address,omitempty"`
}

type Item struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Quantity    interface{} `json:"quantity,omitempty"` // input is string, paypal return is usually int
	Price       string      `json:"price,omitempty"`
	Tax         string      `json:"tax,omitempty"`
	Sku         string      `json:"sku,omitempty"`
	Currency    string      `json:"currency,omitempty"`
}

type ShippingAddress struct {
	RecipientName string `json:"recipient_name,omitempty"`
	Line1         string `json:"line1,omitempty"`
	Line2         string `json:"line2,omitempty"`
	City          string `json:"city,omitempty"`
	CountryCode   string `json:"country_code,omitempty"`
	PostalCode    string `json:"postal_code,omitempty"`
	Phone         string `json:"phone,omitempty"`
	State         string `json:"state,omitempty"`
}

type RedirectUrls struct {
	ReturnUrl string `json:"return_url,omitempty"`
	CancelUrl string `json:"cancel_url,omitempty"`
}

type Link struct {
	Href   string `json:"href,omitempty"`
	Rel    string `json:"rel,omitempty"`
	Method string `json:"method,omitempty"`
}

type ErrorDetails struct {
	Field string `json:"field,omitempty"`
	Issue string `json:"issue,omitempty"`
}

type RelatedResource struct {
	Sale Sale `json:"sale,omitempty"`
}

type Sale struct {
	Id                        string         `json:"id,omitempty"`
	CreateTime                string         `json:"create_time,omitempty"`
	UpdateTime                string         `json:"update_time,omitempty"`
	Amount                    Amount         `json:"amount,omitempty"`
	PaymentMode               string         `json:"payment_mode,omitempty"`
	State                     string         `json:"state,omitempty"`
	ProtectionEligibility     string         `json:"protection_eligibility,omitempty"`
	ProtectionEligibilityType string         `json:"protection_eligibility_type,omitempty"`
	TransactionFee            TransactionFee `json:"transaction_fee,omitempty"`
	ParentPayment             string         `json:"parent_payment,omitempty"`
	Links                     []Link         `json:"links,omitempty"`
}

type TransactionFee struct {
	Value    string `json:"value,omitempty"`
	Currency string `json:"currency,omitempty"`
}
