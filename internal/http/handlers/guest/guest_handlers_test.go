package guest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/diagnosis/luxsuv-bookings/internal/domain"
	"github.com/diagnosis/luxsuv-bookings/internal/http/handlers/guest"
	"github.com/diagnosis/luxsuv-bookings/internal/platform/auth"
	"github.com/diagnosis/luxsuv-bookings/internal/repo/postgres"
)

// ---------- Mocks ----------

type mockMailer struct {
	lastTo   string
	lastCode string
	lastLink string
	sendErr  error
}

func (m *mockMailer) Send(toEmail, toName, subject, text, html string) (string, error) {
	m.lastTo = toEmail
	return "mock-id", m.sendErr
}

func (m *mockMailer) SendGuestAccess(email, code, link string) error {
	m.lastTo = email
	m.lastCode = code
	m.lastLink = link
	return m.sendErr
}

type mockVerifyRepo struct {
	codes          map[string]string // email -> code
	magicTokens    map[string]string // token -> email
	expirations    map[string]time.Time
	checkCodeErr   error
	createAccessErr error
}

func newMockVerifyRepo() *mockVerifyRepo {
	return &mockVerifyRepo{
		codes:       make(map[string]string),
		magicTokens: make(map[string]string),
		expirations: make(map[string]time.Time),
	}
}

func (m *mockVerifyRepo) CreateGuestAccess(_ context.Context, email, codeHash, magic string, expires time.Time, _ net.IP) error {
	if m.createAccessErr != nil {
		return m.createAccessErr
	}
	// Store raw code for simplicity in tests
	m.codes[email] = codeHash
	m.magicTokens[magic] = email
	m.expirations[email] = expires
	return nil
}

func (m *mockVerifyRepo) CheckGuestCode(_ context.Context, email, code string) (bool, error) {
	if m.checkCodeErr != nil {
		return false, m.checkCodeErr
	}
	storedCode, exists := m.codes[email]
	if !exists {
		return false, nil
	}
	if time.Now().After(m.expirations[email]) {
		return false, nil
	}
	return storedCode == code, nil
}

func (m *mockVerifyRepo) ConsumeGuestMagic(_ context.Context, token string) (string, bool, error) {
	email, exists := m.magicTokens[token]
	if !exists {
		return "", false, nil
	}
	delete(m.magicTokens, token)
	return email, true, nil
}

// Implement other interface methods as no-ops
func (m *mockVerifyRepo) CreateEmailVerification(context.Context, int64, string, time.Time) error { return nil }
func (m *mockVerifyRepo) ConsumeEmailVerification(context.Context, string) (int64, error) { return 0, nil }
func (m *mockVerifyRepo) MarkUserVerified(context.Context, int64) error { return nil }
func (m *mockVerifyRepo) IsUserVerified(context.Context, int64) (bool, error) { return false, nil }
func (m *mockVerifyRepo) DeleteExpiredTokens(context.Context) (int64, error) { return 0, nil }

type mockBookingRepo struct {
	nextID   int64
	bookings map[int64]*domain.Booking
	tokens   map[string]int64 // manage_token -> booking_id
	emails   map[string][]int64 // email -> []booking_ids
}

func newMockBookingRepo() *mockBookingRepo {
	return &mockBookingRepo{
		nextID:   1,
		bookings: make(map[int64]*domain.Booking),
		tokens:   make(map[string]int64),
		emails:   make(map[string][]int64),
	}
}

