package guest_test

//
//import (
//	"bytes"
//	"encoding/json"
//	"net"
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//	"time"
//
//	"github.com/go-chi/chi/v5"
//
//	// your project imports
//	"github.com/diagnosis/luxsuv-bookings/internal/domain"
//	guest "github.com/diagnosis/luxsuv-bookings/internal/http/handlers/guest"
//	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
//)
//
//// ---------- fakes / mocks ----------
//
//type fakeMailer struct {
//	lastTo   string
//	lastBody string
//}
//
//func (m *fakeMailer) Send(toEmail, toName, subj, textBody, htmlBody string) (string, error) {
//	m.lastTo = toEmail
//	m.lastBody = textBody + htmlBody
//	return "ok", nil
//}
//func (m *fakeMailer) SendGuestAccess(email, code, link string) (string, error) {
//	m.lastTo = email
//	m.lastBody = code + " " + link
//	return "ok", nil
//}
//
//// Minimal VerifyRepo fake
//type fakeVerifyRepo struct {
//	// email -> bcrypt(code) not needed; we’ll accept any code we stored raw for simplicity
//	codeByEmail   map[string]string
//	magicToEmail  map[string]string
//	expireByEmail map[string]time.Time
//}
//
//func newFakeVerifyRepo() *fakeVerifyRepo {
//	return &fakeVerifyRepo{
//		codeByEmail:   map[string]string{},
//		magicToEmail:  map[string]string{},
//		expireByEmail: map[string]time.Time{},
//	}
//}
//func (f *fakeVerifyRepo) CreateGuestAccess(_ interface{}, email, codeHash, magic string, exp time.Time, _ net.IP) error {
//	// store raw “codeHash” as code to simplify; handler uses bcrypt in real impl
//	f.codeByEmail[email] = codeHash
//	f.magicToEmail[magic] = email
//	f.expireByEmail[email] = exp
//	return nil
//}
//func (f *fakeVerifyRepo) CheckGuestCode(_ interface{}, email, code string) (bool, error) {
//	want, ok := f.codeByEmail[email]
//	if !ok {
//		return false, nil
//	}
//	if time.Now().After(f.expireByEmail[email]) {
//		return false, nil
//	}
//	return want == code, nil
//}
//func (f *fakeVerifyRepo) ConsumeGuestMagic(_ interface{}, token string) (string, bool, error) {
//	email, ok := f.magicToEmail[token]
//	if !ok {
//		return "", false, nil
//	}
//	delete(f.magicToEmail, token)
//	return email, true, nil
//}
//
//// Minimal BookingRepo fake (in-memory)
//type fakeBookingRepo struct {
//	seq     int64
//	byID    map[int64]*domain.Booking
//	byToken map[string]int64
//	byEmail map[string][]int64
//}
//
//func newFakeBookingRepo() *fakeBookingRepo {
//	return &fakeBookingRepo{
//		seq:     0,
//		byID:    map[int64]*domain.Booking{},
//		byToken: map[string]int64{},
//		byEmail: map[string][]int64{},
//	}
//}
//
//func (r *fakeBookingRepo) CreateGuest(_ interface{}, in *domain.BookingGuestReq) (*domain.Booking, error) {
//	r.seq++
//	id := r.seq
//	tok := "tok-" + time.Now().Format("150405.000") + "-" + string(rune(id))
//	b := &domain.Booking{
//		ID:          id,
//		ManageToken: tok,
//		Status:      domain.BookingPending,
//		RiderName:   in.RiderName,
//		RiderEmail:  strings.ToLower(in.RiderEmail),
//		RiderPhone:  in.RiderPhone,
//		Pickup:      in.Pickup,
//		Dropoff:     in.Dropoff,
//		ScheduledAt: in.ScheduledAt,
//		Notes:       in.Notes,
//		Passengers:  in.Passengers,
//		Luggages:    in.Luggages,
//		RideType:    in.RideType,
//		CreatedAt:   time.Now(),
//		UpdatedAt:   time.Now(),
//	}
//	r.byID[id] = b
//	r.byToken[tok] = id
//	k := strings.ToLower(in.RiderEmail)
//	r.byEmail[k] = append(r.byEmail[k], id)
//	return b, nil
//}
//func (r *fakeBookingRepo) GetByIDWithToken(_ interface{}, id int64, token string) (*domain.Booking, error) {
//	if rid, ok := r.byToken[token]; !ok || rid != id {
//		return nil, nil
//	}
//	return r.byID[id], nil
//}
//func (r *fakeBookingRepo) CancelWithToken(_ interface{}, id int64, token string) (bool, error) {
//	b, _ := r.GetByIDWithToken(nil, id, token)
//	if b == nil {
//		return false, nil
//	}
//	if b.Status == domain.BookingCanceled {
//		return false, nil
//	}
//	b.Status = domain.BookingCanceled
//	b.UpdatedAt = time.Now()
//	return true, nil
//}
//func (r *fakeBookingRepo) List(_ interface{}, limit, offset int) ([]domain.Booking, error) {
//	out := []domain.Booking{}
//	for _, b := range r.byID {
//		out = append(out, *b)
//	}
//	// simple slice
//	if offset > len(out) {
//		return []domain.Booking{}, nil
//	}
//	end := offset + limit
//	if end > len(out) {
//		end = len(out)
//	}
//	return out[offset:end], nil
//}
//func (r *fakeBookingRepo) ListByStatus(_ interface{}, st domain.BookingStatus, limit, offset int) ([]domain.Booking, error) {
//	all, _ := r.List(nil, 1000, 0)
//	out := []domain.Booking{}
//	for _, b := range all {
//		if b.Status == st {
//			out = append(out, b)
//		}
//	}
//	if offset > len(out) {
//		return []domain.Booking{}, nil
//	}
//	end := offset + limit
//	if end > len(out) {
//		end = len(out)
//	}
//	return out[offset:end], nil
//}
//func (r *fakeBookingRepo) GetByID(_ interface{}, id int64) (*domain.Booking, error) {
//	return r.byID[id], nil
//}
//func (r *fakeBookingRepo) ListByUserID(_ interface{}, _ int64, limit, offset int, _ *domain.BookingStatus) ([]domain.Booking, error) {
//	return r.List(nil, limit, offset)
//}
//func (r *fakeBookingRepo) CreateForUser(ctx interface{}, userID int64, in *domain.BookingGuestReq) (*domain.Booking, error) {
//	return r.CreateGuest(ctx, in)
//}
//func (r *fakeBookingRepo) UpdateGuest(_ interface{}, id int64, token string, p domain.GuestPatch) (*domain.Booking, error) {
//	b, _ := r.GetByIDWithToken(nil, id, token)
//	if b == nil {
//		return nil, nil
//	}
//	if p.RiderName != nil {
//		b.RiderName = *p.RiderName
//	}
//	if p.RiderPhone != nil {
//		b.RiderPhone = *p.RiderPhone
//	}
//	if p.Pickup != nil {
//		b.Pickup = *p.Pickup
//	}
//	if p.Dropoff != nil {
//		b.Dropoff = *p.Dropoff
//	}
//	if p.ScheduledAt != nil {
//		b.ScheduledAt = *p.ScheduledAt
//	}
//	if p.Notes != nil {
//		b.Notes = *p.Notes
//	}
//	if p.Passengers != nil {
//		b.Passengers = *p.Passengers
//	}
//	if p.Luggages != nil {
//		b.Luggages = *p.Luggages
//	}
//	if p.RideType != nil {
//		b.RideType = *p.RideType
//	}
//	b.UpdatedAt = time.Now()
//	return b, nil
//}
//func (r *fakeBookingRepo) ListByEmail(_ interface{}, email string, limit, offset int, _ *domain.BookingStatus) ([]domain.Booking, error) {
//	ids := r.byEmail[strings.ToLower(email)]
//	out := []domain.Booking{}
//	for _, id := range ids {
//		out = append(out, *r.byID[id])
//	}
//	if offset > len(out) {
//		return []domain.Booking{}, nil
//	}
//	end := offset + limit
//	if end > len(out) {
//		end = len(out)
//	}
//	return out[offset:end], nil
////}
//
//
//// ---------- helpers ----------
//
//func newGuestServer(t *testing.T) (*httptest.Server, *fakeBookingRepo, *fakeVerifyRepo, *fakeMailer) {
//	t.Helper()
//	bookRepo := newFakeBookingRepo()
//	verRepo := newFakeVerifyRepo()
//	m := &fakeMailer{}
//
//	access := guest.NewAccessHandler(verRepo, m)
//	bookings := guest.NewBookingsHandler(bookRepo)
//
//	r := chi.NewRouter()
//	r.Mount("/v1/guest/access", access.Routes())
//	r.Mount("/v1/guest/bookings", bookings.Routes())
//
//	return httptest.NewServer(r), bookRepo, verRepo, m
//}
//
//func authz(token string) http.Header {
//	h := http.Header{}
//	h.Set("Authorization", "Bearer "+token)
//	return h
//}
//
//// ---------- tests ----------
//
//func TestGuest_Create_And_Get_ByToken(t *testing.T) {
//	srv, _, _, _ := newGuestServer(t)
//	defer srv.Close()
//
//	when := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
//	body := `{
//		"rider_name":"Guest One",
//		"rider_email":"guest1@example.com",
//		"rider_phone":"+15551234567",
//		"pickup":"SFO","dropoff":"Downtown",
//		"scheduled_at":"` + when + `",
//		"notes":"2 bags",
//		"passengers":2,"luggages":2,"ride_type":"per_ride"
//	}`
//
//	req := httptest.NewRequest(http.MethodPost, srv.URL+"/v1/guest/bookings", bytes.NewBufferString(body))
//	req.Header.Set("Content-Type", "application/json")
//	res := httptest.NewRecorder()
//	http.DefaultClient.Do(req) // no-op; use server directly below (we’re showing the intent)
//	// we’ll just use http.Post for simplicity:
//	resp, err := http.Post(srv.URL+"/v1/guest/bookings", "application/json", bytes.NewBufferString(body))
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer resp.Body.Close()
//	if resp.StatusCode != http.StatusCreated {
//		t.Fatalf("expected 201, got %d", resp.StatusCode)
//	}
//	var out struct {
//		ID          int64     `json:"id"`
//		ManageToken string    `json:"manage_token"`
//		Status      string    `json:"status"`
//		ScheduledAt time.Time `json:"scheduled_at"`
//	}
//	_ = json.NewDecoder(resp.Body).Decode(&out)
//	if out.ID == 0 || out.ManageToken == "" {
//		t.Fatalf("missing id/token in response: %+v", out)
//	}
//
//	// fetch by manage_token
//	getURL := srv.URL + "/v1/guest/bookings/" + intToStr(out.ID) + "?manage_token=" + out.ManageToken
//	res2, err := http.Get(getURL)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer res2.Body.Close()
//	if res2.StatusCode != http.StatusOK {
//		t.Fatalf("expected 200, got %d", res2.StatusCode)
//	}
//}
//
//func TestGuest_List_Unauthorized_Then_Authorized(t *testing.T) {
//	srv, _, ver, mail := newGuestServer(t)
//	defer srv.Close()
//
//	// Request access
//	email := "guest2@example.com"
//	reqBody := `{"email":"` + email + `"}`
//	resp, err := http.Post(srv.URL+"/v1/guest/access/request", "application/json", bytes.NewBufferString(reqBody))
//	if err != nil {
//		t.Fatal(err)
//	}
//	resp.Body.Close()
//	if resp.StatusCode != http.StatusOK {
//		t.Fatalf("request access expected 200, got %d", resp.StatusCode)
//	}
//	if mail.lastTo != email || !strings.Contains(mail.lastBody, "http://") {
//		t.Fatalf("mailer not invoked correctly: to=%s body=%q", mail.lastTo, mail.lastBody)
//	}
//
//	// Verify with the code we stored (faked as “hash”)
//	code := ver.codeByEmail[email]
//	verifyBody := `{"email":"` + email + `","code":"` + code + `"}`
//	resp2, err := http.Post(srv.URL+"/v1/guest/access/verify", "application/json", bytes.NewBufferString(verifyBody))
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer resp2.Body.Close()
//	if resp2.StatusCode != http.StatusOK {
//		t.Fatalf("verify expected 200, got %d", resp2.StatusCode)
//	}
//	var vout struct {
//		SessionToken string `json:"session_token"`
//	}
//	_ = json.NewDecoder(resp2.Body).Decode(&vout)
//	if vout.SessionToken == "" {
//		t.Fatalf("missing session_token")
//	}
//
//	// List WITHOUT token -> 401
//	reqNoAuth, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/guest/bookings?limit=10&offset=0", nil)
//	resp3, _ := http.DefaultClient.Do(reqNoAuth)
//	if resp3.StatusCode != http.StatusUnauthorized {
//		t.Fatalf("expected 401 without token, got %d", resp3.StatusCode)
//	}
//	resp3.Body.Close()
//
//	// List WITH token -> 200 (even if empty list)
//	reqAuth, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/guest/bookings?limit=10&offset=0", nil)
//	reqAuth.Header = authz(vout.SessionToken)
//	resp4, _ := http.DefaultClient.Do(reqAuth)
//	defer resp4.Body.Close()
//	if resp4.StatusCode != http.StatusOK {
//		t.Fatalf("expected 200 with token, got %d", resp4.StatusCode)
//	}
//}
//
//func TestGuest_Patch_And_Cancel_ByToken(t *testing.T) {
//	srv, repo, _, _ := newGuestServer(t)
//	defer srv.Close()
//
//	// seed booking
//	b, _ := repo.CreateGuest(nil, &domain.BookingGuestReq{
//		RiderName:   "G",
//		RiderEmail:  "g@example.com",
//		RiderPhone:  "+1",
//		Pickup:      "A",
//		Dropoff:     "B",
//		ScheduledAt: time.Now().Add(1 * time.Hour).UTC(),
//		Notes:       "n",
//		Passengers:  1,
//		Luggages:    0,
//		RideType:    domain.RidePerRide,
//	})
//
//	// invalid passengers -> 400
//	patchURL := srv.URL + "/v1/guest/bookings/" + intToStr(b.ID) + "?manage_token=" + b.ManageToken
//	invalid := `{"passengers":0}`
//	reqBad, _ := http.NewRequest(http.MethodPatch, patchURL, bytes.NewBufferString(invalid))
//	reqBad.Header.Set("Content-Type", "application/json")
//	respBad, _ := http.DefaultClient.Do(reqBad)
//	defer respBad.Body.Close()
//	if respBad.StatusCode != http.StatusBadRequest {
//		t.Fatalf("expected 400, got %d", respBad.StatusCode)
//	}
//
//	// valid notes update -> 200 and notes changed
//	valid := `{"notes":"updated notes"}`
//	reqOK, _ := http.NewRequest(http.MethodPatch, patchURL, bytes.NewBufferString(valid))
//	reqOK.Header.Set("Content-Type", "application/json")
//	respOK, _ := http.DefaultClient.Do(reqOK)
//	defer respOK.Body.Close()
//	if respOK.StatusCode != http.StatusOK {
//		t.Fatalf("expected 200, got %d", respOK.StatusCode)
//	}
//	var bout domain.Booking
//	_ = json.NewDecoder(respOK.Body).Decode(&bout)
//	if bout.Notes != "updated notes" {
//		t.Fatalf("notes not updated")
//	}
//
//	// cancel -> 204
//	reqDel, _ := http.NewRequest(http.MethodDelete, patchURL, nil)
//	respDel, _ := http.DefaultClient.Do(reqDel)
//	defer respDel.Body.Close()
//	if respDel.StatusCode != http.StatusNoContent {
//		t.Fatalf("expected 204, got %d", respDel.StatusCode)
//	}
//	if repo.byID[b.ID].Status != domain.BookingCanceled {
//		t.Fatalf("booking not canceled")
//	}
//}
//
//func TestAccess_Magic_Link(t *testing.T) {
//	srv, _, ver, mail := newGuestServer(t)
//	defer srv.Close()
//
//	// Request
//	email := "m@example.com"
//	_ = httpPostJSON(t, srv.URL+"/v1/guest/access/request", map[string]string{"email": email}, http.StatusOK)
//	if mail.lastTo != email {
//		t.Fatalf("expected mail to %s", email)
//	}
//	// find magic token from fake store
//	var magic string
//	for tok, em := range ver.magicToEmail {
//		if em == email {
//			magic = tok
//			break
//		}
//	}
//	if magic == "" {
//		t.Fatalf("magic token not stored")
//	}
//
//	// Call magic endpoint
//	resp := httpGet(t, srv.URL+"/v1/guest/access/magic?token="+magic, http.StatusOK)
//	defer resp.Body.Close()
//	var out struct{ SessionToken string }
//	_ = json.NewDecoder(resp.Body).Decode(&out)
//	if out.SessionToken == "" {
//		t.Fatalf("missing session_token from magic")
//	}
//	// token is a JWT—quick sanity: parse claims
//	if _, err := auth.Parse(out.SessionToken); err != nil {
//		t.Fatalf("session_token not parseable: %v", err)
//	}
//}
//
//// ---------- tiny utilities ----------
//
//func intToStr(i int64) string { return strconvFormatInt(i, 10) }
//
//func strconvFormatInt(i int64, base int) string { return strconvItoa(int(i)) }
//
//func strconvItoa(i int) string {
//	// simple no-import helper to keep the snippet self-contained
//	// (you can replace with strconv.Itoa)
//	sign := ""
//	if i < 0 {
//		sign = "-"
//		i = -i
//	}
//	if i == 0 {
//		return "0"
//	}
//	d := []byte{}
//	for i > 0 {
//		d = append([]byte{byte('0' + i%10)}, d...)
//		i /= 10
//	}
//	return sign + string(d)
//}
//
//func httpPostJSON(t *testing.T, url string, payload any, want int) *http.Response {
//	t.Helper()
//	buf, _ := json.Marshal(payload)
//	resp, err := http.Post(url, "application/json", bytes.NewBuffer(buf))
//	if err != nil {
//		t.Fatalf("post %s: %v", url, err)
//	}
//	if resp.StatusCode != want {
//		t.Fatalf("POST %s: want %d got %d", url, want, resp.StatusCode)
//	}
//	return resp
//}
//
//func httpGet(t *testing.T, url string, want int) *http.Response {
//	t.Helper()
//	resp, err := http.Get(url)
//	if err != nil {
//		t.Fatalf("get %s: %v", url, err)
//	}
//	if resp.StatusCode != want {
//		t.Fatalf("GET %s: want %d got %d", url, want, resp.StatusCode)
//	}
//	return resp
//}
