package paypal

import (
	"github.com/gorilla/mux"
	"net/http"
	"golang.org/x/net/context"
	"github.com/ales6164/apis/errors"
	"github.com/ales6164/apis/module"
	"time"
	"google.golang.org/appengine/urlfetch"
	"bytes"
	"encoding/json"
	"google.golang.org/appengine"
	"net/url"
	"strings"
	"strconv"
	"io"
)

type paypal interface {
	auth(ctx context.Context) error
	CreatePayment(ctx context.Context, payment *Payment) error
}

type PayPal struct {
	router        *mux.Router
	SandboxMode   bool
	Client        string
	Key           string
	SandboxClient string
	SandboxKey    string
	apiUrl        string
	credentials   *credentials
	module.Module
	paypal
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
	return nil
}

func (p *PayPal) Name() string {
	return "paypal"
}

func (p *PayPal) Router(modulePath string) *mux.Router {
	if p.router == nil {
		p.router = mux.NewRouter()
		// add callbacks
		p.router.HandleFunc(modulePath+"/hi", func(writer http.ResponseWriter, request *http.Request) {
			ctx := appengine.NewContext(request)

			moduleHost, _ := appengine.ModuleHostname(ctx, "", "", "")

			payment, err := p.CreatePayment(ctx, &Payment{
				Intent: "sale",
				Payer:  Payer{PaymentMethod: "paypal"},
				Transactions: []Transaction{
					{
						Amount: Amount{
							Total:    "30.11",
							Currency: "EUR",
							Details: AmountDetails{
								Subtotal: "30.00",
								Tax:      "0.07",
								Shipping: "0.03",
							},
						},
						Description:    "The payment transaction description.",
						Custom:         "ROCKETBOOK_EMS_90048630024435",
						InvoiceNumber:  "2314324234",
						PaymentOptions: PaymentOptions{AllowedPaymentMethod: "INSTANT_FUNDING_SOURCE"},
						SoftDescriptor: "ECHI5786786",
						ItemList: ItemList{
							Items: []Item{
								{
									Name:        "hat",
									Description: "Brown hat.",
									Quantity:    "2",
									Price:       "3",
									Tax:         "0.01",
									Sku:         "1",
									Currency:    "EUR",
								},
							},
						},
					},
				},
				NoteToPayer: "Contact us for any questions on your order.",
				RedirectUrls: RedirectUrls{
					ReturnUrl: moduleHost + modulePath + "/return",
					CancelUrl: moduleHost + modulePath + "/cancel",
				},
			})
			if err != nil {
				writer.Write([]byte(err.Error()))
				return
			}
			writer.Write([]byte(payment))
		})
		p.router.HandleFunc(modulePath+"/return", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte("return"))
		})
		p.router.HandleFunc(modulePath+"/cancel", func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte("cancel"))
		})
	}
	return p.router
}

func (p *PayPal) auth(ctx context.Context) error {
	// authorize if no credentials or if token expires in the next 60 seconds
	if p.credentials == nil || time.Now().Add(time.Second * time.Duration(60)).After(p.credentials.expiresAt) {
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

func (p *PayPal) CreatePayment(ctx context.Context, payment *Payment) (io.Reader, error) {
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
	return buf, nil
	/*responsePayment := new(Payment)
	err = json.Unmarshal(buf.Bytes(), responsePayment)
	if err != nil {
		return nil, err
	}

	return responsePayment, nil*/
}