func (m *mockBookingRepo) CreateGuest(_ context.Context, req *domain.BookingGuestReq) (*domain.Booking, error) {
	id := m.nextID
	m.nextID++
	
	token := fmt.Sprintf("token-%d", id)
	booking := &domain.Booking{
		ID:          id,
		ManageToken: token,
		Status:      domain.BookingPending,
		RiderName:   req.RiderName,
		RiderEmail:  req.RiderEmail,
		RiderPhone:  req.RiderPhone,
		Pickup:      req.Pickup,
		Dropoff:     req.Dropoff,
		ScheduledAt: req.ScheduledAt,
		Notes:       req.Notes,
		Passengers:  req.Passengers,
		Luggages:    req.Luggages,
		RideType:    req.RideType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	m.bookings[id] = booking
	m.tokens[token] = id
	
	email := strings.ToLower(req.RiderEmail)
	m.emails[email] = append(m.emails[email], id)
	
	return booking, nil
}

func (m *mockBookingRepo) GetByID(_ context.Context, id int64) (*domain.Booking, error) {
	booking, exists := m.bookings[id]
	if !exists {
		return nil, nil
	}
	return booking, nil
}

func (m *mockBookingRepo) GetByIDWithToken(_ context.Context, id int64, token string) (*domain.Booking, error) {
	bookingID, exists := m.tokens[token]
	if !exists || bookingID != id {
		return nil, nil
	}
	return m.bookings[id], nil
}

func (m *mockBookingRepo) ListByEmail(_ context.Context, email string, limit, offset int, status *domain.BookingStatus) ([]domain.Booking, error) {
	email = strings.ToLower(email)
	ids := m.emails[email]
	
	var result []domain.Booking
	for _, id := range ids {
		booking := m.bookings[id]
		if status != nil && booking.Status != *status {
			continue
		}
		result = append(result, *booking)
	}
	
	// Apply pagination
	if offset >= len(result) {
		return []domain.Booking{}, nil
	}
	end := offset + limit
	if end > len(result) {
		end = len(result)
	}
	
	return result[offset:end], nil
}

func (m *mockBookingRepo) UpdateGuest(_ context.Context, id int64, token string, patch domain.GuestPatch) (*domain.Booking, error) {
	bookingID, exists := m.tokens[token]
	if !exists || bookingID != id {
		return nil, nil
	}
	
	booking := m.bookings[id]
	if patch.Notes != nil {
		booking.Notes = *patch.Notes
	}
	if patch.Passengers != nil {
		booking.Passengers = *patch.Passengers
	}
	booking.UpdatedAt = time.Now()
	
	return booking, nil
}

func (m *mockBookingRepo) CancelWithToken(_ context.Context, id int64, token string) (bool, error) {
	bookingID, exists := m.tokens[token]
	if !exists || bookingID != id {
		return false, nil
	}
	
	booking := m.bookings[id]
	if booking.Status == domain.BookingCanceled {
		return false, nil
	}
	
	booking.Status = domain.BookingCanceled
	booking.UpdatedAt = time.Now()
	return true, nil
}

// Implement remaining interface methods as no-ops for testing
func (m *mockBookingRepo) List(context.Context, int, int) ([]domain.Booking, error) { return nil, nil }
func (m *mockBookingRepo) ListByStatus(context.Context, domain.BookingStatus, int, int) ([]domain.Booking, error) { return nil, nil }
func (m *mockBookingRepo) ListByUserID(context.Context, int64, int, int, *domain.BookingStatus) ([]domain.Booking, error) { return nil, nil }
func (m *mockBookingRepo) CreateForUser(context.Context, int64, *domain.BookingGuestReq) (*domain.Booking, error) { return nil, nil }

type mockIdempotencyRepo struct {
	records map[string]int64 // key_hash -> booking_id
}

func newMockIdempotencyRepo() *mockIdempotencyRepo {
	return &mockIdempotencyRepo{
		records: make(map[string]int64),
	}
}

func (m *mockIdempotencyRepo) CheckOrCreateIdempotency(_ context.Context, key string, bookingID int64) (int64, error) {
	// Simple hash for testing
	keyHash := fmt.Sprintf("hash-%s", key)
	
	if existingBookingID, exists := m.records[keyHash]; exists {
		return existingBookingID, nil
	}
	
	if bookingID > 0 {
		m.records[keyHash] = bookingID
	}
	
	return 0, nil
}

func (m *mockIdempotencyRepo) CleanupExpired(context.Context) (int64, error) {
	return 0, nil
}

// Mock Users Repository
type mockUsersRepo struct {
	users map[string]*postgres.User // email -> user
}

func newMockUsersRepo() *mockUsersRepo {
	return &mockUsersRepo{
		users: make(map[string]*postgres.User),
	}
}

func (m *mockUsersRepo) Create(ctx context.Context, email, hash, name, phone string) (*postgres.User, error) {
	user := &postgres.User{
		ID: int64(len(m.users) + 1),
		Email: email,
		PasswordHash: hash,
		Name: name,
		Phone: phone,
		Role: "rider",
	}
	m.users[email] = user
	return user, nil
}

func (m *mockUsersRepo) FindByEmail(ctx context.Context, email string) (*postgres.User, error) {
	if user, exists := m.users[email]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockUsersRepo) FindByID(ctx context.Context, id int64) (*postgres.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *mockUsersRepo) LinkExistingBookings(ctx context.Context, userID int64, email string) error {
	return nil
}

// ---------- Test Setup ----------

func setupTestServer() (*httptest.Server, *mockBookingRepo, *mockVerifyRepo, *mockMailer, *mockIdempotencyRepo) {
	bookingRepo := newMockBookingRepo()
	verifyRepo := newMockVerifyRepo()
	mailer := &mockMailer{}
	idempotencyRepo := newMockIdempotencyRepo()
	usersRepo := newMockUsersRepo()
	
	accessHandler := guest.NewAccessHandler(verifyRepo, mailer, usersRepo)
	bookingsHandler := guest.NewBookingsHandler(bookingRepo, idempotencyRepo, usersRepo)
	
	r := chi.NewRouter()
	r.Mount("/v1/guest/access", accessHandler.Routes())
	r.Mount("/v1/guest/bookings", bookingsHandler.Routes())
	
	return httptest.NewServer(r), bookingRepo, verifyRepo, mailer, idempotencyRepo
}

// ---------- Tests ----------

func TestGuestAccess_RequestAndVerify_Success(t *testing.T) {
	server, _, verifyRepo, mailer, _ := setupTestServer()
	defer server.Close()
	
	email := "test@example.com"
	
	// Test access request
	requestBody := map[string]string{"email": email}
	resp := postJSON(t, server.URL+"/v1/guest/access/request", requestBody, http.StatusOK)
	
	var requestResult map[string]string
	json.NewDecoder(resp.Body).Decode(&requestResult)
	resp.Body.Close()
	
	if requestResult["message"] == "" {
		t.Fatal("Expected success message")
	}
	
	if mailer.lastTo != email {
		t.Fatalf("Expected email to %s, got %s", email, mailer.lastTo)
	}
	
	// Get the code from our mock (in real implementation it would be from email)
	code := verifyRepo.codes[email]
	if code == "" {
		t.Fatal("No code stored")
	}
	
	// Test code verification
	verifyBody := map[string]string{"email": email, "code": code}
	verifyResp := postJSON(t, server.URL+"/v1/guest/access/verify", verifyBody, http.StatusOK)
	
	var verifyResult map[string]interface{}
	json.NewDecoder(verifyResp.Body).Decode(&verifyResult)
	verifyResp.Body.Close()
	
	token, ok := verifyResult["session_token"].(string)
	if !ok || token == "" {
		t.Fatal("Expected session_token in response")
	}
	
	// Verify the JWT is valid
	claims, err := auth.Parse(token)
	if err != nil {
		t.Fatalf("Failed to parse JWT: %v", err)
	}
	
	if claims.Email != email || claims.Role != "guest" {
		t.Fatalf("Invalid claims: email=%s, role=%s", claims.Email, claims.Role)
	}
}

func TestGuestAccess_InvalidEmail_BadRequest(t *testing.T) {
	server, _, _, _, _ := setupTestServer()
	defer server.Close()
	
	tests := []struct {
		name  string
		email string
	}{
		{"empty email", ""},
		{"invalid email", "not-an-email"},
		{"missing @", "testemailcom"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := map[string]string{"email": tt.email}
			postJSON(t, server.URL+"/v1/guest/access/request", requestBody, http.StatusBadRequest)
		})
	}
}

