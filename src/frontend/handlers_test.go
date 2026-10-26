// Tests for handlers.go / rpc.go / middleware.go of the frontend service.
//
// Idea: instead of talking to the real microservices, every test starts ONE tiny
// in-process gRPC server (fakeBackend) that pretends to be cart / catalog /
// currency / shipping / recommendation / ad / checkout. The frontend handlers
// are then called through the same router that main.go uses.
package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
)

const testProductID = "OLJCESPC7Z"

// ---------------------------------------------------------------------------
// Fake backend (all gRPC services in one struct)
// ---------------------------------------------------------------------------

type fakeBackend struct {
	pb.UnimplementedCartServiceServer
	pb.UnimplementedProductCatalogServiceServer
	pb.UnimplementedCurrencyServiceServer
	pb.UnimplementedShippingServiceServer
	pb.UnimplementedRecommendationServiceServer
	pb.UnimplementedAdServiceServer
	pb.UnimplementedCheckoutServiceServer

	mu      sync.Mutex
	failing map[string]bool // gRPC method name -> make it return an error
	recoIDs []string
}

func (b *fakeBackend) fail(method string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failing[method] = true
}

func (b *fakeBackend) failed(method string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failing[method] {
		return status.Error(codes.Internal, "boom: "+method)
	}
	return nil
}

func usd(units int64, nanos int32) *pb.Money {
	return &pb.Money{CurrencyCode: "USD", Units: units, Nanos: nanos}
}

func (b *fakeBackend) GetSupportedCurrencies(ctx context.Context, _ *pb.Empty) (*pb.GetSupportedCurrenciesResponse, error) {
	if err := b.failed("GetSupportedCurrencies"); err != nil {
		return nil, err
	}
	// "XXX" is not whitelisted, so getCurrencies() must filter it out.
	return &pb.GetSupportedCurrenciesResponse{CurrencyCodes: []string{"USD", "EUR", "CAD", "XXX"}}, nil
}

func (b *fakeBackend) Convert(ctx context.Context, req *pb.CurrencyConversionRequest) (*pb.Money, error) {
	if err := b.failed("Convert"); err != nil {
		return nil, err
	}
	return &pb.Money{CurrencyCode: req.GetToCode(), Units: req.GetFrom().GetUnits(), Nanos: req.GetFrom().GetNanos()}, nil
}

func (b *fakeBackend) ListProducts(ctx context.Context, _ *pb.Empty) (*pb.ListProductsResponse, error) {
	if err := b.failed("ListProducts"); err != nil {
		return nil, err
	}
	return &pb.ListProductsResponse{Products: []*pb.Product{
		{Id: testProductID, Name: "Sunglasses", Description: "cool", Picture: "/static/img/products/sunglasses.jpg",
			PriceUsd: usd(19, 990000000), Categories: []string{"accessories"}},
	}}, nil
}

func (b *fakeBackend) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	if err := b.failed("GetProduct"); err != nil {
		return nil, err
	}
	if req.GetId() == "missing" {
		return nil, status.Error(codes.NotFound, "no such product")
	}
	return &pb.Product{Id: req.GetId(), Name: "Product " + req.GetId(), Description: "desc",
		Picture: "/static/img/products/x.jpg", PriceUsd: usd(19, 990000000), Categories: []string{"accessories"}}, nil
}

func (b *fakeBackend) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.Cart, error) {
	if err := b.failed("GetCart"); err != nil {
		return nil, err
	}
	return &pb.Cart{UserId: req.GetUserId(), Items: []*pb.CartItem{{ProductId: testProductID, Quantity: 2}}}, nil
}

