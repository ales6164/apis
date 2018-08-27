package paypal

import (
	"github.com/gorilla/mux"
	"net/http"
	"golang.org/x/net/context"
	"gopkg.in/ales6164/apis.v2/errors"
	"gopkg.in/ales6164/apis.v2/module"
	"time"
	"google.golang.org/appengine/urlfetch"
	"bytes"
	"encoding/json"
	"google.golang.org/appengine"
	"net/url"
	"strings"
	"strconv"
	"google.golang.org/appengine/log"
)

type paypal interface {
	auth(ctx context.Context) error
	CreatePayment(ctx context.Context, payment *Payment) error
}

type PayPal struct {
	router        *mux.Router
	SandboxMode   bool
	AppHostname   string
	Client        string
	Key           string
	SandboxClient string
	SandboxKey    string
	apiUrl        string
	credentials   *credentials
	modulePath    string
	module.Module
	paypal
	OnPaymentCreated  func(ctx context.Context, paymentId string, payerId string, token string) (redirectUrl string, err error)
	OnPaymentCanceled func(ctx context.Context, token string) (redirectUrl string, err error)
}

type credentials struct {
	Scope       string `json:"scope"`
	Nonce       string `json:"nonce"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	AppID       string `json:"app_id"`
	ExpiresIn   int64  `json:"expires_in"` // seconds
	expiresAt   time.Time                  // added ExpiresIn on time of request
}

func (p *PayPal) Init() error {
	if p.SandboxMode {
		if len(p.SandboxClient) == 0 || len(p.SandboxKey) == 0 {
			return errors.New("SandboxClient and SandboxKey must not be empty")
		}
		p.apiUrl = "https://api.sandbox.paypal.com/v1/"
	} else {
		if len(p.Client) == 0 || len(p.Key) == 0 {
			return errors.New("Client and Key must not be empty")
		}
		p.apiUrl = "https://api.paypal.com/v1/"
	}
	if len(p.AppHostname) == 0 {
		return errors.New("AppHostname must not be empty")
	}
	return nil
}

func (p *PayPal) Name() string {
	return "paypal"
}

func (p *PayPal) ReturnUrl() string {
	return "https://" + p.AppHostname + p.modulePath + "/return"
}

func (p *PayPal) CancelUrl() string {
	return "https://" + p.AppHostname + p.modulePath + "/cancel"
}

func (p *PayPal) Router(modulePath string) *mux.Router {
	p.modulePath = modulePath
	if p.router == nil {
		p.router = mux.NewRouter()
		// add callbacks

		// test order
		/*p.router.HandleFunc(modulePath+"/hi", func(writer http.ResponseWriter, request *http.Request) {
			ctx := appengine.NewContext(request)

			payment, err := p.CreatePayment(ctx, &Payment{
				Intent: "sale",
				Payer:  Payer{PaymentMethod: "paypal"},
				Transactions: []Transaction{
					{
						Amount: Amount{
							Total:    "30.00",
							Currency: "EUR",
							Details: AmountDetails{
								Subtotal: "30.00",
							},
						},
						Description:    "The payment transaction description.",
						Custom:         "ROCKETBOOK_EMS_90048630024435",
						InvoiceNumber:  strconv.Itoa(int(time.Now().Unix())),
						PaymentOptions: PaymentOptions{AllowedPaymentMethod: "INSTANT_FUNDING_SOURCE"},
						SoftDescriptor: "ECHI5786786",
						ItemList: ItemList{
							Items: []Item{
								{
									Name:        "hat",
									Description: "Brown hat.",
									Quantity:    "1",
									Price:       "30.00",
									Sku:         "1",
									Currency:    "EUR",
								},
							},
							ShippingAddress: ShippingAddress{
								RecipientName: "Test Test",
								Line1:         "4thFloot",
								Line2:         "unit#43",
								City:          "San Jose",
								CountryCode:   "US",
								PostalCode:    "95131",
								Phone:         "011862212345678",
								State:         "CA",
							},
						},
					},
				},
				NoteToPayer: "Contact us for any questions on your order.",
				RedirectUrls: RedirectUrls{
					ReturnUrl: "https://" + p.AppHostname + modulePath + "/return",
					CancelUrl: "https://" + p.AppHostname + modulePath + "/cancel",
				},
			})
			if err != nil {
				writer.Write([]byte(err.Error()))
				return
			}
			json.NewEncoder(writer).Encode(payment)
			//writer.Write(payment.Bytes())
		})*/
		p.router.HandleFunc(modulePath+"/return", func(writer http.ResponseWriter, request *http.Request) {
			if p.OnPaymentCreated != nil {
				query := request.URL.Query()
				redirectUrl, err := p.OnPaymentCreated(appengine.NewContext(request), query.Get("paymentId"), query.Get("PayerID"), query.Get("token"))
				if err != nil {
					http.Error(writer, err.Error(), http.StatusInternalServerError)
					return
				}
				if len(redirectUrl) > 0 {
					http.Redirect(writer, request, redirectUrl, http.StatusSeeOther)
				}
			}
		})
		p.router.HandleFunc(modulePath+"/cancel", func(writer http.ResponseWriter, request *http.Request) {
			if p.OnPaymentCreated != nil {
				query := request.URL.Query()
				redirectUrl, err := p.OnPaymentCanceled(appengine.NewContext(request), query.Get("token"))
				if err != nil {
					http.Error(writer, err.Error(), http.StatusInternalServerError)
					return
				}
				if len(redirectUrl) > 0 {
					http.Redirect(writer, request, redirectUrl, http.StatusSeeOther)
				}
			}
		})
	}
	return p.router
}

func (p *PayPal) auth(ctx context.Context) error {
	// authorize if no credentials or if token expires in the next 60 minutes
	// 60 minutes is in the worst case also therefore the longest approved payment can be on hold before executed
	if p.credentials == nil || time.Now().Add(time.Minute * time.Duration(60)).After(p.credentials.expiresAt) {
		// get new credentials

		data := url.Values{}
		data.Set("grant_type", "client_credentials")

		req, _ := http.NewRequest(http.MethodPost, p.apiUrl+"oauth2/token", strings.NewReader(data.Encode()))
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Accept-Language", "en_US")
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
		if p.SandboxMode {
			req.SetBasicAuth(p.SandboxClient, p.SandboxKey)
		} else {
			req.SetBasicAuth(p.Client, p.Key)
		}

		client := urlfetch.Client(ctx)
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			return err
		}
		p.credentials = new(credentials)
		err = json.Unmarshal(buf.Bytes(), p.credentials)
		if err != nil {
			return err
		}
		p.credentials.expiresAt = time.Now().Add(time.Second * time.Duration(p.credentials.ExpiresIn))
	}
	return nil
}

func (p *PayPal) CreatePayment(ctx context.Context, payment *Payment) (*Payment, error) {
	if err := p.auth(ctx); err != nil {
		return nil, err
	}

	postBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest(http.MethodPost, p.apiUrl+"payments/payment", bytes.NewReader(postBytes))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.credentials.AccessToken)

	client := urlfetch.Client(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	responsePayment := new(Payment)
	err = json.Unmarshal(buf.Bytes(), responsePayment)
	return responsePayment, err
}

func (p *PayPal) ExecutePayment(ctx context.Context, paymentId string, payerId string, token string) (*Payment, error) {
	if err := p.auth(ctx); err != nil {
		return nil, err
	}
	body := map[string]string{
		"payer_id": payerId,
	}
	bsBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodPost, p.apiUrl+"payments/payment/"+paymentId+"/execute", bytes.NewReader(bsBody))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+p.credentials.AccessToken)

	client := urlfetch.Client(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Errorf(ctx, "%s", buf.String())
	responsePayment := new(Payment)
	err = json.Unmarshal(buf.Bytes(), responsePayment)
	return responsePayment, err
}