func TestGuestBookings_CreateAndGet_Success(t *testing.T) {
	server, _, _, _, _ := setupTestServer()
	defer server.Close()
	
	// Create booking
	futureTime := time.Now().Add(2 * time.Hour)
	booking := map[string]interface{}{
		"rider_name":   "John Doe",
		"rider_email":  "john@example.com",
		"rider_phone":  "+1234567890",
		"pickup":       "Airport",
		"dropoff":      "Hotel",
		"scheduled_at": futureTime.Format(time.RFC3339),
		"notes":        "Two bags",
		"passengers":   2,
		"luggages":     2,
		"ride_type":    "per_ride",
	}
	
	createResp := postJSON(t, server.URL+"/v1/guest/bookings", booking, http.StatusCreated)
	
	var createResult domain.BookingGuestRes
	json.NewDecoder(createResp.Body).Decode(&createResult)
	createResp.Body.Close()
	
	if createResult.ID == 0 || createResult.ManageToken == "" {
		t.Fatal("Expected booking ID and manage token")
	}
	
	// Get booking by manage token
	getURL := fmt.Sprintf("%s/v1/guest/bookings/%d?manage_token=%s", 
		server.URL, createResult.ID, createResult.ManageToken)
	
	getResp := get(t, getURL, http.StatusOK)
	
	var getResult domain.Booking
	json.NewDecoder(getResp.Body).Decode(&getResult)
	getResp.Body.Close()
	
	if getResult.ID != createResult.ID {
		t.Fatalf("Expected booking ID %d, got %d", createResult.ID, getResult.ID)
	}
	
	if getResult.RiderName != "John Doe" {
		t.Fatalf("Expected rider name 'John Doe', got '%s'", getResult.RiderName)
	}
}