func (b *fakeBackend) AddItem(ctx context.Context, _ *pb.AddItemRequest) (*pb.Empty, error) {
	if err := b.failed("AddItem"); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (b *fakeBackend) EmptyCart(ctx context.Context, _ *pb.EmptyCartRequest) (*pb.Empty, error) {
	if err := b.failed("EmptyCart"); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

func (b *fakeBackend) GetQuote(ctx context.Context, _ *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	if err := b.failed("GetQuote"); err != nil {
		return nil, err
	}
	return &pb.GetQuoteResponse{CostUsd: usd(8, 990000000)}, nil
}

func (b *fakeBackend) ListRecommendations(ctx context.Context, _ *pb.ListRecommendationsRequest) (*pb.ListRecommendationsResponse, error) {
	if err := b.failed("ListRecommendations"); err != nil {
		return nil, err
	}
	return &pb.ListRecommendationsResponse{ProductIds: b.recoIDs}, nil
}

func (b *fakeBackend) GetAds(ctx context.Context, _ *pb.AdRequest) (*pb.AdResponse, error) {
	if err := b.failed("GetAds"); err != nil {
		return nil, err
	}
	return &pb.AdResponse{Ads: []*pb.Ad{{RedirectUrl: "/product/" + testProductID, Text: "buy now"}}}, nil
}

func (b *fakeBackend) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
	if err := b.failed("PlaceOrder"); err != nil {
		return nil, err
	}
	return &pb.PlaceOrderResponse{Order: &pb.OrderResult{
		OrderId:            "order-123",
		ShippingTrackingId: "track-123",
		ShippingCost:       usd(8, 990000000),
		ShippingAddress:    req.GetAddress(),
		Items: []*pb.OrderItem{
			{Item: &pb.CartItem{ProductId: testProductID, Quantity: 1}, Cost: usd(19, 990000000)},
		},
	}}, nil
}

// ---------------------------------------------------------------------------
// Test environment
// ---------------------------------------------------------------------------

type testEnv struct {
	fe *frontendServer
	be *fakeBackend
	h  http.Handler
}

func quietLogger() *logrus.Logger {
	l := logrus.New()
	l.Out = io.Discard
	return l
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	be := &fakeBackend{failing: map[string]bool{}, recoIDs: []string{testProductID}}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	pb.RegisterCartServiceServer(srv, be)
	pb.RegisterProductCatalogServiceServer(srv, be)
	pb.RegisterCurrencyServiceServer(srv, be)
	pb.RegisterShippingServiceServer(srv, be)
	pb.RegisterRecommendationServiceServer(srv, be)
	pb.RegisterAdServiceServer(srv, be)
	pb.RegisterCheckoutServiceServer(srv, be)
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	dial := func() *grpc.ClientConn {
		c, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("grpc client: %v", err)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}

	fe := &frontendServer{
		productCatalogSvcConn: dial(),
		currencySvcConn:       dial(),
		cartSvcConn:           dial(),
		recommendationSvcConn: dial(),
		checkoutSvcConn:       dial(),
		shippingSvcConn:       dial(),
		adSvcConn:             dial(),
	}

	// Same routes as main.go
	r := mux.NewRouter()
	r.HandleFunc("/", fe.homeHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/product/{id}", fe.productHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/cart", fe.viewCartHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc("/cart", fe.addToCartHandler).Methods(http.MethodPost)
	r.HandleFunc("/cart/empty", fe.emptyCartHandler).Methods(http.MethodPost)
	r.HandleFunc("/setCurrency", fe.setCurrencyHandler).Methods(http.MethodPost)
	r.HandleFunc("/logout", fe.logoutHandler).Methods(http.MethodGet)
	r.HandleFunc("/cart/checkout", fe.placeOrderHandler).Methods(http.MethodPost)
	r.HandleFunc("/assistant", fe.assistantHandler).Methods(http.MethodGet)
	r.HandleFunc("/product-meta/{ids}", fe.getProductByID).Methods(http.MethodGet)
	r.HandleFunc("/bot", fe.chatBotHandler).Methods(http.MethodPost)

	var h http.Handler = r
	h = &logHandler{log: quietLogger(), next: h}
	h = ensureSessionID(h)

	return &testEnv{fe: fe, be: be, h: h}
}

func (e *testEnv) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func validOrderForm() url.Values {
	return url.Values{
		"email":                        {"someone@example.com"},
		"street_address":               {"1600 Amphitheatre Parkway"},
		"zip_code":                     {"94043"},
		"city":                         {"Mountain View"},
		"state":                        {"CA"},
		"country":                      {"United States"},
		"credit_card_number":           {"4111111111111111"},
		"credit_card_expiration_month": {"1"},
		"credit_card_expiration_year":  {"2035"},
		"credit_card_cvv":              {"672"},
	}
}

// ---------------------------------------------------------------------------
// Pure helper functions
// ---------------------------------------------------------------------------

func TestRenderMoney(t *testing.T) {
	if got := renderMoney(pb.Money{CurrencyCode: "EUR", Units: 12, Nanos: 340000000}); got != "€12.34" {
		t.Errorf("renderMoney EUR = %q", got)
	}
	if got := renderMoney(pb.Money{CurrencyCode: "USD", Units: 5, Nanos: 50000000}); got != "$5.05" {
		t.Errorf("renderMoney USD = %q", got)
	}
}

func TestRenderCurrencyLogo(t *testing.T) {
	cases := map[string]string{"USD": "$", "CAD": "$", "JPY": "¥", "EUR": "€", "TRY": "₺", "GBP": "£", "XYZ": "$"}
	for code, want := range cases {
		if got := renderCurrencyLogo(code); got != want {
			t.Errorf("renderCurrencyLogo(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestStringInSlice(t *testing.T) {
	if !stringinSlice(validEnvs, "gcp") {
		t.Error("gcp should be a valid env")
	}
	if stringinSlice(validEnvs, "mars") {
		t.Error("mars should not be a valid env")
	}
}

func TestCartHelpers(t *testing.T) {
	items := []*pb.CartItem{{ProductId: "a", Quantity: 2}, {ProductId: "b", Quantity: 3}}
	if got := cartSize(items); got != 5 {
		t.Errorf("cartSize = %d, want 5", got)
	}
	ids := cartIDs(items)
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("cartIDs = %v", ids)
	}
	if cartSize(nil) != 0 || len(cartIDs(nil)) != 0 {
		t.Error("empty cart helpers should return zero values")
	}
}

func TestSetPlatformDetails(t *testing.T) {
	cases := map[string]string{
		"aws": "AWS", "onprem": "On-Premises", "azure": "Azure",
		"gcp": "Google Cloud", "alibaba": "Alibaba Cloud", "local": "local", "other": "local",
	}
	for env, provider := range cases {
		p := &platformDetails{}
		p.setPlatformDetails(env)
		if p.provider != provider {
			t.Errorf("env %q: provider = %q, want %q", env, p.provider, provider)
		}
		if p.css == "" {
			t.Errorf("env %q: css should not be empty", env)
		}
	}
}

func TestCurrentCurrencyAndSessionID(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := currentCurrency(r); got != defaultCurrency {
		t.Errorf("default currency = %q", got)
	}
	r.AddCookie(&http.Cookie{Name: cookieCurrency, Value: "EUR"})
	if got := currentCurrency(r); got != "EUR" {
		t.Errorf("cookie currency = %q", got)
	}

	if got := sessionID(r); got != "" {
		t.Errorf("sessionID without context = %q", got)
	}
	r = r.WithContext(context.WithValue(r.Context(), ctxKeySessionID{}, "abc"))
	if got := sessionID(r); got != "abc" {
		t.Errorf("sessionID = %q", got)
	}
}

func TestInjectCommonTemplateData(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	data := injectCommonTemplateData(r, map[string]interface{}{"custom": 42, "user_currency": "JPY"})
	if data["custom"] != 42 {
		t.Error("payload values must be copied")
	}
	if data["user_currency"] != "JPY" {
		t.Error("payload must override defaults")
	}
	if _, ok := data["currentYear"]; !ok {
		t.Error("currentYear missing")
	}
}

func TestRenderHTTPError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	renderHTTPError(quietLogger(), r, rec, io.ErrUnexpectedEOF, http.StatusTeapot)
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("error page body should not be empty")
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func TestEnsureSessionIDSetsCookie(t *testing.T) {
	var seen string
	h := ensureSessionID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = sessionID(r)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if seen == "" {
		t.Error("session id should be generated")
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), cookieSessionID) {
		t.Error("session cookie should be set")
	}

	// Existing cookie is reused.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieSessionID, Value: "existing"})
	h.ServeHTTP(httptest.NewRecorder(), req)
	if seen != "existing" {
		t.Errorf("session id = %q, want existing", seen)
	}
}

// ---------------------------------------------------------------------------
// Page handlers (happy path)
// ---------------------------------------------------------------------------

func TestPagesRenderOK(t *testing.T) {
	e := newTestEnv(t)
	for _, path := range []string{"/", "/product/" + testProductID, "/cart", "/assistant"} {
		rec := e.do(http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, body: %s", path, rec.Code, rec.Body.String())
		}
		if rec.Body.Len() == 0 {
			t.Errorf("GET %s returned empty body", path)
		}
	}
}

func TestGetProductByID(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(http.MethodGet, "/product-meta/"+testProductID, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), testProductID) {
		t.Errorf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	// Unknown product -> handler just returns, nothing written.
	rec = e.do(http.MethodGet, "/product-meta/missing", nil)
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body for missing product, got %q", rec.Body.String())
	}
}

func TestProductHandlerWithoutID(t *testing.T) {
	e := newTestEnv(t)
	ctx := context.WithValue(context.Background(), ctxKeyLog{}, quietLogger())
	req := httptest.NewRequest(http.MethodGet, "/product/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	e.fe.productHandler(rec, req) // no mux vars -> empty id
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Cart / currency / logout / checkout
// ---------------------------------------------------------------------------

func TestAddToCart(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(http.MethodPost, "/cart", url.Values{"product_id": {testProductID}, "quantity": {"2"}})
	if rec.Code != http.StatusFound || rec.Header().Get("location") != "/cart" {
		t.Errorf("valid add: %d, location=%q", rec.Code, rec.Header().Get("location"))
	}

	rec = e.do(http.MethodPost, "/cart", url.Values{"product_id": {testProductID}, "quantity": {"0"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid quantity: %d, want 422", rec.Code)
	}

	rec = e.do(http.MethodPost, "/cart", url.Values{"product_id": {"missing"}, "quantity": {"1"}})
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("unknown product: %d, want 500", rec.Code)
	}
}

func TestEmptyCart(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(http.MethodPost, "/cart/empty", url.Values{})
	if rec.Code != http.StatusFound || rec.Header().Get("location") != "/" {
		t.Errorf("empty cart: %d, location=%q", rec.Code, rec.Header().Get("location"))
	}
}

func TestSetCurrency(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(http.MethodPost, "/setCurrency", url.Values{"currency_code": {"EUR"}})
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(strings.Join(rec.Header().Values("Set-Cookie"), ";"), cookieCurrency+"=EUR") {
		t.Error("currency cookie not set")
	}
	if rec.Header().Get("Location") != "/" {
		t.Errorf("Location = %q, want /", rec.Header().Get("Location"))
	}

	// With a referer we go back to it.
	req := httptest.NewRequest(http.MethodPost, "/setCurrency", strings.NewReader("currency_code=USD"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("referer", "/product/"+testProductID)
	rec = httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Header().Get("Location") != "/product/"+testProductID {
		t.Errorf("Location = %q", rec.Header().Get("Location"))
	}

	rec = e.do(http.MethodPost, "/setCurrency", url.Values{"currency_code": {"NOPE"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid currency: %d, want 422", rec.Code)
	}
}

func TestLogout(t *testing.T) {
	e := newTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: cookieSessionID, Value: "s1"})
	req.AddCookie(&http.Cookie{Name: cookieCurrency, Value: "EUR"})
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Errorf("logout: %d, location=%q", rec.Code, rec.Header().Get("Location"))
	}
	if len(rec.Header().Values("Set-Cookie")) < 2 {
		t.Error("all cookies should be expired")
	}
}

func TestPlaceOrder(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(http.MethodPost, "/cart/checkout", validOrderForm())
	if rec.Code != http.StatusOK {
		t.Errorf("valid order: %d, body: %s", rec.Code, rec.Body.String())
	}

	bad := validOrderForm()
	bad.Set("email", "not-an-email")
	rec = e.do(http.MethodPost, "/cart/checkout", bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("invalid email: %d, want 422", rec.Code)
	}

	e.be.fail("PlaceOrder")
	rec = e.do(http.MethodPost, "/cart/checkout", validOrderForm())
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("checkout failure: %d, want 500", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Backend failures must become HTTP 500 (not a crash)
// ---------------------------------------------------------------------------

func TestHandlersReturn500WhenBackendFails(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		form   url.Values
		fail   string
	}{
		{"home currencies", http.MethodGet, "/", nil, "GetSupportedCurrencies"},
		{"home products", http.MethodGet, "/", nil, "ListProducts"},
		{"home cart", http.MethodGet, "/", nil, "GetCart"},
		{"home convert", http.MethodGet, "/", nil, "Convert"},
		{"product get", http.MethodGet, "/product/" + testProductID, nil, "GetProduct"},
		{"product currencies", http.MethodGet, "/product/" + testProductID, nil, "GetSupportedCurrencies"},
		{"product cart", http.MethodGet, "/product/" + testProductID, nil, "GetCart"},
		{"product convert", http.MethodGet, "/product/" + testProductID, nil, "Convert"},
		{"cart currencies", http.MethodGet, "/cart", nil, "GetSupportedCurrencies"},
		{"cart get", http.MethodGet, "/cart", nil, "GetCart"},
		{"cart quote", http.MethodGet, "/cart", nil, "GetQuote"},
		{"cart product", http.MethodGet, "/cart", nil, "GetProduct"},
		{"cart convert", http.MethodGet, "/cart", nil, "Convert"},
		{"add insert", http.MethodPost, "/cart", url.Values{"product_id": {testProductID}, "quantity": {"1"}}, "AddItem"},
		{"empty cart", http.MethodPost, "/cart/empty", url.Values{}, "EmptyCart"},
		{"assistant currencies", http.MethodGet, "/assistant", nil, "GetSupportedCurrencies"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEnv(t)
			e.be.fail(tc.fail)
			rec := e.do(tc.method, tc.path, tc.form)
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", rec.Code)
			}
		})
	}
}

// Recommendations and ads are "not critical": the page must still render.
func TestNonCriticalFailuresStillRender(t *testing.T) {
	e := newTestEnv(t)
	e.be.fail("ListRecommendations")
	e.be.fail("GetAds")
	for _, path := range []string{"/", "/product/" + testProductID, "/cart"} {
		if rec := e.do(http.MethodGet, path, nil); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// rpc.go helpers called directly
// ---------------------------------------------------------------------------

func TestGetCurrenciesFiltersWhitelist(t *testing.T) {
	e := newTestEnv(t)
	got, err := e.fe.getCurrencies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if c == "XXX" {
			t.Error("non whitelisted currency should be filtered out")
		}
	}
	if len(got) != 3 {
		t.Errorf("got %v, want 3 whitelisted currencies", got)
	}
}

func TestGetRecommendationsKeepsOnlyFour(t *testing.T) {
	e := newTestEnv(t)
	e.be.recoIDs = []string{"p1", "p2", "p3", "p4", "p5", "p6"}
	got, err := e.fe.getRecommendations(context.Background(), "u1", []string{"p1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("len = %d, want 4", len(got))
	}
}

func TestGetRecommendationsUnknownProductFails(t *testing.T) {
	e := newTestEnv(t)
	e.be.recoIDs = []string{"missing"}
	if _, err := e.fe.getRecommendations(context.Background(), "u1", nil); err == nil {
		t.Error("expected an error for an unknown recommended product")
	}
}

func TestGetShippingQuote(t *testing.T) {
	e := newTestEnv(t)
	got, err := e.fe.getShippingQuote(context.Background(), []*pb.CartItem{{ProductId: testProductID, Quantity: 1}}, "EUR")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetCurrencyCode() != "EUR" || got.GetUnits() != 8 {
		t.Errorf("quote = %v", got)
	}

	e.be.fail("GetQuote")
	if _, err := e.fe.getShippingQuote(context.Background(), nil, "USD"); err == nil {
		t.Error("expected error when shipping service fails")
	}
}

func TestChooseAd(t *testing.T) {
	e := newTestEnv(t)
	ad := e.fe.chooseAd(context.Background(), []string{"accessories"}, quietLogger())
	if ad == nil || ad.GetText() != "buy now" {
		t.Errorf("ad = %v", ad)
	}
	e.be.fail("GetAds")
	if ad := e.fe.chooseAd(context.Background(), nil, quietLogger()); ad != nil {
		t.Error("ad should be nil when the ad service fails")
	}
}

// ---------------------------------------------------------------------------
// Chat bot handler (talks plain HTTP to the shopping assistant)
// ---------------------------------------------------------------------------

func TestChatBotHandler(t *testing.T) {
	e := newTestEnv(t)

	assistant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":"hello from the bot","details":{}}`))
	}))
	defer assistant.Close()
	e.fe.shoppingAssistantSvcAddr = strings.TrimPrefix(assistant.URL, "http://")

	req := httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(`{"message":"hi"}`))
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var out struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("bad json: %v (%s)", err, rec.Body.String())
	}
	if out.Message != "hello from the bot" {
		t.Errorf("message = %q", out.Message)
	}
}

func TestChatBotHandlerBadResponse(t *testing.T) {
	e := newTestEnv(t)
	assistant := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`this is not json`))
	}))
	defer assistant.Close()
	e.fe.shoppingAssistantSvcAddr = strings.TrimPrefix(assistant.URL, "http://")

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestChatBotHandlerUnreachable(t *testing.T) {
	e := newTestEnv(t)
	e.fe.shoppingAssistantSvcAddr = "127.0.0.1:1" // nothing listens here

	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
