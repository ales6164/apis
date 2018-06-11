package paypal

type Payment struct {
	Id         string `json:"id,omitempty"`
	CreateTime string `json:"create_time,omitempty"`
	UpdateTime string `json:"update_time,omitempty"`
	State      string `json:"state,omitempty"`
	Links      []Link `json:"links,omitempty"`

	Intent       string        `json:"intent,omitempty"`
	Payer        Payer         `json:"payer,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	NoteToPayer  string        `json:"note_to_payer,omitempty"`
	RedirectUrls RedirectUrls  `json:"redirect_urls,omitempty"`
}

type Payer struct {
	PaymentMethod string `json:"payment_method,omitempty"`
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
	Total    string        `json:"total,omitempty"` // 30.11
	Currency string        `json:"total,omitempty"` // USD
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
}

type ItemList struct {
	Items           []Item          `json:"items,omitempty"`
	ShippingAddress ShippingAddress `json:"shipping_address,omitempty"`
}

type Item struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	Price       string `json:"price,omitempty"`
	Tax         string `json:"tax,omitempty"`
	Sku         string `json:"sku,omitempty"`
	Currency    string `json:"currency,omitempty"`
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