func TestGuestBookings_CreateWithIdempotency_ReturnsExisting(t *testing.T) {
	server, _, _, _, _ := setupTestServer()
	defer server.Close()
	
	idempotencyKey := "test-key-123"
	futureTime := time.Now().Add(2 * time.Hour)
	
	booking := map[string]interface{}{
		"rider_name":   "Jane Doe",
		"rider_email":  "jane@example.com", 
		"rider_phone":  "+1234567890",
		"pickup":       "Home",
		"dropoff":      "Office",
		"scheduled_at": futureTime.Format(time.RFC3339),
		"passengers":   1,
		"luggages":     0,
		"ride_type":    "per_ride",
	}
	
	// First request with idempotency key
	req1, _ := http.NewRequest("POST", server.URL+"/v1/guest/bookings", 
		bytes.NewBuffer(jsonBytes(booking)))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp1.Body.Close()
	
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp1.StatusCode)
	}
	
	var result1 domain.BookingGuestRes
	json.NewDecoder(resp1.Body).Decode(&result1)
	
	// Second request with same idempotency key
	req2, _ := http.NewRequest("POST", server.URL+"/v1/guest/bookings", 
		bytes.NewBuffer(jsonBytes(booking)))
	req2.Header.Set("Content-Type", "application/json") 
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 for idempotent request, got %d", resp2.StatusCode)
	}
	
	var result2 domain.BookingGuestRes
	json.NewDecoder(resp2.Body).Decode(&result2)
	
	if result1.ID != result2.ID {
		t.Fatalf("Expected same booking ID for idempotent requests: %d vs %d", 
			result1.ID, result2.ID)
	}
}

func TestGuestBookings_InvalidInput_BadRequest(t *testing.T) {
	server, _, _, _, _ := setupTestServer()
	defer server.Close()
	
	tests := []struct {
		name    string
		booking map[string]interface{}
	}{
		{
			"missing rider_name",
			map[string]interface{}{
				"rider_email": "test@example.com",
				"rider_phone": "+1234567890", 
				"pickup": "A", "dropoff": "B",
				"scheduled_at": time.Now().Add(time.Hour).Format(time.RFC3339),
				"passengers": 1, "luggages": 0, "ride_type": "per_ride",
			},
		},
		{
			"invalid email",
			map[string]interface{}{
				"rider_name": "Test", "rider_email": "invalid-email",
				"rider_phone": "+1234567890",
				"pickup": "A", "dropoff": "B",
				"scheduled_at": time.Now().Add(time.Hour).Format(time.RFC3339),
				"passengers": 1, "luggages": 0, "ride_type": "per_ride",
			},
		},
		{
			"past scheduled_at",
			map[string]interface{}{
				"rider_name": "Test", "rider_email": "test@example.com",
				"rider_phone": "+1234567890",
				"pickup": "A", "dropoff": "B",
				"scheduled_at": time.Now().Add(-time.Hour).Format(time.RFC3339),
				"passengers": 1, "luggages": 0, "ride_type": "per_ride",
			},
		},
		{
			"invalid passengers",
			map[string]interface{}{
				"rider_name": "Test", "rider_email": "test@example.com",
				"rider_phone": "+1234567890",
				"pickup": "A", "dropoff": "B", 
				"scheduled_at": time.Now().Add(time.Hour).Format(time.RFC3339),
				"passengers": 0, "luggages": 0, "ride_type": "per_ride",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postJSON(t, server.URL+"/v1/guest/bookings", tt.booking, http.StatusBadRequest)
		})
	}
}

func TestGuestBookings_SessionBasedAccess_RequiresAuth(t *testing.T) {
	server, bookingRepo, _, _, _ := setupTestServer()
	defer server.Close()
	
	// Create a booking directly in repo
	futureTime := time.Now().Add(2 * time.Hour)
	booking, _ := bookingRepo.CreateGuest(context.Background(), &domain.BookingGuestReq{
		RiderName: "Test User", RiderEmail: "test@example.com", RiderPhone: "+1234567890",
		Pickup: "A", Dropoff: "B", ScheduledAt: futureTime,
		Passengers: 1, Luggages: 0, RideType: domain.RidePerRide,
	})
	
	// Try to list bookings without session - should fail
	listURL := server.URL + "/v1/guest/bookings"
	get(t, listURL, http.StatusUnauthorized)
	
	// Try to get booking by ID without token or session - should fail  
	getURL := fmt.Sprintf("%s/v1/guest/bookings/%d", server.URL, booking.ID)
	get(t, getURL, http.StatusUnauthorized)
	
	// Create valid guest session
	token, _ := auth.NewGuestSession("test@example.com", 30*time.Minute)
	
	// List with valid session - should work
	req, _ := http.NewRequest("GET", listURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 with valid session, got %d", resp.StatusCode)
	}
	
	var bookings []domain.BookingDTO
	json.NewDecoder(resp.Body).Decode(&bookings)
	
	if len(bookings) != 1 {
		t.Fatalf("Expected 1 booking, got %d", len(bookings))
	}
}

// ---------- Helper Functions ----------

func postJSON(t *testing.T, url string, data interface{}, expectedStatus int) *http.Response {
	t.Helper()
	
	body := jsonBytes(data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	
	if resp.StatusCode != expectedStatus {
		t.Fatalf("POST %s: expected status %d, got %d", url, expectedStatus, resp.StatusCode)
	}
	
	return resp
}

func get(t *testing.T, url string, expectedStatus int) *http.Response {
	t.Helper()
	
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	
	if resp.StatusCode != expectedStatus {
		t.Fatalf("GET %s: expected status %d, got %d", url, expectedStatus, resp.StatusCode)
	}
	
	return resp
}

func jsonBytes(data interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}